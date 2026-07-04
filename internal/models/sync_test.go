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
	if len(resp.Groups) != 1 || len(resp.Groups[0].Providers) != 1 || len(resp.Groups[0].Providers[0].Keys) != 1 {
		t.Fatalf("groups after initial sync = %+v, want one key event", resp.Groups)
	}
	initial := resp.Groups[0].Providers[0].Keys[0]
	if got := initial.AddedModels; len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("initial added models = %+v", got)
	}
	if initial.RemovedCount != 0 {
		t.Fatalf("initial removed count = %d, want 0", initial.RemovedCount)
	}

	payload = `{"object":"list","data":[{"id":"beta","object":"model"},{"id":"alpha","object":"model"}]}`
	if result := service.SyncKey(p.ID, k.ID); !result.OK {
		t.Fatalf("same-list sync failed: %+v", result)
	}
	resp, err = modelRepo.ListChangeEvents(20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Groups) != 1 {
		t.Fatalf("group count after same-list sync = %d, want 1", len(resp.Groups))
	}

	payload = `{"object":"list","data":[{"id":"beta","object":"model"},{"id":"gamma","object":"model"}]}`
	if result := service.SyncKey(p.ID, k.ID); !result.OK {
		t.Fatalf("changed sync failed: %+v", result)
	}
	resp, err = modelRepo.ListChangeEvents(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasMore || resp.NextBeforeID == 0 || len(resp.Groups) != 1 {
		t.Fatalf("paged response = %+v, want one item with more pages", resp)
	}
	latestGroup := resp.Groups[0]
	if latestGroup.AddedCount != 1 || latestGroup.RemovedCount != 1 {
		t.Fatalf("latest group counts = +%d -%d, want +1 -1", latestGroup.AddedCount, latestGroup.RemovedCount)
	}
	if len(latestGroup.Providers) != 1 || len(latestGroup.Providers[0].Keys) != 1 {
		t.Fatalf("latest group shape = %+v", latestGroup)
	}
	latestProvider := latestGroup.Providers[0]
	latest := latestProvider.Keys[0]
	if latestProvider.ProviderName != "P" || latest.KeyName != "K" || latestProvider.BaseURL != upstream.URL {
		t.Fatalf("snapshot fields = provider %+v key %+v", latestProvider, latest)
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
	if len(next.Groups) != 1 || next.Groups[0].ID >= resp.NextBeforeID {
		t.Fatalf("next page = %+v, before_id = %d", next, resp.NextBeforeID)
	}
}

func TestSyncAllWritesOneChangeGroupForMultipleChangedKeys(t *testing.T) {
	upstreamA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"alpha","object":"model"}]}`))
	}))
	defer upstreamA.Close()
	upstreamB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"beta","object":"model"}]}`))
	}))
	defer upstreamB.Close()

	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	providerRepo := providers.NewRepository(db)
	p1, err := providerRepo.CreateProvider(providers.CreateProviderRequest{Name: "P1", BaseURL: upstreamA.URL, Enabled: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := providerRepo.CreateKey(p1.ID, providers.CreateKeyRequest{Name: "K1", APIKey: "sk-1", Enabled: boolPtr(true)}); err != nil {
		t.Fatal(err)
	}
	p2, err := providerRepo.CreateProvider(providers.CreateProviderRequest{Name: "P2", BaseURL: upstreamB.URL, Enabled: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := providerRepo.CreateKey(p2.ID, providers.CreateKeyRequest{Name: "K2", APIKey: "sk-2", Enabled: boolPtr(true)}); err != nil {
		t.Fatal(err)
	}

	modelRepo := NewRepository(db)
	service := NewSyncService(providerRepo, modelRepo, http.DefaultClient)
	results := service.SyncAll()
	if len(results) != 2 || !results[0].OK || !results[1].OK {
		t.Fatalf("sync all results = %+v, want two successful results", results)
	}

	resp, err := modelRepo.ListChangeEvents(20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Groups) != 1 {
		t.Fatalf("group count = %d, want 1: %+v", len(resp.Groups), resp.Groups)
	}
	group := resp.Groups[0]
	if group.AddedCount != 2 || group.RemovedCount != 0 {
		t.Fatalf("group counts = +%d -%d, want +2 -0", group.AddedCount, group.RemovedCount)
	}
	if len(group.Providers) != 2 {
		t.Fatalf("provider group count = %d, want 2: %+v", len(group.Providers), group.Providers)
	}
	if len(group.Providers[0].Keys) != 1 || len(group.Providers[1].Keys) != 1 {
		t.Fatalf("key groups = %+v, want one key per provider", group.Providers)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
