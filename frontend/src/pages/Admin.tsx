import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useAuth } from '../AuthContext'
import { formatPacificTime } from '../AuthContext'
import type { UserWithFavoriteCount } from '../types'
import OverlapGraphModal from '../components/OverlapGraphModal'

export default function Admin() {
  const { user, admin, loading: authLoading } = useAuth()
  const navigate = useNavigate()
  const [users, setUsers] = useState<UserWithFavoriteCount[]>([])
  const [loading, setLoading] = useState(true)
  const [graphUserId, setGraphUserId] = useState<number | null>(null)

  useEffect(() => {
    if (!authLoading && !user) {
      navigate('/login')
      return
    }
    if (!authLoading && user && !admin) {
      navigate('/')
      return
    }
    if (!admin) return
    api
      .adminUsers()
      .then((d) => setUsers(d.users || []))
      .catch(() => setUsers([]))
      .finally(() => setLoading(false))
  }, [user, admin, authLoading, navigate])

  if (authLoading) return <div className="container mx-auto px-4 py-8 animate-pulse text-slate-400">Loading…</div>

  return (
    <div className="container mx-auto px-4 py-8 min-w-0 max-w-full">
      <h1 className="text-4xl font-bold text-white mb-2">Admin</h1>
      <p className="text-slate-300 mb-6">Users who have signed up and their favorite writers count.</p>
      <div className="rounded-xl border border-slate-600 overflow-hidden bg-slate-800/50">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-slate-300">
            <thead className="bg-slate-700/80 text-slate-200 border-b border-slate-600">
              <tr>
                <th className="px-4 py-3 font-semibold">ID</th>
                <th className="px-4 py-3 font-semibold">Email</th>
                <th className="px-4 py-3 font-semibold text-right">Favorite writers</th>
                <th className="px-4 py-3 font-semibold text-right">Logins</th>
                <th className="px-4 py-3 font-semibold">Last login (PT)</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id} className="border-b border-slate-600/80 hover:bg-slate-700/40 transition">
                  <td className="px-4 py-3 font-mono text-slate-400">{u.id}</td>
                  <td className="px-4 py-3">{u.email}</td>
                  <td className="px-4 py-3 text-right">
                    <button
                      type="button"
                      onClick={() => setGraphUserId(u.id)}
                      className="text-indigo-400 hover:text-indigo-300 font-medium focus:outline-none focus:ring-2 focus:ring-indigo-500 rounded"
                      aria-label={`View overlap graph for user ${u.id}`}
                    >
                      {u.favorite_writers_count}
                    </button>
                  </td>
                  <td className="px-4 py-3 text-right">{u.login_count}</td>
                  <td className="px-4 py-3 text-slate-400">{formatPacificTime(u.last_login_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
      {users.length === 0 && !loading && <p className="text-slate-400 mt-4">No users yet.</p>}
      {graphUserId != null && <OverlapGraphModal onClose={() => setGraphUserId(null)} userId={graphUserId} />}
    </div>
  )
}
