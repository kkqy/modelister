# Modelister 前端 API 对接文档

本文档面向前端开发，描述 Modelister 后端第一版 REST API。所有示例均以同源部署为假设，登录后浏览器会自动携带 HttpOnly Cookie。

## 基础约定

- API Base Path：`/api`
- 请求和响应格式：JSON。
- 除 `POST /api/auth/login`、`GET /api/auth/me`、`POST /api/auth/logout` 和 `GET /healthz` 外，其余管理接口都需要登录。
- 生产环境建议通过 HTTPS 访问，避免 Cookie 和请求内容在传输层泄露。
- API Key 在数据库中明文保存，但列表响应不会返回完整 Key，只返回 `api_key_masked`。

## 错误格式

```json
{
  "error": {
    "code": "invalid_request",
    "message": "请求参数无效"
  }
}
```

常见错误码：

- `unauthorized`：未登录或登录失败。
- `invalid_request`：请求参数错误。
- `not_found`：资源不存在。
- `upstream_error`：上游 `/v1/models` 调用失败。
- `internal_error`：服务内部错误。

## 认证

### 登录

`POST /api/auth/login`

请求：

```json
{
  "username": "admin",
  "password": "change-me"
}
```

成功响应：

```json
{
  "ok": true,
  "username": "admin"
}
```

后端会设置 `modelister_session` HttpOnly Cookie。前端不需要读取 Cookie，只需要在 `fetch` 中设置 `credentials: "include"`。

### 当前登录状态

`GET /api/auth/me`

响应：

```json
{
  "authenticated": true,
  "username": "admin"
}
```

未登录时：

```json
{
  "authenticated": false
}
```

### 退出

`POST /api/auth/logout`

响应：

```json
{
  "ok": true
}
```

## Provider API

### 创建供应商

`POST /api/providers`

请求：

```json
{
  "name": "OpenAI",
  "base_url": "https://api.openai.com",
  "note": "主供应商",
  "enabled": true
}
```

响应：

```json
{
  "id": 1,
  "name": "OpenAI",
  "base_url": "https://api.openai.com",
  "note": "主供应商",
  "enabled": true,
  "created_at": "2026-06-20T10:00:00.000Z",
  "updated_at": "2026-06-20T10:00:00.000Z"
}
```

### 供应商列表

`GET /api/providers`

响应：

```json
{
  "providers": [
    {
      "id": 1,
      "name": "OpenAI",
      "base_url": "https://api.openai.com",
      "note": "主供应商",
      "enabled": true,
      "created_at": "2026-06-20T10:00:00.000Z",
      "updated_at": "2026-06-20T10:00:00.000Z"
    }
  ]
}
```

### 更新供应商

`PUT /api/providers/{provider_id}`

请求字段同创建接口。

### 删除供应商

`DELETE /api/providers/{provider_id}`

成功响应状态码：`204`。删除供应商会级联删除其 Key 和模型缓存。

## Provider Key API

### 创建 Key

`POST /api/providers/{provider_id}/keys`

请求：

```json
{
  "name": "生产 Key",
  "api_key": "sk-abcdef123456",
  "note": "高额度",
  "enabled": true
}
```

响应：

```json
{
  "id": 1,
  "provider_id": 1,
  "name": "生产 Key",
  "api_key_masked": "sk-...3456",
  "note": "高额度",
  "enabled": true,
  "last_sync_at": "",
  "last_sync_error": "",
  "created_at": "2026-06-20T10:00:00.000Z",
  "updated_at": "2026-06-20T10:00:00.000Z"
}
```

### Key 列表

`GET /api/providers/{provider_id}/keys`

响应：

```json
{
  "keys": [
    {
      "id": 1,
      "provider_id": 1,
      "name": "生产 Key",
      "api_key_masked": "sk-...3456",
      "note": "高额度",
      "enabled": true,
      "last_sync_at": "",
      "last_sync_error": "",
      "created_at": "2026-06-20T10:00:00.000Z",
      "updated_at": "2026-06-20T10:00:00.000Z"
    }
  ]
}
```

### 更新 Key

`PUT /api/providers/{provider_id}/keys/{key_id}`

请求：

```json
{
  "name": "生产 Key",
  "api_key": "",
  "note": "更新后的备注",
  "enabled": true
}
```

`api_key` 为空字符串时保持原 Key 不变。如需替换 Key，提交新的完整 Key。

### 删除 Key

`DELETE /api/providers/{provider_id}/keys/{key_id}`

成功响应状态码：`204`。删除 Key 会级联删除该 Key 的模型缓存。

## 模型同步 API

### 同步单个 Key

`POST /api/providers/{provider_id}/keys/{key_id}/sync`

响应：

```json
{
  "results": [
    {
      "provider_id": 1,
      "key_id": 1,
      "ok": true,
      "count": 2
    }
  ]
}
```

### 同步单个供应商

`POST /api/providers/{provider_id}/sync`

同步该供应商下所有启用 Key。

### 同步全部

`POST /api/models/sync`

同步所有启用供应商下的所有启用 Key。

每次 Key 同步成功后，后端会把新返回的模型 ID 集合与同步前缓存的模型 ID 集合比较。只要存在新增或移除模型，就会写入一条模型变动记录；模型 ID 集合完全一致时不会重复写记录。首次从空缓存同步到非空列表会记录为新增模型。

## 模型变动记录 API

### 时间轴分页查询

`GET /api/model-changes`

查询参数：

- `limit`：每页数量，默认 `20`，取值范围 `1` 到 `100`。
- `before_id`：可选。传入上一页响应的 `next_before_id` 后，按从新到旧继续加载更早记录。

响应按同步批次 `id` 从大到小返回，不提供全量读取接口，管理后台应按需加载下一页。每个批次是一条时间轴项目，批次内已经按供应商和 Key 组织好，前端不需要再按时间或供应商自行聚合。

```json
{
  "groups": [
    {
      "id": 12,
      "created_at": "2026-06-20T10:05:00.000Z",
      "added_count": 1,
      "removed_count": 2,
      "providers": [
        {
          "provider_id": 1,
          "provider_name": "OpenAI",
          "base_url": "https://api.openai.com",
          "added_count": 1,
          "removed_count": 2,
          "keys": [
            {
              "id": 33,
              "key_id": 3,
              "key_name": "生产 Key",
              "added_count": 1,
              "removed_count": 2,
              "added_models": ["gpt-4.1"],
              "removed_models": ["gpt-4o", "gpt-4o-mini"],
              "created_at": "2026-06-20T10:05:00.000Z"
            }
          ]
        }
      ]
    }
  ],
  "has_more": true,
  "next_before_id": 12
}
```

`next_before_id` 指向下一页之前的同步批次 ID。`provider_name`、`key_name` 和 `base_url` 是写入事件时的快照，用于后台时间轴展示。当前比较口径是模型 ID 集合的新增/移除，不比较 `owned_by` 或原始模型 JSON 的元数据变化。

## 模型列表 API

### 默认按 Key 分组

`GET /api/models`

等价于：

`GET /api/models?mode=by_key`

响应：

```json
{
  "mode": "by_key",
  "providers": [
    {
      "id": 1,
      "name": "OpenAI",
      "base_url": "https://api.openai.com",
      "note": "主供应商",
      "keys": [
        {
          "id": 1,
          "name": "生产 Key",
          "note": "高额度",
          "last_sync_at": "2026-06-20T10:00:00.000Z",
          "last_sync_error": "",
          "models": [
            {
              "id": "gpt-4o",
              "owned_by": "openai"
            }
          ]
        }
      ]
    }
  ]
}
```

如果同一供应商下两个 Key 都支持 `gpt-4o`，`by_key` 会在两个 Key 下各显示一次。

### 汇总去重

`GET /api/models?mode=merged`

同一供应商下按模型 ID 去重：

```json
{
  "mode": "merged",
  "providers": [
    {
      "id": 1,
      "name": "OpenAI",
      "base_url": "https://api.openai.com",
      "note": "主供应商",
      "models": [
        {
          "id": "gpt-4o",
          "owned_by": "openai"
        }
      ]
    }
  ]
}
```

### 强制刷新

列表和搜索接口都支持 `refresh=true`：

`GET /api/models?refresh=true`

后端会先同步所有启用供应商和启用 Key，再返回缓存查询结果。

## 模型搜索 API

`GET /api/models/search?q=gpt`

默认 `mode=by_key`，大小写不敏感匹配模型 ID。响应结构与模型列表一致，并额外返回 `query`：

```json
{
  "mode": "by_key",
  "query": "gpt",
  "providers": [
    {
      "id": 1,
      "name": "OpenAI",
      "base_url": "https://api.openai.com",
      "note": "主供应商",
      "keys": [
        {
          "id": 1,
          "name": "Key 1",
          "note": "高额度",
          "last_sync_at": "2026-06-20T10:00:00.000Z",
          "last_sync_error": "",
          "models": [
            {
              "id": "gpt-4o",
              "owned_by": "openai"
            }
          ]
        },
        {
          "id": 2,
          "name": "Key 2",
          "note": "备用",
          "last_sync_at": "2026-06-20T10:01:00.000Z",
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

搜索也支持汇总模式：

`GET /api/models/search?q=gpt&mode=merged`
