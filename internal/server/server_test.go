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
	handler := New(auth.NewManager("admin", "secret", "session-secret"), providers.NewHandler(providerRepo), models.NewHandler(modelRepo, nil), nil)

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

func TestStaticHandlerServesRootButNotApiOrHealth(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	providerRepo := providers.NewRepository(db)
	modelRepo := models.NewRepository(db)

	staticHits := 0
	staticHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		staticHits++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("frontend"))
	})
	handler := New(auth.NewManager("admin", "secret", "session-secret"), providers.NewHandler(providerRepo), models.NewHandler(modelRepo, nil), staticHandler)

	// 根路径与未知路径走静态前端。
	rootRec := httptest.NewRecorder()
	handler.ServeHTTP(rootRec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rootRec.Code != http.StatusOK || rootRec.Body.String() != "frontend" {
		t.Fatalf("root: status=%d body=%q", rootRec.Code, rootRec.Body.String())
	}

	spaRec := httptest.NewRecorder()
	handler.ServeHTTP(spaRec, httptest.NewRequest(http.MethodGet, "/models", nil))
	if spaRec.Code != http.StatusOK || spaRec.Body.String() != "frontend" {
		t.Fatalf("spa: status=%d body=%q", spaRec.Code, spaRec.Body.String())
	}

	// /healthz 与 /api/ 更具体，不应落到静态处理器。
	healthRec := httptest.NewRecorder()
	handler.ServeHTTP(healthRec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if healthRec.Code != http.StatusOK {
		t.Fatalf("health status = %d", healthRec.Code)
	}

	apiRec := httptest.NewRecorder()
	handler.ServeHTTP(apiRec, httptest.NewRequest(http.MethodGet, "/api/providers", nil))
	if apiRec.Code != http.StatusUnauthorized {
		t.Fatalf("api status = %d", apiRec.Code)
	}

	if staticHits != 2 {
		t.Fatalf("static handler hit %d times, want 2", staticHits)
	}
}
