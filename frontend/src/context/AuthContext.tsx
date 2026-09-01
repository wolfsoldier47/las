import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import api from '../api/client'

const TOKEN_KEY = 'ulas_token'
const USERNAME_KEY = 'ulas_username'
const USER_INFO_KEY = 'ulas_user_info'

export interface AuthUserInfo {
  [key: string]: string
}

interface AuthUser {
  token: string
  username: string
  info: AuthUserInfo
}

interface AuthContextValue {
  user: AuthUser | null
  loading: boolean
  error: string | null
  login: (username: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

function loadUserInfo(): AuthUserInfo {
  try {
    const raw = localStorage.getItem(USER_INFO_KEY)
    return raw ? (JSON.parse(raw) as AuthUserInfo) : {}
  } catch {
    return {}
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const token = localStorage.getItem(TOKEN_KEY)
    const storedUsername = localStorage.getItem(USERNAME_KEY)

    if (!token) {
      setLoading(false)
      return
    }

    // Validate the stored token before showing the application.
    api
      .get('/me')
      .then((res) => {
        const username: string = res.data.username || storedUsername || ''
        localStorage.setItem(USERNAME_KEY, username)
        setUser({ token, username, info: loadUserInfo() })
      })
      .catch(() => {
        localStorage.removeItem(TOKEN_KEY)
        localStorage.removeItem(USERNAME_KEY)
        localStorage.removeItem(USER_INFO_KEY)
        setUser(null)
      })
      .finally(() => {
        setLoading(false)
      })
  }, [])

  const login = async (username: string, password: string) => {
    setError(null)
    try {
      const res = await api.post('/login', { username, password })
      const accessToken: string = res.data.access_token
      const returnedUsername: string = res.data.username || username
      const userInfo: AuthUserInfo = res.data.user_info || {}
      if (!accessToken) {
        throw new Error('No access token received')
      }
      localStorage.setItem(TOKEN_KEY, accessToken)
      localStorage.setItem(USERNAME_KEY, returnedUsername)
      localStorage.setItem(USER_INFO_KEY, JSON.stringify(userInfo))
      setUser({ token: accessToken, username: returnedUsername, info: userInfo })
    } catch (err: any) {
      const message = err.response?.data?.error || err.message || 'Login failed'
      setError(message)
      throw new Error(message)
    }
  }

  const logout = () => {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USERNAME_KEY)
    localStorage.removeItem(USER_INFO_KEY)
    setUser(null)
    window.location.href = '/login'
  }

  return (
    <AuthContext.Provider value={{ user, loading, error, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return ctx
}
