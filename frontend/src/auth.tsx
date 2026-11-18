import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, TOKEN_KEY, type User } from './api'

type AuthState = {
  user: User | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (email: string, name: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(() => localStorage.getItem(TOKEN_KEY) !== null)

  useEffect(() => {
    if (!localStorage.getItem(TOKEN_KEY)) {
      return
    }
    api
      .me()
      .then(setUser)
      .catch(() => {
        localStorage.removeItem(TOKEN_KEY)
        setUser(null)
      })
      .finally(() => setLoading(false))
  }, [])

  async function login(email: string, password: string) {
    const res = await api.login(email, password)
    localStorage.setItem(TOKEN_KEY, res.token)
    setUser(res.user)
  }

  async function register(email: string, name: string, password: string) {
    const res = await api.register(email, name, password)
    localStorage.setItem(TOKEN_KEY, res.token)
    setUser(res.user)
  }

  function logout() {
    localStorage.removeItem(TOKEN_KEY)
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return ctx
}
