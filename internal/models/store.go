package models

import (
	"database/sql"
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
		if _, err := tx.Exec(`delete from model_cache where provider_key_id=?`, keyID); err != nil {
			return err
		}
		for _, item := range items {
			if strings.TrimSpace(item.ID) == "" {
				continue
			}
			raw := item.RawJSON
			if raw == "" {
				raw = "{}"
			}
			_, err := tx.Exec(
				`insert into model_cache (provider_id, provider_key_id, model_id, owned_by, raw_json) values (?, ?, ?, ?, ?)`,
				providerID, keyID, item.ID, item.OwnedBy, raw,
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
