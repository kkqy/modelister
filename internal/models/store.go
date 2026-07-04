package models

import (
	"database/sql"
	"encoding/json"
	"sort"
	"strings"

	"modelister/internal/store"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ReplaceKeyModels(providerID, keyID int64, items []Model) error {
	synced := SyncedKeyModels{ProviderID: providerID, KeyID: keyID, Models: items}
	return r.ReplaceSyncedKeyModels("key", []SyncedKeyModels{synced})
}

func (r *Repository) ReplaceSyncedKeyModels(scope string, items []SyncedKeyModels) error {
	if len(items) == 0 {
		return nil
	}
	return store.WithTx(nil, r.db, func(tx *sql.Tx) error {
		var changes []changeToInsert
		for _, item := range items {
			if err := fillSyncedKeySnapshot(tx, &item); err != nil {
				return err
			}
			oldIDs, err := modelIDSet(tx, item.KeyID)
			if err != nil {
				return err
			}
			added, removed := diffModelIDs(oldIDs, item.Models)
			if len(added) > 0 || len(removed) > 0 {
				changes = append(changes, changeToInsert{item: item, added: added, removed: removed})
			}

			if _, err := tx.Exec(`delete from model_cache where provider_key_id=?`, item.KeyID); err != nil {
				return err
			}
			for _, model := range item.Models {
				modelID := strings.TrimSpace(model.ID)
				if modelID == "" {
					continue
				}
				raw := model.RawJSON
				if raw == "" {
					raw = "{}"
				}
				_, err := tx.Exec(
					`insert into model_cache (provider_id, provider_key_id, model_id, owned_by, raw_json) values (?, ?, ?, ?, ?)`,
					item.ProviderID, item.KeyID, modelID, model.OwnedBy, raw,
				)
				if err != nil {
					return err
				}
			}
			if _, err := tx.Exec(
				`update provider_keys set last_sync_at=strftime('%Y-%m-%dT%H:%M:%fZ','now'), last_sync_error='', updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') where provider_id=? and id=?`,
				item.ProviderID, item.KeyID,
			); err != nil {
				return err
			}
		}

		if len(changes) == 0 {
			return nil
		}
		runID, createdAt, err := insertChangeRun(tx, scope)
		if err != nil {
			return err
		}
		for _, change := range changes {
			if err := insertChangeEvent(tx, runID, createdAt, change); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) SetKeySyncSuccess(providerID, keyID int64) error {
	_, err := r.db.Exec(
		`update provider_keys set last_sync_at=strftime('%Y-%m-%dT%H:%M:%fZ','now'), last_sync_error='', updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') where provider_id=? and id=?`,
		providerID, keyID,
	)
	return err
}

func (r *Repository) SetKeySyncError(providerID, keyID int64, message string) error {
	_, err := r.db.Exec(
		`update provider_keys set last_sync_error=?, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') where provider_id=? and id=?`,
		message, providerID, keyID,
	)
	return err
}

func (r *Repository) ListChangeEvents(limit int, beforeID int64) (ChangeEventsResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(`
		select id, created_at
		from model_change_runs r
		where (? = 0 or r.id < ?)
		  and exists (select 1 from model_change_events e where e.run_id = r.id)
		order by r.id desc
		limit ?`, beforeID, beforeID, limit+1)
	if err != nil {
		return ChangeEventsResponse{}, err
	}
	defer rows.Close()

	groups := make([]ModelChangeGroup, 0, limit+1)
	for rows.Next() {
		var group ModelChangeGroup
		if err := rows.Scan(&group.ID, &group.CreatedAt); err != nil {
			return ChangeEventsResponse{}, err
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return ChangeEventsResponse{}, err
	}

	resp := ChangeEventsResponse{Groups: groups}
	if len(groups) > limit {
		resp.HasMore = true
		resp.Groups = groups[:limit]
		resp.NextBeforeID = resp.Groups[len(resp.Groups)-1].ID
	}
	if len(resp.Groups) == 0 {
		return resp, nil
	}

	if err := r.loadChangeGroupEvents(&resp); err != nil {
		return ChangeEventsResponse{}, err
	}
	return resp, nil
}

func (r *Repository) ListByKey(query string) ([]ProviderModels, error) {
	rows, err := r.db.Query(`
		select
			p.id, p.name, p.base_url, p.note,
			k.id, k.name, k.note, k.last_sync_at, k.last_sync_error,
			m.model_id, m.owned_by, m.raw_json
		from providers p
		join provider_keys k on k.provider_id = p.id
		join model_cache m on m.provider_key_id = k.id
		where p.enabled=1 and k.enabled=1
		  and (? = '' or lower(m.model_id) like '%' || lower(?) || '%')
		order by p.id, k.id, m.model_id`, query, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	providerIndex := map[int64]int{}
	keyIndex := map[int64]int{}
	var result []ProviderModels
	for rows.Next() {
		var providerID, keyID int64
		var providerName, baseURL, providerNote string
		var keyName, keyNote, lastSyncAt, lastSyncError string
		var model Model
		if err := rows.Scan(&providerID, &providerName, &baseURL, &providerNote, &keyID, &keyName, &keyNote, &lastSyncAt, &lastSyncError, &model.ID, &model.OwnedBy, &model.RawJSON); err != nil {
			return nil, err
		}
		pi, ok := providerIndex[providerID]
		if !ok {
			result = append(result, ProviderModels{ID: providerID, Name: providerName, BaseURL: baseURL, Note: providerNote})
			pi = len(result) - 1
			providerIndex[providerID] = pi
		}
		compoundKey := providerID<<32 | keyID
		ki, ok := keyIndex[compoundKey]
		if !ok {
			result[pi].Keys = append(result[pi].Keys, KeyModels{ID: keyID, Name: keyName, Note: keyNote, LastSyncAt: lastSyncAt, LastSyncError: lastSyncError})
			ki = len(result[pi].Keys) - 1
			keyIndex[compoundKey] = ki
		}
		result[pi].Keys[ki].Models = append(result[pi].Keys[ki].Models, model)
	}
	return result, rows.Err()
}

func (r *Repository) ListMerged(query string) ([]ProviderModels, error) {
	byKey, err := r.ListByKey(query)
	if err != nil {
		return nil, err
	}
	for pi := range byKey {
		seen := map[string]Model{}
		for _, key := range byKey[pi].Keys {
			for _, model := range key.Models {
				if _, ok := seen[model.ID]; !ok {
					seen[model.ID] = model
				}
			}
		}
		models := make([]Model, 0, len(seen))
		for _, key := range byKey[pi].Keys {
			for _, model := range key.Models {
				if kept, ok := seen[model.ID]; ok {
					models = append(models, kept)
					delete(seen, model.ID)
				}
			}
		}
		byKey[pi].Keys = nil
		byKey[pi].Models = models
	}
	return byKey, nil
}

func modelIDSet(tx *sql.Tx, keyID int64) (map[string]struct{}, error) {
	rows, err := tx.Query(`select model_id from model_cache where provider_key_id=?`, keyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		id = strings.TrimSpace(id)
		if id != "" {
			ids[id] = struct{}{}
		}
	}
	return ids, rows.Err()
}

func diffModelIDs(oldIDs map[string]struct{}, items []Model) ([]string, []string) {
	newIDs := map[string]struct{}{}
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id != "" {
			newIDs[id] = struct{}{}
		}
	}

	var added []string
	for id := range newIDs {
		if _, ok := oldIDs[id]; !ok {
			added = append(added, id)
		}
	}

	var removed []string
	for id := range oldIDs {
		if _, ok := newIDs[id]; !ok {
			removed = append(removed, id)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

type changeToInsert struct {
	item    SyncedKeyModels
	added   []string
	removed []string
}

func fillSyncedKeySnapshot(tx *sql.Tx, item *SyncedKeyModels) error {
	if item.ProviderName != "" && item.BaseURL != "" && item.KeyName != "" {
		return nil
	}
	return tx.QueryRow(`
		select p.name, p.base_url, k.name
		from providers p
		join provider_keys k on k.provider_id = p.id
		where p.id=? and k.id=?`, item.ProviderID, item.KeyID).Scan(
		&item.ProviderName, &item.BaseURL, &item.KeyName,
	)
}

func insertChangeRun(tx *sql.Tx, scope string) (int64, string, error) {
	res, err := tx.Exec(`insert into model_change_runs (scope) values (?)`, scope)
	if err != nil {
		return 0, "", err
	}
	runID, err := res.LastInsertId()
	if err != nil {
		return 0, "", err
	}
	var createdAt string
	if err := tx.QueryRow(`select created_at from model_change_runs where id=?`, runID).Scan(&createdAt); err != nil {
		return 0, "", err
	}
	return runID, createdAt, nil
}

func insertChangeEvent(tx *sql.Tx, runID int64, createdAt string, change changeToInsert) error {
	addedJSON, err := marshalModelIDList(change.added)
	if err != nil {
		return err
	}
	removedJSON, err := marshalModelIDList(change.removed)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		insert into model_change_events (
			run_id, provider_id, provider_key_id, provider_name, key_name, base_url,
			added_count, removed_count, added_models, removed_models, created_at
		) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		runID, change.item.ProviderID, change.item.KeyID,
		change.item.ProviderName, change.item.KeyName, change.item.BaseURL,
		len(change.added), len(change.removed), addedJSON, removedJSON, createdAt,
	)
	return err
}

func (r *Repository) loadChangeGroupEvents(resp *ChangeEventsResponse) error {
	groupIndex := map[int64]int{}
	runIDs := make([]int64, 0, len(resp.Groups))
	for i := range resp.Groups {
		groupIndex[resp.Groups[i].ID] = i
		runIDs = append(runIDs, resp.Groups[i].ID)
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(runIDs)), ",")
	args := make([]any, 0, len(runIDs))
	for _, id := range runIDs {
		args = append(args, id)
	}
	rows, err := r.db.Query(`
		select
			run_id, id, provider_id, provider_key_id, provider_name, key_name, base_url,
			added_count, removed_count, added_models, removed_models, created_at
		from model_change_events
		where run_id in (`+placeholders+`)
		order by run_id desc, provider_id, provider_key_id, id desc`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	providerIndex := map[int64]map[int64]int{}
	for rows.Next() {
		runID, event, err := scanChangeEvent(rows)
		if err != nil {
			return err
		}
		gi, ok := groupIndex[runID]
		if !ok {
			continue
		}
		group := &resp.Groups[gi]
		group.AddedCount += event.AddedCount
		group.RemovedCount += event.RemovedCount

		byProvider, ok := providerIndex[runID]
		if !ok {
			byProvider = map[int64]int{}
			providerIndex[runID] = byProvider
		}
		pi, ok := byProvider[event.ProviderID]
		if !ok {
			group.Providers = append(group.Providers, ModelChangeProviderGroup{
				ProviderID:   event.ProviderID,
				ProviderName: event.ProviderName,
				BaseURL:      event.BaseURL,
			})
			pi = len(group.Providers) - 1
			byProvider[event.ProviderID] = pi
		}
		provider := &group.Providers[pi]
		provider.AddedCount += event.AddedCount
		provider.RemovedCount += event.RemovedCount
		provider.Keys = append(provider.Keys, ModelChangeKeyEvent{
			ID:            event.ID,
			KeyID:         event.KeyID,
			KeyName:       event.KeyName,
			AddedCount:    event.AddedCount,
			RemovedCount:  event.RemovedCount,
			AddedModels:   event.AddedModels,
			RemovedModels: event.RemovedModels,
			CreatedAt:     event.CreatedAt,
		})
	}
	return rows.Err()
}

func scanChangeEvent(rows *sql.Rows) (int64, ModelChangeEvent, error) {
	var runID int64
	var event ModelChangeEvent
	var addedJSON, removedJSON string
	if err := rows.Scan(
		&runID,
		&event.ID, &event.ProviderID, &event.KeyID, &event.ProviderName, &event.KeyName, &event.BaseURL,
		&event.AddedCount, &event.RemovedCount, &addedJSON, &removedJSON, &event.CreatedAt,
	); err != nil {
		return 0, ModelChangeEvent{}, err
	}

	added, err := parseModelIDList(addedJSON)
	if err != nil {
		return 0, ModelChangeEvent{}, err
	}
	removed, err := parseModelIDList(removedJSON)
	if err != nil {
		return 0, ModelChangeEvent{}, err
	}
	event.AddedModels = added
	event.RemovedModels = removed
	return runID, event, nil
}

func marshalModelIDList(items []string) (string, error) {
	if items == nil {
		items = []string{}
	}
	data, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseModelIDList(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}, nil
	}
	var items []string
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil, err
	}
	if items == nil {
		return []string{}, nil
	}
	return items, nil
}
