# Modelister

Modelister 是一个 OpenAI 兼容供应商模型管理后端。它可以通过 Docker 部署到服务器，使用 SQLite 保存供应商、Key 和模型缓存。

后端提供 REST API（见 [docs/frontend-api.md](docs/frontend-api.md)），并内置一个 React 管理控制台。前端源码在 [`frontend/`](frontend/)，构建产物会被 Go 二进制通过 `//go:embed` 嵌入，与 `/api` 同源托管，单容器即可运行。

## 功能

- 管理 OpenAI 兼容供应商：名称、Base URL、备注、启用状态。
- 每个供应商可管理多个 API Key：名称、明文 Key、备注、启用状态。
- 通过 `{base_url}/v1/models` 同步每个 Key 支持的模型。
- 模型缓存保存在 SQLite 中，查询默认按 Provider -> Key -> Models 分组。
- 支持 `merged` 模式在同一供应商内按模型 ID 去重。
- 支持关键词搜索模型，默认按 Key 分组返回命中的供应商、Key 和模型。
- 管理接口使用环境变量管理员账号密码登录，登录态通过 HttpOnly Cookie 保存。

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

生产环境建议通过 HTTPS 反向代理暴露服务。API Key 当前按明文保存在 SQLite 中，请确保服务器和数据库卷的访问权限可信。

## 本地测试

```powershell
go test ./...
```

## 前端开发

前端是 Vite + React 项目，位于 `frontend/`。

```powershell
cd frontend
npm install
npm run dev      # 开发服务器，默认 http://localhost:5173，自动把 /api 代理到 :8080
npm run build    # 构建产物输出到 ../internal/webui/dist，供 Go 嵌入
```

开发时需要同时运行后端（`go run ./cmd/modelister`，记得设置环境变量）。如需指向非默认后端地址，设置 `MODELISTER_BACKEND` 环境变量后再运行 `npm run dev`。

修改前端后，先 `npm run build` 再 `go build` / 重启服务，嵌入的页面才会更新。Docker 镜像会在构建阶段自动执行前端构建，无需手动操作。
