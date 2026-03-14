import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { api, setCsrfToken, type ApiMe } from './api'

interface AuthState {
  user: ApiMe['user']
  csrfToken: string
  admin: boolean
  loading: boolean
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<ApiMe['user']>(null)
  const [csrfToken, setCsrf] = useState('')
  const [admin, setAdmin] = useState(false)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      const data = await api.me()
      setUser(data.user)
      setCsrf(data.csrf_token)
      setAdmin(data.admin)
      setCsrfToken(data.csrf_token)
    } catch {
      setUser(null)
      setCsrf('')
      setAdmin(false)
      setCsrfToken('')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  return (
    <AuthContext.Provider value={{ user, csrfToken, admin, loading, refresh }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

export function initial(s: string | null | undefined): string {
  const t = (s ?? '').trim()
  if (!t) return '?'
  return t[0].toUpperCase()
}

export function avatarColor(s: string | null | undefined): string {
  const str = s ?? ''
  const colors = ['bg-purple-500', 'bg-fuchsia-500']
  let h = 0
  for (let i = 0; i < str.length; i++) h += str.charCodeAt(i)
  if (h < 0) h = -h
  return colors[h % colors.length]
}

export function formatDate(s: string): string {
  if (!s) return s
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

export function formatPacificTime(t: string | null | undefined): string {
  if (t == null) return '—'
  const d = new Date(t)
  if (isNaN(d.getTime())) return '—'
  return d.toLocaleString('en-US', {
    timeZone: 'America/Los_Angeles',
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })
}
