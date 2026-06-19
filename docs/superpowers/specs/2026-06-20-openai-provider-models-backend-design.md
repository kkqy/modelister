# OpenAI 兼容供应商模型管理后端设计

## 背景

本项目第一版要实现一个可 Docker 部署到服务器的后端服务，用于管理 OpenAI 兼容 API 供应商。管理员可以维护供应商的 Base URL 和多个 API Key，后端通过调用各 Key 对应供应商的 `/v1/models` 接口获取支持的模型，并提供按 Key 分组或汇总去重的查询接口。

第一版只实现后端，不实现前端页面。后端需要同时提供前端开发对接文档，方便后续开发管理界面。

## 技术选型

- 开发语言：Golang。
- 部署方式：Docker 容器。
- 数据库：SQLite。
- API 风格：REST JSON API。
- 管理员认证：环境变量配置管理员用户名和密码，登录后使用 HttpOnly Cookie 会话。
- API Key 存储：明文保存到 SQLite。该设计基于服务器本地可信部署模型；如果服务器文件泄露，环境变量也通常无法视为安全边界，因此第一版不引入本地加密存储。

## 环境变量

- `APP_ADMIN_USERNAME`：管理员用户名，必填。
- `APP_ADMIN_PASSWORD`：管理员密码，必填。
- `APP_SESSION_SECRET`：Cookie 会话签名密钥，必填。
- `APP_DATABASE_PATH`：SQLite 数据库路径，默认可为 `/data/modelister.db`。
- `APP_HTTP_ADDR`：HTTP 监听地址，默认可为 `:8080`。

启动时如果必填环境变量缺失，服务应拒绝启动并输出清晰错误。

## 核心概念

### Provider

供应商表示一个 OpenAI 兼容 API 服务端。

字段：

- `id`：唯一 ID。
- `name`：供应商名称。
- `base_url`：供应商 Base URL，例如 `https://api.openai.com`。后端调用模型接口时拼接为 `{base_url}/v1/models`。
- `note`：供应商备注。
- `enabled`：是否启用。
- `created_at`：创建时间。
- `updated_at`：更新时间。

### Provider Key

一个供应商可以有多个 Key。Key 归属于 Provider。

字段：

- `id`：唯一 ID。
- `provider_id`：所属供应商 ID。
- `name`：Key 显示名称。
- `api_key`：API Key 明文。
- `note`：Key 备注。
- `enabled`：是否启用。
- `last_sync_at`：最近一次模型同步时间。
- `last_sync_error`：最近一次模型同步错误，成功同步后清空。
- `created_at`：创建时间。
- `updated_at`：更新时间。

### Model Cache

后端将每个 Key 同步到的模型列表写入 SQLite。模型缓存以 Key 为来源保存，即使多个 Key 返回同一个模型 ID，也保留各自的来源关系。

字段：

- `id`：唯一 ID。
- `provider_id`：供应商 ID。
- `provider_key_id`：Key ID。
- `model_id`：模型 ID，即 `/v1/models` 返回对象的 `id`。
- `owned_by`：上游返回的 `owned_by`，可能为空。
- `raw_json`：模型对象原始 JSON，方便前端或后续扩展读取额外字段。
- `created_at`：首次写入时间。
- `updated_at`：最近更新时间。

## 认证设计

### 登录

接口：`POST /api/auth/login`

请求：

```json
{
  "username": "admin",
  "password": "password"
}
```

行为：

- 与 `APP_ADMIN_USERNAME` 和 `APP_ADMIN_PASSWORD` 比对。
- 成功后设置 HttpOnly Cookie。
- Cookie 应设置 `HttpOnly`、`SameSite=Lax`。
- 第一版不强制 `Secure`，因为是否启用 HTTPS 由反向代理或部署环境决定；文档需要说明生产环境应通过 HTTPS 暴露。

### 当前登录状态

接口：`GET /api/auth/me`

返回当前用户状态，供前端刷新页面后判断是否仍已登录。

### 退出

接口：`POST /api/auth/logout`

行为：清除会话 Cookie。

### 受保护接口

除健康检查和登录接口外，所有 `/api/*` 管理接口都需要登录。

## Provider 管理接口

### 创建 Provider

`POST /api/providers`

请求字段：

- `name`：必填。
- `base_url`：必填。
- `note`：可选。
- `enabled`：可选，默认 `true`。

校验：

- `base_url` 必须是合法 HTTP 或 HTTPS URL。
- 保存时去掉末尾多余 `/`，避免拼接 `/v1/models` 时出现重复斜杠。

### 列表 Provider

`GET /api/providers`

返回 Provider 列表，并可包含每个 Provider 下 Key 数量和最近同步摘要。

### 更新 Provider

`PUT /api/providers/{provider_id}`

允许更新：

- `name`
- `base_url`
- `note`
- `enabled`

### 删除 Provider

`DELETE /api/providers/{provider_id}`

删除 Provider 时同步删除其 Key 和模型缓存。

## Provider Key 管理接口

### 创建 Key

`POST /api/providers/{provider_id}/keys`

请求字段：

- `name`：必填。
- `api_key`：必填。
- `note`：可选。
- `enabled`：可选，默认 `true`。

### 列表 Key

`GET /api/providers/{provider_id}/keys`

返回指定 Provider 下所有 Key。为避免前端误展示完整密钥，返回时默认不返回完整 `api_key`，只返回脱敏字段，例如 `sk-...abcd`。如果后续需要编辑完整 Key，应由前端重新提交新值。

### 更新 Key

`PUT /api/providers/{provider_id}/keys/{key_id}`

允许更新：

- `name`
- `api_key`：可选；为空或未传时保持原值。
- `note`
- `enabled`

### 删除 Key

`DELETE /api/providers/{provider_id}/keys/{key_id}`

删除 Key 时同步删除其模型缓存。

## 模型同步

### 同步单个 Key

`POST /api/providers/{provider_id}/keys/{key_id}/sync`

行为：

- 校验 Provider 和 Key 都存在。
- 即使 Provider 或 Key 处于禁用状态，管理员手动同步接口仍可返回明确错误，默认不执行同步。
- 使用 `Authorization: Bearer {api_key}` 请求 `{base_url}/v1/models`。
- 请求成功后，用返回的 `data` 数组替换该 Key 的模型缓存。
- 成功时更新 `last_sync_at`，清空 `last_sync_error`。
- 失败时保留旧模型缓存，写入 `last_sync_error`。

### 同步单个 Provider

`POST /api/providers/{provider_id}/sync`

行为：

- 同步该 Provider 下所有启用的 Key。
- 返回每个 Key 的同步结果。

### 同步全部

`POST /api/models/sync`

行为：

- 同步所有启用 Provider 下所有启用 Key。
- 返回按 Provider 和 Key 分组的同步结果。

## 模型查询

### 模型列表

`GET /api/models`

查询参数：

- `mode`：展示模式，默认 `by_key`。
  - `by_key`：按 Provider -> Key -> Models 分组。
  - `merged`：同一 Provider 下按 `model_id` 去重汇总。
- `refresh`：可选布尔值。为 `true` 时，先同步所有启用 Provider 和启用 Key，再返回查询结果。

默认行为：

- 只返回启用 Provider 和启用 Key 的模型缓存。
- 默认按 Key 分组显示。
- 同一个 Provider 下两个 Key 都有同一个模型时，`by_key` 模式会显示两个 Key，各自包含该模型。

### 模型搜索

`GET /api/models/search?q={keyword}`

查询参数：

- `q`：搜索关键词，必填。
- `mode`：默认 `by_key`，也支持 `merged`。
- `refresh`：可选布尔值。为 `true` 时，先同步所有启用 Provider 和启用 Key，再返回搜索结果。

匹配规则：

- 对模型 ID 做大小写不敏感的包含匹配。
- 如果原始模型对象中存在可用名称字段，后续可以纳入匹配；第一版以 OpenAI `/v1/models` 标准返回的 `id` 为主要模型名称。

默认返回结构：

```json
{
  "mode": "by_key",
  "query": "gpt",
  "providers": [
    {
      "id": 1,
      "name": "供应商 A",
      "base_url": "https://example.com",
      "note": "主要供应商",
      "keys": [
        {
          "id": 11,
          "name": "Key 1",
          "note": "高额度 key",
          "last_sync_at": "2026-06-20T10:00:00Z",
          "last_sync_error": "",
          "models": [
            {
              "id": "gpt-4o",
              "owned_by": "openai"
            }
          ]
        },
        {
          "id": 12,
          "name": "Key 2",
          "note": "备用 key",
          "last_sync_at": "2026-06-20T10:01:00Z",
          "last_sync_error": "",
          "models": [
            {
              "id": "gpt-4o-mini",
              "owned_by": "openai"
            }
          ]
        }
      ]
    }
  ]
}
```

`merged` 模式返回同一 Provider 下去重后的模型，并列出支持该模型的 Key，供前端实现汇总视图。

## 错误处理

统一错误格式：

```json
{
  "error": {
    "code": "invalid_request",
    "message": "请求参数无效"
  }
}
```

建议错误码：

- `unauthorized`：未登录或会话无效。
- `forbidden`：认证成功但操作不允许，第一版通常不使用。
- `invalid_request`：参数错误。
- `not_found`：资源不存在。
- `upstream_error`：调用上游 `/v1/models` 失败。
- `internal_error`：服务内部错误。

## 前端对接文档要求

实现后需要提供前端开发文档，至少包含：

- Docker 和环境变量说明。
- 登录、退出、登录状态检查流程。
- Provider CRUD API。
- Provider Key CRUD API。
- 模型同步 API。
- 模型列表和搜索 API。
- `by_key` 与 `merged` 返回结构示例。
- API Key 返回脱敏规则。
- 错误格式和常见错误码。

## 第一版不包含

- 前端页面。
- 多用户、多角色权限。
- API Key 本地加密存储。
- 定时后台同步任务。
- 完整审计日志。
- 供应商请求代理或聊天补全代理。

这些能力可以在后续版本中扩展，不影响第一版后端接口设计。

## 测试策略

- 配置加载测试：缺少必填环境变量时启动配置校验失败。
- 认证测试：登录成功、登录失败、受保护接口未登录拒绝。
- Provider 和 Key CRUD 测试：创建、更新备注、禁用、删除级联缓存。
- 模型同步测试：使用本地测试 HTTP Server 模拟 `/v1/models`，验证成功缓存、失败保留旧缓存并记录错误。
- 模型列表测试：验证默认 `by_key`，以及 `merged` 去重。
- 模型搜索测试：验证大小写不敏感包含匹配，默认按 Key 分组，并能显示同一 Provider 下多个命中 Key。
- API 响应测试：验证 Key 列表不返回完整明文密钥，只返回脱敏值。

