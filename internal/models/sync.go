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
	items, err := s.fetchModels(p.BaseURL, k.APIKey)
	if err != nil {
		_ = s.modelRepo.SetKeySyncError(providerID, keyID, err.Error())
		return SyncResult{ProviderID: providerID, KeyID: keyID, OK: false, Error: err.Error()}
	}
	if err := s.modelRepo.ReplaceKeyModels(providerID, keyID, items); err != nil {
		_ = s.modelRepo.SetKeySyncError(providerID, keyID, err.Error())
		return SyncResult{ProviderID: providerID, KeyID: keyID, OK: false, Error: err.Error()}
	}
	if err := s.modelRepo.SetKeySyncSuccess(providerID, keyID); err != nil {
		return SyncResult{ProviderID: providerID, KeyID: keyID, OK: false, Error: err.Error()}
	}
	return SyncResult{ProviderID: providerID, KeyID: keyID, OK: true, Count: len(items)}
}

func (s *SyncService) SyncProvider(providerID int64) []SyncResult {
	items, err := s.providerRepo.ListEnabledProviderKeys()
	if err != nil {
		return []SyncResult{{ProviderID: providerID, OK: false, Error: err.Error()}}
	}
	var results []SyncResult
	for _, item := range items {
		if item.Provider.ID == providerID {
			results = append(results, s.SyncKey(item.Provider.ID, item.Key.ID))
		}
	}
	return results
}

func (s *SyncService) SyncAll() []SyncResult {
	items, err := s.providerRepo.ListEnabledProviderKeys()
	if err != nil {
		return []SyncResult{{OK: false, Error: err.Error()}}
	}
	results := make([]SyncResult, 0, len(items))
	for _, item := range items {
		results = append(results, s.SyncKey(item.Provider.ID, item.Key.ID))
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
