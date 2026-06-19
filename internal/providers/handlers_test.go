package providers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modelister/internal/store"
)

func TestCreateProviderHandler(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	h := NewHandler(NewRepository(db))
	req := httptest.NewRequest(http.MethodPost, "/api/providers", strings.NewReader(`{"name":"P","base_url":"https://example.com","note":"备注","enabled":true}`))
	rec := httptest.NewRecorder()

	h.CreateProvider(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"note":"备注"`) {
		t.Fatalf("response missing note: %s", rec.Body.String())
	}
}
