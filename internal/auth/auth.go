package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"modelister/internal/httpx"
)

const CookieName = "modelister_session"

type Manager struct {
	adminUsername string
	adminPassword string
	sessionSecret []byte
}

func NewManager(username, password, secret string) *Manager {
	return &Manager{
		adminUsername: username,
		adminPassword: password,
		sessionSecret: []byte(secret),
	}
}

func (m *Manager) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "请求 JSON 无效")
		return
	}
	if req.Username != m.adminUsername || req.Password != m.adminPassword {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "用户名或密码错误")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    m.signedValue(m.adminUsername),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"username": m.adminUsername,
	})
}

func (m *Manager) Me(w http.ResponseWriter, r *http.Request) {
	if !m.Authenticated(r) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"username":      m.adminUsername,
	})
}

func (m *Manager) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (m *Manager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.Authenticated(r) {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Manager) Authenticated(r *http.Request) bool {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return false
	}
	parts := strings.Split(cookie.Value, "|")
	if len(parts) != 2 {
		return false
	}
	if parts[0] != m.adminUsername {
		return false
	}
	expected := m.signature(parts[0])
	return hmac.Equal([]byte(parts[1]), []byte(expected))
}

func (m *Manager) signedValue(username string) string {
	return username + "|" + m.signature(username)
}

func (m *Manager) signature(value string) string {
	mac := hmac.New(sha256.New, m.sessionSecret)
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
