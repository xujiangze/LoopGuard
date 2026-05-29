import { createContext, useContext, useState, useCallback, type ReactNode } from "react"
import type { Role } from "@/types"
import { getToken, setToken, clearToken, getUser, setUser as saveUser } from "@/lib/auth"
import { api } from "@/lib/api"

interface AuthState {
  userId: number | null
  role: Role | null
  username: string | null
  loggedIn: boolean
}

interface AuthContextValue extends AuthState {
  login: (username: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

function getInitialState(): AuthState {
  const user = getUser()
  const token = getToken()
  if (user && token) {
    return { userId: user.user_id, role: user.role as Role, username: user.username || null, loggedIn: true }
  }
  return { userId: null, role: null, username: null, loggedIn: false }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>(getInitialState)

  const login = useCallback(async (username: string, password: string) => {
    const data = await api.post<{ token: string; role: Role; user_id: number; username: string }>("/auth/login", {
      username,
      password,
    })
    setToken(data.token)
    saveUser({ user_id: data.user_id, role: data.role, username: data.username })
    setState({ userId: data.user_id, role: data.role, username: data.username, loggedIn: true })
  }, [])

  const logout = useCallback(() => {
    clearToken()
    setState({ userId: null, role: null, username: null, loggedIn: false })
  }, [])

  return <AuthContext.Provider value={{ ...state, login, logout }}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth must be used within AuthProvider")
  return ctx
}
