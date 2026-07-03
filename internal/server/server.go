package server

import (
	"net/http"

	"modelister/internal/auth"
	"modelister/internal/httpx"
	"modelister/internal/models"
	"modelister/internal/providers"
)

func New(authManager *auth.Manager, providerHandler *providers.Handler, modelHandler *models.Handler, staticHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /api/auth/login", authManager.Login)
	mux.HandleFunc("GET /api/auth/me", authManager.Me)
	mux.HandleFunc("POST /api/auth/logout", authManager.Logout)

	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/providers", providerHandler.ListProviders)
	protected.HandleFunc("POST /api/providers", providerHandler.CreateProvider)
	protected.HandleFunc("PUT /api/providers/{provider_id}", providerHandler.UpdateProvider)
	protected.HandleFunc("DELETE /api/providers/{provider_id}", providerHandler.DeleteProvider)
	protected.HandleFunc("GET /api/providers/{provider_id}/keys", providerHandler.ListKeys)
	protected.HandleFunc("POST /api/providers/{provider_id}/keys", providerHandler.CreateKey)
	protected.HandleFunc("PUT /api/providers/{provider_id}/keys/{key_id}", providerHandler.UpdateKey)
	protected.HandleFunc("DELETE /api/providers/{provider_id}/keys/{key_id}", providerHandler.DeleteKey)
	protected.HandleFunc("POST /api/providers/{provider_id}/sync", modelHandler.SyncProvider)
	protected.HandleFunc("POST /api/providers/{provider_id}/keys/{key_id}/sync", modelHandler.SyncKey)
	protected.HandleFunc("GET /api/models", modelHandler.List)
	protected.HandleFunc("GET /api/models/search", modelHandler.Search)
	protected.HandleFunc("POST /api/models/sync", modelHandler.SyncAll)
	protected.HandleFunc("GET /api/model-changes", modelHandler.ListChangeEvents)

	mux.Handle("/api/", authManager.RequireAuth(protected))

	// 静态前端：catch-all，/healthz 与 /api/ 因更具体而优先匹配。
	// 测试中可传 nil 以跳过静态资源挂载。
	if staticHandler != nil {
		mux.Handle("/", staticHandler)
	}
	return mux
}
