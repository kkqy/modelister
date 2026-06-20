// Package webui 嵌入已构建的前端资源（Vite 输出），并提供 SPA fallback。
// dist 目录由 ./frontend 下的 `npm run build` 生成；提交 .gitkeep 是为了让首次构建前也能编译。
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var embedded embed.FS

// Handler 返回用于服务前端静态资源的 http.Handler。
// 未知 GET 路径会回退到 index.html，以支持客户端路由；如果前端尚未构建，则返回构建提示。
func Handler() http.Handler {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		return notBuilt()
	}
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return notBuilt()
	}
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" || clean == "index.html" {
			writeIndex(w, index)
			return
		}
		if info, statErr := fs.Stat(dist, clean); statErr == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		// 未知路径：GET 回退到 SPA，其它方法返回 404。
		if r.Method == http.MethodGet {
			writeIndex(w, index)
			return
		}
		http.NotFound(w, r)
	})
}

func writeIndex(w http.ResponseWriter, index []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(index)
}

func notBuilt() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("前端尚未构建。请在 ./frontend 执行 `npm install && npm run build` 后重启服务。\n" +
			"Frontend not built. Run `npm install && npm run build` in ./frontend, then restart."))
	})
}
