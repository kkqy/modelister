package models

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modelister/internal/providers"
	"modelister/internal/store"
)

func TestSearchDefaultsToByKey(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	providerRepo := providers.NewRepository(db)
	p, err := providerRepo.CreateProvider(providers.CreateProviderRequest{Name: "P", BaseURL: "https://example.com", Note: "供应商备注", Enabled: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	k1, _ := providerRepo.CreateKey(p.ID, providers.CreateKeyRequest{Name: "K1", APIKey: "sk-1", Note: "第一个", Enabled: boolPtr(true)})
	k2, _ := providerRepo.CreateKey(p.ID, providers.CreateKeyRequest{Name: "K2", APIKey: "sk-2", Note: "第二个", Enabled: boolPtr(true)})

	repo := NewRepository(db)
	if err := repo.ReplaceKeyModels(p.ID, k1.ID, []Model{{ID: "gpt-4o"}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceKeyModels(p.ID, k2.ID, []Model{{ID: "gpt-4o-mini"}}); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(repo, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/models/search?q=GPT", nil)
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"mode":"by_key"`) || !strings.Contains(body, `"name":"K1"`) || !strings.Contains(body, `"name":"K2"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}
