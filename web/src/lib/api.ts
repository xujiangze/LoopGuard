import { getToken, clearToken } from "./auth"

const BASE_URL = "/api/v1"

export class ApiError extends Error {
  status: number
  constructor(
    status: number,
    message: string,
  ) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    ...(init?.headers as Record<string, string>),
  }
  // upload 时 body 是 FormData，不设 Content-Type（浏览器自动设 boundary）
  if (!(init?.body instanceof FormData)) {
    headers["Content-Type"] = "application/json"
  }
  if (token) {
    headers["Authorization"] = `Bearer ${token}`
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...init, headers })

  if (res.status === 401) {
    clearToken()
    window.location.hash = "#/login"
    throw new ApiError(401, "登录已过期，请重新登录")
  }

  const data = await res.json()
  if (!res.ok) {
    throw new ApiError(res.status, data.error || "请求失败")
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>(path),

  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "POST", body: body ? JSON.stringify(body) : undefined }),

  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PUT", body: body ? JSON.stringify(body) : undefined }),

  del: <T>(path: string) => request<T>(path, { method: "DELETE" }),

  upload: <T>(path: string, formData: FormData) =>
    request<T>(path, { method: "POST", body: formData }),

  uploadPut: <T>(path: string, formData: FormData) =>
    request<T>(path, { method: "PUT", body: formData }),

  getText: async (path: string): Promise<string> => {
    const token = getToken()
    const headers: Record<string, string> = {}
    if (token) headers["Authorization"] = `Bearer ${token}`
    const res = await fetch(`${BASE_URL}${path}`, { headers })
    if (res.status === 401) {
      clearToken()
      window.location.hash = "#/login"
      throw new ApiError(401, "登录已过期，请重新登录")
    }
    if (!res.ok) {
      const data = await res.json().catch(() => ({ error: "请求失败" }))
      throw new ApiError(res.status, data.error || "请求失败")
    }
    return res.text()
  },

  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PATCH", body: body ? JSON.stringify(body) : undefined }),

  // Webhook API
  webhooks: {
    list: (programId?: number) =>
      programId
        ? api.get<import("../types").WebhookConfig[]>(`/webhooks?program_id=${programId}`)
        : api.get<import("../types").WebhookConfig[]>("/webhooks"),

    create: (data: {
      program_id: number
      name: string
      url: string
      enabled: boolean
      event_types: string
    }) => api.post<import("../types").WebhookConfig>("/webhooks", data),

    delete: (id: number) => api.del(`/webhooks/${id}`),

    toggle: (id: number, enabled: boolean) =>
      api.patch<import("../types").WebhookConfig>(`/webhooks/${id}`, { enabled }),

    deliveries: (id: number) =>
      api.get<import("../types").WebhookDelivery[]>(`/webhooks/${id}/deliveries`),
  },
}
