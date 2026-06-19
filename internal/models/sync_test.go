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

func boolPtr(v bool) *bool {
	return &v
}
