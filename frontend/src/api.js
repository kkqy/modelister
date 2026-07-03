// 与 Modelister 后端 REST API 的唯一对接层。
// 同源部署，统一携带 HttpOnly Cookie（credentials: "include"）。

const BASE = "/api";

// 后端错误结构：{ "error": { "code", "message" } }
export class ApiError extends Error {
  constructor(status, code, message) {
    super(message || code || `HTTP ${status}`);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

async function request(path, { method = "GET", body } = {}) {
  const opts = {
    method,
    credentials: "include",
    headers: {},
  };
  if (body !== undefined) {
    opts.headers["Content-Type"] = "application/json";
    opts.body = JSON.stringify(body);
  }

  let res;
  try {
    res = await fetch(`${BASE}${path}`, opts);
  } catch (networkErr) {
    throw new ApiError(0, "network_error", "无法连接服务器，请检查网络或后端状态");
  }

  if (res.status === 204) {
    return null;
  }

  let data = null;
  const text = await res.text();
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = null;
    }
  }

  if (!res.ok) {
    const detail = data && data.error ? data.error : {};
    throw new ApiError(res.status, detail.code, detail.message);
  }
  return data;
}

export const api = {
  // 认证
  login: (username, password) =>
    request("/auth/login", { method: "POST", body: { username, password } }),
  me: () => request("/auth/me"),
  logout: () => request("/auth/logout", { method: "POST" }),

  // 供应商
  listProviders: () => request("/providers"),
  createProvider: (payload) => request("/providers", { method: "POST", body: payload }),
  updateProvider: (id, payload) =>
    request(`/providers/${id}`, { method: "PUT", body: payload }),
  deleteProvider: (id) => request(`/providers/${id}`, { method: "DELETE" }),

  // 供应商 Key
  listKeys: (providerId) => request(`/providers/${providerId}/keys`),
  createKey: (providerId, payload) =>
    request(`/providers/${providerId}/keys`, { method: "POST", body: payload }),
  updateKey: (providerId, keyId, payload) =>
    request(`/providers/${providerId}/keys/${keyId}`, { method: "PUT", body: payload }),
  deleteKey: (providerId, keyId) =>
    request(`/providers/${providerId}/keys/${keyId}`, { method: "DELETE" }),

  // 同步
  syncKey: (providerId, keyId) =>
    request(`/providers/${providerId}/keys/${keyId}/sync`, { method: "POST" }),
  syncProvider: (providerId) =>
    request(`/providers/${providerId}/sync`, { method: "POST" }),
  syncAll: () => request("/models/sync", { method: "POST" }),

  // 模型
  listModels: ({ mode = "by_key", refresh = false } = {}) => {
    const params = new URLSearchParams({ mode });
    if (refresh) params.set("refresh", "true");
    return request(`/models?${params.toString()}`);
  },
  searchModels: ({ q, mode = "by_key", refresh = false }) => {
    const params = new URLSearchParams({ q, mode });
    if (refresh) params.set("refresh", "true");
    return request(`/models/search?${params.toString()}`);
  },

  // 模型变动记录
  listModelChanges: ({ limit = 20, beforeId } = {}) => {
    const params = new URLSearchParams({ limit: String(limit) });
    if (beforeId) params.set("before_id", String(beforeId));
    return request(`/model-changes?${params.toString()}`);
  },
};
