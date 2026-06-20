package webui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// built 判断前端是否已经构建并进入 embed 资源。
func built() bool {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		return false
	}
	_, err = fs.ReadFile(dist, "index.html")
	return err == nil
}

func TestHandlerServesIndexWhenBuilt(t *testing.T) {
	if !built() {
		t.Skip("前端未构建，跳过 / frontend not built, skipping")
	}
	h := Handler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("root status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("root content-type = %q", ct)
	}

	// 未知路径回退到 index.html（SPA fallback）。
	spa := httptest.NewRecorder()
	h.ServeHTTP(spa, httptest.NewRequest(http.MethodGet, "/some/spa/route", nil))
	if spa.Code != http.StatusOK {
		t.Fatalf("spa fallback status = %d", spa.Code)
	}

	// Vite 产物中的 JS/CSS 资源也必须能通过 embed 静态服务直接访问。
	assetPath := findBuiltAsset(t)
	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/"+assetPath, nil))
	if asset.Code != http.StatusOK {
		t.Fatalf("asset %q status = %d", assetPath, asset.Code)
	}
}

func findBuiltAsset(t *testing.T) string {
	t.Helper()
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		t.Fatal(err)
	}
	var found string
	err = fs.WalkDir(dist, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if found == "" {
		t.Fatal("未找到已构建的前端资源")
	}
	return found
}

func TestHandlerReportsNotBuilt(t *testing.T) {
	if built() {
		t.Skip("前端已构建，跳过未构建分支 / frontend built, skipping not-built branch")
	}
	h := Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("not-built status = %d, want 503", rec.Code)
	}
}
