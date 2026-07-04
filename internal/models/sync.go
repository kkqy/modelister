package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"modelister/internal/providers"
)

type SyncService struct {
	providerRepo *providers.Repository
	modelRepo    *Repository
	client       *http.Client
}

func NewSyncService(providerRepo *providers.Repository, modelRepo *Repository, client *http.Client) *SyncService {
	if client == nil {
		client = http.DefaultClient
	}
	return &SyncService{providerRepo: providerRepo, modelRepo: modelRepo, client: client}
}

func (s *SyncService) SyncKey(providerID, keyID int64) SyncResult {
	p, err := s.providerRepo.GetProvider(providerID)
	if err != nil {
		return SyncResult{ProviderID: providerID, KeyID: keyID, OK: false, Error: err.Error()}
	}
	k, err := s.providerRepo.GetKeyWithSecret(providerID, keyID)
	if err != nil {
		return SyncResult{ProviderID: providerID, KeyID: keyID, OK: false, Error: err.Error()}
	}
	if !p.Enabled || !k.Enabled {
		msg := "provider or key is disabled"
		_ = s.modelRepo.SetKeySyncError(providerID, keyID, msg)
		return SyncResult{ProviderID: providerID, KeyID: keyID, OK: false, Error: msg}
	}
	results := s.syncEnabledKeys("key", []providers.ProviderWithKey{{Provider: p, Key: k}})
	return results[0]
}

func (s *SyncService) SyncProvider(providerID int64) []SyncResult {
	items, err := s.providerRepo.ListEnabledProviderKeys()
	if err != nil {
		return []SyncResult{{ProviderID: providerID, OK: false, Error: err.Error()}}
	}
	var targets []providers.ProviderWithKey
	for _, item := range items {
		if item.Provider.ID == providerID {
			targets = append(targets, item)
		}
	}
	return s.syncEnabledKeys("provider", targets)
}

func (s *SyncService) SyncAll() []SyncResult {
	items, err := s.providerRepo.ListEnabledProviderKeys()
	if err != nil {
		return []SyncResult{{OK: false, Error: err.Error()}}
	}
	return s.syncEnabledKeys("all", items)
}

func (s *SyncService) syncEnabledKeys(scope string, targets []providers.ProviderWithKey) []SyncResult {
	results := make([]SyncResult, 0, len(targets))
	synced := make([]SyncedKeyModels, 0, len(targets))

	for _, target := range targets {
		result := SyncResult{ProviderID: target.Provider.ID, KeyID: target.Key.ID}
		items, err := s.fetchModels(target.Provider.BaseURL, target.Key.APIKey)
		if err != nil {
			_ = s.modelRepo.SetKeySyncError(target.Provider.ID, target.Key.ID, err.Error())
			result.OK = false
			result.Error = err.Error()
			results = append(results, result)
			continue
		}
		result.OK = true
		result.Count = len(items)
		results = append(results, result)
		synced = append(synced, SyncedKeyModels{
			ProviderID:   target.Provider.ID,
			ProviderName: target.Provider.Name,
			BaseURL:      target.Provider.BaseURL,
			KeyID:        target.Key.ID,
			KeyName:      target.Key.Name,
			Models:       items,
		})
	}

	if err := s.modelRepo.ReplaceSyncedKeyModels(scope, synced); err != nil {
		for i := range results {
			if !results[i].OK {
				continue
			}
			_ = s.modelRepo.SetKeySyncError(results[i].ProviderID, results[i].KeyID, err.Error())
			results[i].OK = false
			results[i].Error = err.Error()
			results[i].Count = 0
		}
	}
	return results
}

func (s *SyncService) fetchModels(baseURL, apiKey string) ([]Model, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream_error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream_error: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("upstream_error: invalid response: %w", err)
	}
	items := make([]Model, 0, len(payload.Data))
	for _, raw := range payload.Data {
		var item struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		item.ID = strings.TrimSpace(item.ID)
		if item.ID == "" {
			continue
		}
		items = append(items, Model{ID: item.ID, OwnedBy: item.OwnedBy, RawJSON: string(raw)})
	}
	return items, nil
}
