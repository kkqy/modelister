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
	return store.WithTx(nil, r.db, func(tx *sql.Tx) error {
		oldIDs, err := modelIDSet(tx, keyID)
		if err != nil {
			return err
		}
		added, removed := diffModelIDs(oldIDs, items)
		if len(added) > 0 || len(removed) > 0 {
			if err := insertChangeEvent(tx, providerID, keyID, added, removed); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`delete from model_cache where provider_key_id=?`, keyID); err != nil {
			return err
		}
		for _, item := range items {
			modelID := strings.TrimSpace(item.ID)
			if modelID == "" {
				continue
			}
			raw := item.RawJSON
			if raw == "" {
				raw = "{}"
			}
			_, err := tx.Exec(
				`insert into model_cache (provider_id, provider_key_id, model_id, owned_by, raw_json) values (?, ?, ?, ?, ?)`,
				providerID, keyID, modelID, item.OwnedBy, raw,
			)
			if err != nil {
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
		select
			id, provider_id, provider_key_id, provider_name, key_name, base_url,
			added_count, removed_count, added_models, removed_models, created_at
		from model_change_events
		where (? = 0 or id < ?)
		order by id desc
		limit ?`, beforeID, beforeID, limit+1)
	if err != nil {
		return ChangeEventsResponse{}, err
	}
	defer rows.Close()

	events := make([]ModelChangeEvent, 0, limit+1)
	for rows.Next() {
		event, err := scanChangeEvent(rows)
		if err != nil {
			return ChangeEventsResponse{}, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return ChangeEventsResponse{}, err
	}

	resp := ChangeEventsResponse{Events: events}
	if len(events) > limit {
		resp.HasMore = true
		resp.Events = events[:limit]
		resp.NextBeforeID = resp.Events[len(resp.Events)-1].ID
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

func insertChangeEvent(tx *sql.Tx, providerID, keyID int64, added, removed []string) error {
	var providerName, baseURL, keyName string
	if err := tx.QueryRow(`
		select p.name, p.base_url, k.name
		from providers p
		join provider_keys k on k.provider_id = p.id
		where p.id=? and k.id=?`, providerID, keyID).Scan(&providerName, &baseURL, &keyName); err != nil {
		return err
	}

	addedJSON, err := marshalModelIDList(added)
	if err != nil {
		return err
	}
	removedJSON, err := marshalModelIDList(removed)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		insert into model_change_events (
			provider_id, provider_key_id, provider_name, key_name, base_url,
			added_count, removed_count, added_models, removed_models
		) values (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		providerID, keyID, providerName, keyName, baseURL,
		len(added), len(removed), addedJSON, removedJSON,
	)
	return err
}

func scanChangeEvent(rows *sql.Rows) (ModelChangeEvent, error) {
	var event ModelChangeEvent
	var addedJSON, removedJSON string
	if err := rows.Scan(
		&event.ID, &event.ProviderID, &event.KeyID, &event.ProviderName, &event.KeyName, &event.BaseURL,
		&event.AddedCount, &event.RemovedCount, &addedJSON, &removedJSON, &event.CreatedAt,
	); err != nil {
		return ModelChangeEvent{}, err
	}

	added, err := parseModelIDList(addedJSON)
	if err != nil {
		return ModelChangeEvent{}, err
	}
	removed, err := parseModelIDList(removedJSON)
	if err != nil {
		return ModelChangeEvent{}, err
	}
	event.AddedModels = added
	event.RemovedModels = removed
	return event, nil
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
