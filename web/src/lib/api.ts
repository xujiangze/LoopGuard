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
    "Content-Type": "application/json",
    ...(init?.headers as Record<string, string>),
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
}
