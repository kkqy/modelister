package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"modelister/internal/auth"
	"modelister/internal/models"
	"modelister/internal/providers"
	"modelister/internal/store"
)

func TestHealthIsPublicAndProvidersAreProtected(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	providerRepo := providers.NewRepository(db)
	modelRepo := models.NewRepository(db)
	handler := New(auth.NewManager("admin", "secret", "session-secret"), providers.NewHandler(providerRepo), models.NewHandler(modelRepo, nil))

	healthRec := httptest.NewRecorder()
	handler.ServeHTTP(healthRec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if healthRec.Code != http.StatusOK {
		t.Fatalf("health status = %d", healthRec.Code)
	}

	protectedRec := httptest.NewRecorder()
	handler.ServeHTTP(protectedRec, httptest.NewRequest(http.MethodGet, "/api/providers", nil))
	if protectedRec.Code != http.StatusUnauthorized {
		t.Fatalf("protected status = %d", protectedRec.Code)
	}
}
