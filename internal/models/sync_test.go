package models

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"modelister/internal/providers"
	"modelister/internal/store"
)

func TestSyncKeyCachesModelsAndClearsError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gpt-4o","object":"model","owned_by":"openai"}]}`))
	}))
	defer upstream.Close()

	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	providerRepo := providers.NewRepository(db)
	p, err := providerRepo.CreateProvider(providers.CreateProviderRequest{Name: "P", BaseURL: upstream.URL, Enabled: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	k, err := providerRepo.CreateKey(p.ID, providers.CreateKeyRequest{Name: "K", APIKey: "sk-test", Enabled: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}

	modelRepo := NewRepository(db)
	service := NewSyncService(providerRepo, modelRepo, upstream.Client())
	result := service.SyncKey(p.ID, k.ID)

	if !result.OK {
		t.Fatalf("sync failed: %+v", result)
	}
	groups, err := modelRepo.ListByKey("")
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || len(groups[0].Keys) != 1 || len(groups[0].Keys[0].Models) != 1 {
		t.Fatalf("unexpected groups: %+v", groups)
	}
	if groups[0].Keys[0].Models[0].ID != "gpt-4o" {
		t.Fatalf("unexpected model: %+v", groups[0].Keys[0].Models[0])
	}
}

func TestSyncKeyWritesChangeEventsOnlyWhenModelIDsChange(t *testing.T) {
	payload := `{"object":"list","data":[{"id":"alpha","object":"model"},{"id":"beta","object":"model"}]}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer upstream.Close()

	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	providerRepo := providers.NewRepository(db)
	p, err := providerRepo.CreateProvider(providers.CreateProviderRequest{Name: "P", BaseURL: upstream.URL, Enabled: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	k, err := providerRepo.CreateKey(p.ID, providers.CreateKeyRequest{Name: "K", APIKey: "sk-test", Enabled: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}

	modelRepo := NewRepository(db)
	service := NewSyncService(providerRepo, modelRepo, upstream.Client())

	if result := service.SyncKey(p.ID, k.ID); !result.OK {
		t.Fatalf("initial sync failed: %+v", result)
	}
	resp, err := modelRepo.ListChangeEvents(20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Events) != 1 {
		t.Fatalf("event count after initial sync = %d, want 1", len(resp.Events))
	}
	if got := resp.Events[0].AddedModels; len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("initial added models = %+v", got)
	}
	if resp.Events[0].RemovedCount != 0 {
		t.Fatalf("initial removed count = %d, want 0", resp.Events[0].RemovedCount)
	}

	payload = `{"object":"list","data":[{"id":"beta","object":"model"},{"id":"alpha","object":"model"}]}`
	if result := service.SyncKey(p.ID, k.ID); !result.OK {
		t.Fatalf("same-list sync failed: %+v", result)
	}
	resp, err = modelRepo.ListChangeEvents(20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Events) != 1 {
		t.Fatalf("event count after same-list sync = %d, want 1", len(resp.Events))
	}

	payload = `{"object":"list","data":[{"id":"beta","object":"model"},{"id":"gamma","object":"model"}]}`
	if result := service.SyncKey(p.ID, k.ID); !result.OK {
		t.Fatalf("changed sync failed: %+v", result)
	}
	resp, err = modelRepo.ListChangeEvents(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasMore || resp.NextBeforeID == 0 || len(resp.Events) != 1 {
		t.Fatalf("paged response = %+v, want one item with more pages", resp)
	}
	latest := resp.Events[0]
	if latest.ProviderName != "P" || latest.KeyName != "K" || latest.BaseURL != upstream.URL {
		t.Fatalf("snapshot fields = %+v", latest)
	}
	if latest.AddedCount != 1 || latest.RemovedCount != 1 {
		t.Fatalf("latest counts = +%d -%d, want +1 -1", latest.AddedCount, latest.RemovedCount)
	}
	if len(latest.AddedModels) != 1 || latest.AddedModels[0] != "gamma" {
		t.Fatalf("latest added models = %+v", latest.AddedModels)
	}
	if len(latest.RemovedModels) != 1 || latest.RemovedModels[0] != "alpha" {
		t.Fatalf("latest removed models = %+v", latest.RemovedModels)
	}

	next, err := modelRepo.ListChangeEvents(1, resp.NextBeforeID)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Events) != 1 || next.Events[0].ID >= resp.NextBeforeID {
		t.Fatalf("next page = %+v, before_id = %d", next, resp.NextBeforeID)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
