import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useAuth } from '../AuthContext'
import type { Person } from '../types'
import OverlapGraphModal from '../components/OverlapGraphModal'

const PROFILE_BASE = 'https://image.tmdb.org/t/p/h632'

export default function FavoriteWriters() {
  const { user, loading: authLoading } = useAuth()
  const navigate = useNavigate()
  const [writers, setWriters] = useState<(Person | null)[]>([])
  const [loading, setLoading] = useState(true)
  const [graphOpen, setGraphOpen] = useState(false)

  useEffect(() => {
    if (!authLoading && !user) {
      navigate('/login')
      return
    }
    if (!user) return
    api
      .favoriteWriters()
      .then((d) => setWriters(d.writers.filter(Boolean) as Person[]))
      .catch(() => setWriters([]))
      .finally(() => setLoading(false))
  }, [user, authLoading, navigate])

  if (authLoading) return <div className="container mx-auto px-4 py-8 animate-pulse text-slate-400">Loading…</div>

  return (
    <div className="container mx-auto px-4 py-8 min-w-0 max-w-full">
      <h1 className="text-4xl font-bold text-white mb-2">Favorite Writers</h1>
      <p className="text-slate-300 mb-6">Writers you&apos;ve favorited. Click to visit their page.</p>
      {writers.length > 0 && (
        <div className="mb-8">
          <button
            type="button"
            onClick={() => setGraphOpen(true)}
            className="px-4 py-2 rounded-full border-2 border-indigo-200 bg-indigo-50 text-indigo-700 hover:bg-indigo-100 hover:border-indigo-300 transition font-semibold text-sm"
          >
            See overlap graph
          </button>
        </div>
      )}
      {loading ? (
        <div className="animate-pulse text-slate-400">Loading writers…</div>
      ) : writers.length > 0 ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-6">
          {writers.filter((w): w is Person => w != null).map((w) => (
            <Link to={`/writer?id=${w.id}`} key={w.id} className="group block text-center">
              <div className="aspect-[2/3] rounded-xl overflow-hidden bg-slate-700 mb-3 shadow-md group-hover:shadow-lg transition">
                {w.profile_path ? (
                  <img src={PROFILE_BASE + w.profile_path} alt={w.name} className="w-full h-full object-cover group-hover:scale-105 transition duration-300" />
                ) : (
                  <div className="w-full h-full flex items-center justify-center text-slate-400 text-4xl">?</div>
                )}
              </div>
              <span className="font-semibold text-white group-hover:text-indigo-300 transition">{w.name}</span>
            </Link>
          ))}
        </div>
      ) : (
        <div className="bg-white/10 rounded-xl border border-slate-600 p-12 text-center">
          <p className="text-slate-300 mb-4">You haven&apos;t favorited any writers yet.</p>
          <Link to="/" className="text-indigo-400 font-semibold hover:text-indigo-300">
            Discover writers →
          </Link>
        </div>
      )}
      {graphOpen && writers.length > 0 && <OverlapGraphModal onClose={() => setGraphOpen(false)} />}
    </div>
  )
}
