const API_BASE = ''

export interface ApiMe {
  user: { id: number; email: string; created_at: string; login_count: number; last_login_at: string | null } | null
  csrf_token: string
  admin: boolean
}

let csrfToken: string | null = null

export function setCsrfToken(token: string) {
  csrfToken = token
}

export function getCsrfToken(): string | null {
  return csrfToken
}

async function request<T>(
  path: string,
  options: RequestInit & { expectJson?: boolean } = {}
): Promise<T> {
  const { expectJson = true, ...init } = options
  const headers: Record<string, string> = {
    ...(init.headers as Record<string, string>),
  }
  if (init.method && init.method !== 'GET' && csrfToken) {
    headers['X-CSRF-Token'] = csrfToken
    headers['Content-Type'] = headers['Content-Type'] ?? 'application/json'
  }
  const res = await fetch(API_BASE + path, { ...init, headers, credentials: 'include' })
  if (!res.ok) {
    const text = await res.text()
    let err: { error?: string }
    try {
      err = JSON.parse(text)
    } catch {
      err = { error: text || res.statusText }
    }
    throw new Error((err as { error?: string }).error || res.statusText)
  }
  if (expectJson) {
    const data = await res.json()
    return data as T
  }
  return undefined as T
}

export const api = {
  me: () => request<ApiMe>('/api/me'),
  home: () => request<{ suggested_shows: import('./types').TVShow[] }>('/api/home'),
  search: (q: string) =>
    request<{ results_type: string; query: string; shows?: import('./types').TVShow[]; people?: import('./types').Person[] }>(
      `/api/search?q=${encodeURIComponent(q)}`
    ),
  show: (id: number) =>
    request<{ show: import('./types').TVShowDetails; seasons: import('./types').Season[] }>(`/api/show?id=${id}`),
  writer: (id: number) =>
    request<{
      person: import('./types').Person
      credits: import('./types').WriterCredit[]
      is_favorited: boolean
    }>(`/api/writer?id=${id}`),
  episode: (showId: number, season: number, episode: number) =>
    request<{
      episode: import('./types').Episode
      show: import('./types').TVShowDetails | null
      writing_staff: import('./types').AggregateCredit[]
    }>(`/api/episode?show_id=${showId}&season=${season}&episode=${episode}`),
  favoriteWriters: () =>
    request<{ writers: (import('./types').Person | null)[] }>('/api/favorite-writers'),
  overlapGraph: (userId?: number) =>
    request<{ nodes: import('./types').OverlapGraphNode[]; edges: import('./types').OverlapGraphEdge[] }>(
      userId ? `/api/favorite-writers/overlap-graph?user_id=${userId}` : '/api/favorite-writers/overlap-graph'
    ),
  addFavorite: (personId: number) =>
    request<{ ok: boolean }>('/api/favorite-writers', {
      method: 'POST',
      body: JSON.stringify({ person_id: personId }),
    }),
  removeFavorite: (personId: number) =>
    request<{ ok: boolean }>(`/api/favorite-writers/${personId}`, { method: 'DELETE' }),
  adminUsers: () =>
    request<{ users: import('./types').UserWithFavoriteCount[] }>('/api/admin/users'),
  login: (email: string) =>
    request<{ ok: boolean }>('/api/login', { method: 'POST', body: JSON.stringify({ email }) }),
  logout: () => request<{ ok: boolean }>('/api/logout', { method: 'POST' }),
}
