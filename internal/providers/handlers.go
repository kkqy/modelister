package providers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"modelister/internal/httpx"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var req CreateProviderRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p, err := h.repo.CreateProvider(req)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListProviders()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "读取供应商失败")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"providers": items})
}

func (h *Handler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "provider_id")
	if !ok {
		return
	}
	var req UpdateProviderRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p, err := h.repo.UpdateProvider(id, req)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "provider_id")
	if !ok {
		return
	}
	if err := h.repo.DeleteProvider(id); err != nil {
		writeRepoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateKey(w http.ResponseWriter, r *http.Request) {
	providerID, ok := pathID(w, r, "provider_id")
	if !ok {
		return
	}
	var req CreateKeyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	key, err := h.repo.CreateKey(providerID, req)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, key)
}

func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	providerID, ok := pathID(w, r, "provider_id")
	if !ok {
		return
	}
	keys, err := h.repo.ListKeys(providerID)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

func (h *Handler) UpdateKey(w http.ResponseWriter, r *http.Request) {
	providerID, ok := pathID(w, r, "provider_id")
	if !ok {
		return
	}
	keyID, ok := pathID(w, r, "key_id")
	if !ok {
		return
	}
	var req UpdateKeyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	key, err := h.repo.UpdateKey(providerID, keyID, req)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, key)
}

func (h *Handler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	providerID, ok := pathID(w, r, "provider_id")
	if !ok {
		return
	}
	keyID, ok := pathID(w, r, "key_id")
	if !ok {
		return
	}
	if err := h.repo.DeleteKey(providerID, keyID); err != nil {
		writeRepoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "请求 JSON 无效")
		return false
	}
	return true
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

func writeRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "资源不存在")
	case errors.Is(err, ErrInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "服务内部错误")
	}
}
