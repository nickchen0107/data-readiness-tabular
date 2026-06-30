import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'
import apiClient from '../api/client'

interface User {
  id: string
  email: string
}

interface AuthState {
  token: string | null
  user: User | null
  isAuthenticated: boolean
}

interface AuthContextType extends AuthState {
  login: (token: string) => Promise<void>
  logout: () => void
  loadUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>(() => {
    const token = localStorage.getItem('token')
    const userStr = localStorage.getItem('user')
    const user = userStr ? JSON.parse(userStr) : null
    return {
      token,
      user,
      isAuthenticated: !!token,
    }
  })

  const loadUser = useCallback(async () => {
    if (!state.token) return
    try {
      const res = await apiClient.get('/auth/me')
      const user: User = { id: res.data.id, email: res.data.email }
      localStorage.setItem('user', JSON.stringify(user))
      setState((prev) => ({ ...prev, user, isAuthenticated: true }))
    } catch {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      setState({ token: null, user: null, isAuthenticated: false })
    }
  }, [state.token])

  const login = useCallback(async (token: string) => {
    localStorage.setItem('token', token)
    setState((prev) => ({ ...prev, token, isAuthenticated: true }))
    // Fetch user info
    try {
      const res = await apiClient.get('/auth/me', {
        headers: { Authorization: `Bearer ${token}` },
      })
      const user: User = { id: res.data.id, email: res.data.email }
      localStorage.setItem('user', JSON.stringify(user))
      setState({ token, user, isAuthenticated: true })
    } catch {
      // Token is set, user fetch failed; still authenticated
      setState((prev) => ({ ...prev, token, isAuthenticated: true }))
    }
  }, [])

  const logout = useCallback(() => {
    apiClient.post('/auth/logout').catch(() => {})
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setState({ token: null, user: null, isAuthenticated: false })
  }, [])

  useEffect(() => {
    if (state.token && !state.user) {
      loadUser()
    }
  }, [state.token, state.user, loadUser])

  return (
    <AuthContext.Provider value={{ ...state, login, logout, loadUser }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextType {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
