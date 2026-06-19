package models

import (
	"net/http"
	"strconv"
	"strings"

	"modelister/internal/httpx"
)

type Handler struct {
	repo *Repository
	sync *SyncService
}

func NewHandler(repo *Repository, sync *SyncService) *Handler {
	return &Handler{repo: repo, sync: sync}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if h.maybeRefresh(w, r) {
		return
	}
	mode := modeOrDefault(r)
	resp, ok := h.listResponse(w, mode, "")
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "搜索关键词不能为空")
		return
	}
	if h.maybeRefresh(w, r) {
		return
	}
	mode := modeOrDefault(r)
	resp, ok := h.listResponse(w, mode, query)
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SyncAll(w http.ResponseWriter, r *http.Request) {
	if h.sync == nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "同步服务未初始化")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": h.sync.SyncAll()})
}

func (h *Handler) SyncProvider(w http.ResponseWriter, r *http.Request) {
	if h.sync == nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "同步服务未初始化")
		return
	}
	providerID, ok := pathID(w, r, "provider_id")
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": h.sync.SyncProvider(providerID)})
}

func (h *Handler) SyncKey(w http.ResponseWriter, r *http.Request) {
	if h.sync == nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "同步服务未初始化")
		return
	}
	providerID, ok := pathID(w, r, "provider_id")
	if !ok {
		return
	}
	keyID, ok := pathID(w, r, "key_id")
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": []SyncResult{h.sync.SyncKey(providerID, keyID)}})
}

func (h *Handler) maybeRefresh(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Query().Get("refresh") != "true" {
		return false
	}
	if h.sync == nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "同步服务未初始化")
		return true
	}
	h.sync.SyncAll()
	return false
}

func (h *Handler) listResponse(w http.ResponseWriter, mode, query string) (ListResponse, bool) {
	var providers []ProviderModels
	var err error
	switch mode {
	case "by_key":
		providers, err = h.repo.ListByKey(query)
	case "merged":
		providers, err = h.repo.ListMerged(query)
	default:
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "mode 只能是 by_key 或 merged")
		return ListResponse{}, false
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "读取模型失败")
		return ListResponse{}, false
	}
	return ListResponse{Mode: mode, Query: query, Providers: providers}, true
}

func modeOrDefault(r *http.Request) string {
	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	if mode == "" {
		return "by_key"
	}
	return mode
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	value := r.PathValue(name)
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "路径 ID 无效")
		return 0, false
	}
	return id, true
}
