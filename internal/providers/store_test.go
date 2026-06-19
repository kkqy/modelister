package providers

import (
	"testing"

	"modelister/internal/store"
)

func TestRepositoryCreatesProviderWithNote(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewRepository(db)
	p, err := repo.CreateProvider(CreateProviderRequest{
		Name:    "OpenAI",
		BaseURL: "https://api.openai.com/",
		Note:    "主供应商",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if p.BaseURL != "https://api.openai.com" {
		t.Fatalf("base_url normalized to %q", p.BaseURL)
	}
	if p.Note != "主供应商" {
		t.Fatalf("note = %q", p.Note)
	}
}

func TestRepositoryMasksKey(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewRepository(db)
	p, err := repo.CreateProvider(CreateProviderRequest{Name: "P", BaseURL: "https://example.com", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	k, err := repo.CreateKey(p.ID, CreateKeyRequest{Name: "K", APIKey: "sk-abcdef123456", Note: "备用", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if k.APIKey != "" {
		t.Fatalf("api_key should not be returned, got %q", k.APIKey)
	}
	if k.APIKeyMasked != "sk-...3456" {
		t.Fatalf("masked key = %q", k.APIKeyMasked)
	}
}
