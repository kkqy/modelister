# OpenAI Provider Models Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **中文说明：** 执行本计划时必须逐任务推进，每个行为变更先写失败测试，再写实现代码。

**Goal:** 构建一个可 Docker 部署的 Golang 后端，用 SQLite 管理 OpenAI 兼容供应商、多个 Key、模型同步缓存、按 Key 分组展示和关键词搜索。

**Architecture:** 使用标准库 `net/http` 提供 REST API，`database/sql` 连接 SQLite，业务拆分为 `config`、`store`、`auth`、`providers`、`models`、`server` 等包。模型同步通过可注入 `http.Client` 调用 `{base_url}/v1/models`，接口默认读取缓存，支持 `refresh=true` 触发同步。

**Tech Stack:** Go 1.22、SQLite、`modernc.org/sqlite`、标准库 `net/http`、Docker 多阶段构建。

---

## File Structure

- Create: `go.mod`，Go 模块定义，模块名使用 `modelister`。
- Create: `cmd/modelister/main.go`，程序入口，加载配置、打开数据库、注册路由、启动 HTTP 服务。
- Create: `internal/config/config.go`，环境变量配置加载和校验。
- Create: `internal/config/config_test.go`，配置加载测试。
- Create: `internal/httpx/json.go`，统一 JSON 响应和错误响应。
- Create: `internal/store/store.go`，SQLite 初始化、schema、事务辅助。
- Create: `internal/store/store_test.go`，数据库初始化和级联删除测试。
- Create: `internal/auth/auth.go`，登录校验、签名 Cookie、认证中间件。
- Create: `internal/auth/auth_test.go`，登录、退出、受保护接口测试。
- Create: `internal/providers/types.go`，Provider 和 Key 请求响应类型。
- Create: `internal/providers/store.go`，Provider 和 Key 数据访问。
- Create: `internal/providers/handlers.go`，Provider 和 Key CRUD HTTP 处理器。
- Create: `internal/providers/handlers_test.go`，Provider 和 Key CRUD API 测试。
- Create: `internal/models/types.go`，模型缓存、同步结果、查询响应类型。
- Create: `internal/models/store.go`，模型缓存读写、分组、搜索、汇总查询。
- Create: `internal/models/sync.go`，调用 OpenAI 兼容 `/v1/models` 的同步服务。
- Create: `internal/models/handlers.go`，模型同步、列表、搜索 HTTP 处理器。
- Create: `internal/models/handlers_test.go`，同步、列表、搜索 API 测试。
- Create: `internal/server/server.go`，应用装配、路由注册、中间件挂载。
- Create: `internal/server/server_test.go`，健康检查和认证保护测试。
- Create: `Dockerfile`，后端容器构建。
- Create: `.dockerignore`，Docker 构建忽略规则。
- Create: `README.md`，项目运行和部署说明，包含中文说明。
- Create: `docs/frontend-api.md`，前端对接文档，包含认证、CRUD、同步、搜索和错误格式。

---

### Task 1: Go Module and Config

**Files:**
- Create: `go.mod`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing config tests**

Create `internal/config/config_test.go`:

```go
package config

import "testing"

func TestLoadRequiresAdminUsernamePasswordAndSessionSecret(t *testing.T) {
	t.Setenv("APP_ADMIN_USERNAME", "")
	t.Setenv("APP_ADMIN_PASSWORD", "")
	t.Setenv("APP_SESSION_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected missing required env vars to fail")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Setenv("APP_ADMIN_USERNAME", "admin")
	t.Setenv("APP_ADMIN_PASSWORD", "secret")
	t.Setenv("APP_SESSION_SECRET", "session-secret")
	t.Setenv("APP_DATABASE_PATH", "")
	t.Setenv("APP_HTTP_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load: %v", err)
	}
	if cfg.DatabasePath != "/data/modelister.db" {
		t.Fatalf("unexpected database path: %q", cfg.DatabasePath)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("unexpected http addr: %q", cfg.HTTPAddr)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./internal/config
```

Expected: FAIL because `go.mod` and `internal/config` do not exist.

- [ ] **Step 3: Create module and minimal config implementation**

Create `go.mod`:

```go
module modelister

go 1.22

require modernc.org/sqlite v1.36.0
```

Create `internal/config/config.go`:

```go
package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	AdminUsername string
	AdminPassword string
	SessionSecret string
	DatabasePath  string
	HTTPAddr      string
}

func Load() (Config, error) {
	cfg := Config{
		AdminUsername: strings.TrimSpace(os.Getenv("APP_ADMIN_USERNAME")),
		AdminPassword: os.Getenv("APP_ADMIN_PASSWORD"),
		SessionSecret: os.Getenv("APP_SESSION_SECRET"),
		DatabasePath:  strings.TrimSpace(os.Getenv("APP_DATABASE_PATH")),
		HTTPAddr:      strings.TrimSpace(os.Getenv("APP_HTTP_ADDR")),
	}
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "/data/modelister.db"
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	if cfg.AdminUsername == "" || cfg.AdminPassword == "" || cfg.SessionSecret == "" {
		return Config{}, errors.New("APP_ADMIN_USERNAME, APP_ADMIN_PASSWORD and APP_SESSION_SECRET are required")
	}
	return cfg, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```powershell
go test ./internal/config
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add go.mod internal/config/config.go internal/config/config_test.go
git commit -m "feat: add config loading"
```

---

### Task 2: SQLite Store and Schema

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/store_test.go`

- [ ] **Step 1: Write the failing store tests**

Create `internal/store/store_test.go`:

```go
package store

import (
	"database/sql"
	"testing"
)

func TestOpenInitializesSchema(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()

	tables := []string{"providers", "provider_keys", "model_cache"}
	for _, table := range tables {
		var name string
		err := db.QueryRow(`select name from sqlite_master where type='table' and name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("expected table %s to exist: %v", table, err)
		}
	}
}

func TestForeignKeysCascadeProviderDelete(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`insert into providers (name, base_url, note, enabled) values ('p', 'https://example.com', '', 1)`)
	if err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	providerID, err := lastID(db)
	if err != nil {
		t.Fatalf("provider id: %v", err)
	}
	_, err = db.Exec(`insert into provider_keys (provider_id, name, api_key, note, enabled) values (?, 'k', 'secret', '', 1)`, providerID)
	if err != nil {
		t.Fatalf("insert key: %v", err)
	}
	keyID, err := lastID(db)
	if err != nil {
		t.Fatalf("key id: %v", err)
	}
	_, err = db.Exec(`insert into model_cache (provider_id, provider_key_id, model_id, owned_by, raw_json) values (?, ?, 'gpt-4o', 'openai', '{}')`, providerID, keyID)
	if err != nil {
		t.Fatalf("insert model: %v", err)
	}

	if _, err := db.Exec(`delete from providers where id=?`, providerID); err != nil {
		t.Fatalf("delete provider: %v", err)
	}

	assertCount(t, db, "provider_keys", 0)
	assertCount(t, db, "model_cache", 0)
}

func lastID(db *sql.DB) (int64, error) {
	var id int64
	err := db.QueryRow(`select last_insert_rowid()`).Scan(&id)
	return id, err
}

func assertCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`select count(*) from ` + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("count %s = %d, want %d", table, got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./internal/store
```

Expected: FAIL because `Open` is not implemented.

- [ ] **Step 3: Implement store schema**

Create `internal/store/store.go`:

```go
package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`pragma foreign_keys = on`); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	statements := []string{
		`create table if not exists providers (
			id integer primary key autoincrement,
			name text not null,
			base_url text not null,
			note text not null default '',
			enabled integer not null default 1,
			created_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			updated_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)`,
		`create table if not exists provider_keys (
			id integer primary key autoincrement,
			provider_id integer not null references providers(id) on delete cascade,
			name text not null,
			api_key text not null,
			note text not null default '',
			enabled integer not null default 1,
			last_sync_at text not null default '',
			last_sync_error text not null default '',
			created_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			updated_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)`,
		`create table if not exists model_cache (
			id integer primary key autoincrement,
			provider_id integer not null references providers(id) on delete cascade,
			provider_key_id integer not null references provider_keys(id) on delete cascade,
			model_id text not null,
			owned_by text not null default '',
			raw_json text not null default '{}',
			created_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			updated_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			unique(provider_key_id, model_id)
		)`,
		`create index if not exists idx_provider_keys_provider_id on provider_keys(provider_id)`,
		`create index if not exists idx_model_cache_provider_key_id on model_cache(provider_key_id)`,
		`create index if not exists idx_model_cache_model_id on model_cache(model_id)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```powershell
go test ./internal/store
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add go.mod go.sum internal/store/store.go internal/store/store_test.go
git commit -m "feat: initialize sqlite store"
```

---

### Task 3: JSON Helpers and Cookie Authentication

**Files:**
- Create: `internal/httpx/json.go`
- Create: `internal/auth/auth.go`
- Create: `internal/auth/auth_test.go`

- [ ] **Step 1: Write the failing auth tests**

Create `internal/auth/auth_test.go`:

```go
package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginSetsHttpOnlyCookie(t *testing.T) {
	manager := NewManager("admin", "secret", "session-secret")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()

	manager.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	if !cookies[0].HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}
}

func TestProtectedRejectsMissingCookie(t *testing.T) {
	manager := NewManager("admin", "secret", "session-secret")
	next := manager.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./internal/auth
```

Expected: FAIL because auth package is not implemented.

- [ ] **Step 3: Implement JSON helper and auth manager**

Create `internal/httpx/json.go` with `WriteJSON(w,status,value)` and `WriteError(w,status,code,message)` helpers. Error response must be:

```json
{"error":{"code":"invalid_request","message":"请求参数无效"}}
```

Create `internal/auth/auth.go` with:

```go
type Manager struct {
	adminUsername string
	adminPassword string
	sessionSecret []byte
}
```

Required behavior:

- `NewManager(username, password, secret string) *Manager`
- `Login(w http.ResponseWriter, r *http.Request)` decodes JSON username/password and compares exact strings.
- Successful login writes signed cookie named `modelister_session`.
- Cookie value format: `admin|hex_hmac_sha256(admin, sessionSecret)`.
- Cookie attributes: `Path=/`, `HttpOnly=true`, `SameSite=http.SameSiteLaxMode`.
- `Me(w,r)` returns `{"authenticated":true,"username":"admin"}` when authenticated.
- `Logout(w,r)` expires the cookie and returns `{"ok":true}`.
- `RequireAuth(next http.Handler) http.Handler` rejects invalid or missing cookie with `401 unauthorized`.

- [ ] **Step 4: Run test to verify it passes**

Run:

```powershell
go test ./internal/auth
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add internal/httpx/json.go internal/auth/auth.go internal/auth/auth_test.go
git commit -m "feat: add cookie authentication"
```

---

### Task 4: Provider and Key Repository

**Files:**
- Create: `internal/providers/types.go`
- Create: `internal/providers/store.go`
- Create: `internal/providers/store_test.go`

- [ ] **Step 1: Write the failing repository tests**

Create `internal/providers/store_test.go`:

```go
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
		Name: "OpenAI",
		BaseURL: "https://api.openai.com/",
		Note: "主供应商",
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./internal/providers
```

Expected: FAIL because provider repository is not implemented.

- [ ] **Step 3: Implement provider and key repository**

Create `internal/providers/types.go`:

```go
package providers

type Provider struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	BaseURL   string `json:"base_url"`
	Note      string `json:"note"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ProviderKey struct {
	ID            int64  `json:"id"`
	ProviderID    int64  `json:"provider_id"`
	Name          string `json:"name"`
	APIKey        string `json:"-"`
	APIKeyMasked  string `json:"api_key_masked"`
	Note          string `json:"note"`
	Enabled       bool   `json:"enabled"`
	LastSyncAt    string `json:"last_sync_at"`
	LastSyncError string `json:"last_sync_error"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type CreateProviderRequest struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Note    string `json:"note"`
	Enabled bool   `json:"enabled"`
}

type UpdateProviderRequest struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Note    string `json:"note"`
	Enabled bool   `json:"enabled"`
}

type CreateKeyRequest struct {
	Name    string `json:"name"`
	APIKey  string `json:"api_key"`
	Note    string `json:"note"`
	Enabled bool   `json:"enabled"`
}

type UpdateKeyRequest struct {
	Name    string `json:"name"`
	APIKey  string `json:"api_key"`
	Note    string `json:"note"`
	Enabled bool   `json:"enabled"`
}
```

Create `internal/providers/store.go` with:

- `NewRepository(db *sql.DB) *Repository`
- `CreateProvider(req CreateProviderRequest) (Provider, error)`
- `ListProviders() ([]Provider, error)`
- `GetProvider(id int64) (Provider, error)`
- `UpdateProvider(id int64, req UpdateProviderRequest) (Provider, error)`
- `DeleteProvider(id int64) error`
- `CreateKey(providerID int64, req CreateKeyRequest) (ProviderKey, error)`
- `ListKeys(providerID int64) ([]ProviderKey, error)`
- `GetKey(providerID, keyID int64) (ProviderKey, error)`
- `GetKeyWithSecret(providerID, keyID int64) (ProviderKey, error)`
- `UpdateKey(providerID, keyID int64, req UpdateKeyRequest) (ProviderKey, error)`
- `DeleteKey(providerID, keyID int64) error`
- `ListEnabledProviderKeys() ([]ProviderWithKey, error)` for model sync.

Validation rules:

- Trim `name`, `base_url`, and `note`.
- Provider `name` and `base_url` cannot be empty.
- Key `name` and new `api_key` cannot be empty.
- Provider `base_url` must parse as `http` or `https`.
- Normalize `base_url` by trimming trailing `/`.
- List and create responses must not expose full `api_key`; use `api_key_masked`.

- [ ] **Step 4: Run test to verify it passes**

Run:

```powershell
go test ./internal/providers
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add internal/providers/types.go internal/providers/store.go internal/providers/store_test.go
git commit -m "feat: add provider repository"
```

---

### Task 5: Provider and Key HTTP Handlers

**Files:**
- Create: `internal/providers/handlers.go`
- Create: `internal/providers/handlers_test.go`

- [ ] **Step 1: Write the failing handler tests**

Create `internal/providers/handlers_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./internal/providers
```

Expected: FAIL because HTTP handlers are not implemented.

- [ ] **Step 3: Implement Provider and Key handlers**

Create `internal/providers/handlers.go` with:

- `type Handler struct { repo *Repository }`
- `NewHandler(repo *Repository) *Handler`
- `CreateProvider`, `ListProviders`, `UpdateProvider`, `DeleteProvider`
- `CreateKey`, `ListKeys`, `UpdateKey`, `DeleteKey`

Path parsing rules:

- Use standard library route patterns in `internal/server` for exact routing where possible.
- For dynamic IDs in Go 1.22 `ServeMux`, use `r.PathValue("provider_id")` and `r.PathValue("key_id")`.
- Convert ID with `strconv.ParseInt`, return `400 invalid_request` when invalid.

Response rules:

- Create returns `201`.
- Update and list return `200`.
- Delete returns `204` with empty body.
- Repository validation errors return `400 invalid_request`.
- Missing rows return `404 not_found`.

- [ ] **Step 4: Run test to verify it passes**

Run:

```powershell
go test ./internal/providers
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add internal/providers/handlers.go internal/providers/handlers_test.go
git commit -m "feat: add provider http handlers"
```

---

### Task 6: Model Sync Service and Cache Store

**Files:**
- Create: `internal/models/types.go`
- Create: `internal/models/store.go`
- Create: `internal/models/sync.go`
- Create: `internal/models/sync_test.go`

- [ ] **Step 1: Write the failing sync tests**

Create `internal/models/sync_test.go`:

```go
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
	p, err := providerRepo.CreateProvider(providers.CreateProviderRequest{Name: "P", BaseURL: upstream.URL, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	k, err := providerRepo.CreateKey(p.ID, providers.CreateKeyRequest{Name: "K", APIKey: "sk-test", Enabled: true})
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./internal/models
```

Expected: FAIL because model repository and sync service are not implemented.

- [ ] **Step 3: Implement model types, cache repository, and sync service**

Create `internal/models/types.go` with:

```go
package models

type Model struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
	RawJSON string `json:"-"`
}

type KeyModels struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Note          string  `json:"note"`
	LastSyncAt    string  `json:"last_sync_at"`
	LastSyncError string  `json:"last_sync_error"`
	Models        []Model `json:"models"`
}

type ProviderModels struct {
	ID      int64       `json:"id"`
	Name    string      `json:"name"`
	BaseURL string      `json:"base_url"`
	Note    string      `json:"note"`
	Keys    []KeyModels `json:"keys,omitempty"`
	Models  []Model     `json:"models,omitempty"`
}

type ListResponse struct {
	Mode      string           `json:"mode"`
	Query     string           `json:"query,omitempty"`
	Providers []ProviderModels `json:"providers"`
}

type SyncResult struct {
	ProviderID int64  `json:"provider_id"`
	KeyID      int64  `json:"key_id"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
	Count      int    `json:"count"`
}
```

Create `internal/models/store.go` with:

- `ReplaceKeyModels(providerID, keyID int64, items []Model) error`
- `SetKeySyncSuccess(providerID, keyID int64) error`
- `SetKeySyncError(providerID, keyID int64, message string) error`
- `ListByKey(query string) ([]ProviderModels, error)`
- `ListMerged(query string) ([]ProviderModels, error)`

Query rules:

- Only include enabled providers and enabled keys.
- Empty query returns all cached models.
- Non-empty query uses `lower(model_id) like lower('%' || ? || '%')`.
- `ListByKey` preserves Provider -> Key -> Models hierarchy.
- `ListMerged` returns one Provider row with de-duplicated `Models` by `model_id`.

Create `internal/models/sync.go` with:

- `NewSyncService(providerRepo *providers.Repository, modelRepo *Repository, client *http.Client) *SyncService`
- `SyncKey(providerID, keyID int64) SyncResult`
- `SyncProvider(providerID int64) []SyncResult`
- `SyncAll() []SyncResult`

Upstream parsing rules:

- Request URL is `{base_url}/v1/models`.
- Header is `Authorization: Bearer {api_key}`.
- Non-2xx status returns `upstream_error` text and leaves old cache intact.
- Successful response decodes `{"data":[{"id":"...","owned_by":"..."}]}`.
- Empty or missing model ID entries are ignored.
- Store original model JSON in `raw_json`.

- [ ] **Step 4: Run test to verify it passes**

Run:

```powershell
go test ./internal/models
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add internal/models/types.go internal/models/store.go internal/models/sync.go internal/models/sync_test.go
git commit -m "feat: add model sync service"
```

---

### Task 7: Model HTTP Handlers for Sync, List, and Search

**Files:**
- Create: `internal/models/handlers.go`
- Create: `internal/models/handlers_test.go`

- [ ] **Step 1: Write the failing model handler tests**

Create `internal/models/handlers_test.go`:

```go
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
	p, err := providerRepo.CreateProvider(providers.CreateProviderRequest{Name: "P", BaseURL: "https://example.com", Note: "供应商备注", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	k1, _ := providerRepo.CreateKey(p.ID, providers.CreateKeyRequest{Name: "K1", APIKey: "sk-1", Note: "第一个", Enabled: true})
	k2, _ := providerRepo.CreateKey(p.ID, providers.CreateKeyRequest{Name: "K2", APIKey: "sk-2", Note: "第二个", Enabled: true})

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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./internal/models
```

Expected: FAIL because model handlers are not implemented.

- [ ] **Step 3: Implement model handlers**

Create `internal/models/handlers.go` with:

- `NewHandler(repo *Repository, sync *SyncService) *Handler`
- `List(w,r)` for `GET /api/models`
- `Search(w,r)` for `GET /api/models/search`
- `SyncAll(w,r)` for `POST /api/models/sync`
- `SyncProvider(w,r)` for `POST /api/providers/{provider_id}/sync`
- `SyncKey(w,r)` for `POST /api/providers/{provider_id}/keys/{key_id}/sync`

Rules:

- `mode` defaults to `by_key`.
- Accepted modes are `by_key` and `merged`; other values return `400 invalid_request`.
- `Search` requires non-empty `q`.
- `refresh=true` calls `SyncAll()` before list/search.
- Response uses `ListResponse`.
- Sync endpoints return `{"results":[...]}`.

- [ ] **Step 4: Run test to verify it passes**

Run:

```powershell
go test ./internal/models
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add internal/models/handlers.go internal/models/handlers_test.go
git commit -m "feat: add model query handlers"
```

---

### Task 8: Server Routing and Main Entrypoint

**Files:**
- Create: `internal/server/server.go`
- Create: `internal/server/server_test.go`
- Create: `cmd/modelister/main.go`

- [ ] **Step 1: Write the failing server tests**

Create `internal/server/server_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./internal/server
```

Expected: FAIL because server package is not implemented.

- [ ] **Step 3: Implement routing and main**

Create `internal/server/server.go` with:

- `func New(auth *auth.Manager, providers *providers.Handler, models *models.Handler) http.Handler`
- Public routes:
  - `GET /healthz`
  - `POST /api/auth/login`
  - `GET /api/auth/me`
  - `POST /api/auth/logout`
- Protected routes under `/api/`:
  - Provider CRUD
  - Key CRUD
  - Model sync/list/search
- Wrap protected routes with `auth.RequireAuth`.

Create `cmd/modelister/main.go` with:

- Load config.
- Open SQLite store.
- Build repositories, auth manager, sync service, handlers, router.
- Start `http.ListenAndServe(cfg.HTTPAddr, router)`.
- Log fatal startup errors in Chinese and English enough for operators, for example `配置错误 / config error: ...`。

- [ ] **Step 4: Run test to verify it passes**

Run:

```powershell
go test ./internal/server
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add internal/server/server.go internal/server/server_test.go cmd/modelister/main.go
git commit -m "feat: wire http server"
```

---

### Task 9: Docker, README, and Frontend API Documentation

**Files:**
- Create: `Dockerfile`
- Create: `.dockerignore`
- Create: `README.md`
- Create: `docs/frontend-api.md`

- [ ] **Step 1: Write documentation and Docker verification checklist**

Create `README.md` with these sections:

```markdown
# Modelister

Modelister 是一个 OpenAI 兼容供应商模型管理后端。它可以通过 Docker 部署到服务器，使用 SQLite 保存供应商、Key 和模型缓存。

## 环境变量

- `APP_ADMIN_USERNAME`：管理员用户名，必填。
- `APP_ADMIN_PASSWORD`：管理员密码，必填。
- `APP_SESSION_SECRET`：Cookie 会话签名密钥，必填。
- `APP_DATABASE_PATH`：SQLite 路径，默认 `/data/modelister.db`。
- `APP_HTTP_ADDR`：监听地址，默认 `:8080`。

## Docker 运行

```powershell
docker build -t modelister .
docker run --rm -p 8080:8080 -v modelister-data:/data `
  -e APP_ADMIN_USERNAME=admin `
  -e APP_ADMIN_PASSWORD=change-me `
  -e APP_SESSION_SECRET=change-me-session-secret `
  modelister
```

生产环境建议通过 HTTPS 反向代理暴露服务。
```

Create `docs/frontend-api.md` containing:

- 登录流程：`POST /api/auth/login`、`GET /api/auth/me`、`POST /api/auth/logout`
- Provider CRUD request/response examples
- Key CRUD request/response examples and `api_key_masked` rule
- Sync endpoints
- `GET /api/models?mode=by_key`
- `GET /api/models?mode=merged`
- `GET /api/models/search?q=gpt`
- `refresh=true`
- Unified error response

- [ ] **Step 2: Create Docker files**

Create `.dockerignore`:

```text
.git
docs/superpowers
*.db
*.db-shm
*.db-wal
```

Create `Dockerfile`:

```dockerfile
FROM golang:1.22-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/modelister ./cmd/modelister

FROM debian:bookworm-slim

WORKDIR /app
RUN mkdir -p /data
COPY --from=build /out/modelister /app/modelister
EXPOSE 8080
ENV APP_DATABASE_PATH=/data/modelister.db
ENV APP_HTTP_ADDR=:8080
CMD ["/app/modelister"]
```

- [ ] **Step 3: Run full test suite**

Run:

```powershell
go test ./...
```

Expected: PASS.

- [ ] **Step 4: Build Docker image**

Run:

```powershell
docker build -t modelister .
```

Expected: image builds successfully.

- [ ] **Step 5: Commit**

```powershell
git add Dockerfile .dockerignore README.md docs/frontend-api.md
git commit -m "docs: add deployment and frontend api docs"
```

---

### Task 10: Final Verification

**Files:**
- Modify only if verification reveals a real bug.

- [ ] **Step 1: Run all tests**

Run:

```powershell
go test ./...
```

Expected: PASS.

- [ ] **Step 2: Run formatting**

Run:

```powershell
gofmt -w cmd internal
```

Expected: no command error.

- [ ] **Step 3: Re-run tests after formatting**

Run:

```powershell
go test ./...
```

Expected: PASS.

- [ ] **Step 4: Inspect git diff**

Run:

```powershell
git diff --stat
git diff
```

Expected:

- Only project files from this plan changed.
- No full API Key is returned in JSON response types.
- Chinese user-facing documentation is present in `README.md` and `docs/frontend-api.md`.

- [ ] **Step 5: Provide completion summary**

Final summary must include:

- Implemented backend capabilities.
- Test command and result.
- Docker build command and result, or reason it was not run.
- Frontend documentation path: `docs/frontend-api.md`.
