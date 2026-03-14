import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import { useAuth } from '../AuthContext'
import { formatDate } from '../AuthContext'
import type { Person, WriterCredit } from '../types'

const IMAGE_BASE = 'https://image.tmdb.org/t/p'
const PROFILE = IMAGE_BASE + '/h632'
const POSTER = IMAGE_BASE + '/w500'

export default function Writer() {
  const [searchParams] = useSearchParams()
  const id = parseInt(searchParams.get('id') || '0', 10)
  const { user } = useAuth()
  const [person, setPerson] = useState<Person | null>(null)
  const [credits, setCredits] = useState<WriterCredit[]>([])
  const [isFavorited, setIsFavorited] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) {
      setError('Invalid writer ID')
      setLoading(false)
      return
    }
    api
      .writer(id)
      .then((d) => {
        setPerson(d.person)
        setCredits(d.credits || [])
        setIsFavorited(d.is_favorited)
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'Not found'))
      .finally(() => setLoading(false))
  }, [id])

  const toggleFavorite = async () => {
    if (!user) {
      window.location.href = '/login'
      return
    }
    try {
      if (isFavorited) {
        await api.removeFavorite(id)
        setIsFavorited(false)
      } else {
        await api.addFavorite(id)
        setIsFavorited(true)
      }
    } catch {
      // ignore
    }
  }

  if (loading) return <div className="container mx-auto px-4 py-8 animate-pulse text-slate-400">Loading…</div>
  if (error || !person) return <div className="container mx-auto px-4 py-8 text-red-300">{error || 'Not found'}</div>

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-12 flex flex-col md:flex-row gap-10 items-start">
        <div className="md:w-64 flex-shrink-0 relative group">
          {person.profile_path && (
            <img src={PROFILE + person.profile_path} alt={person.name} className="rounded-xl shadow-2xl w-full h-auto object-cover transform transition duration-500 group-hover:scale-105" />
          )}
        </div>
        <div className="flex-grow pt-4 flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          <div>
            <h1 className="text-5xl font-extrabold text-white mb-2">{person.name}</h1>
            <div className="flex items-center gap-4 mb-6">
              <span className="bg-indigo-500/30 text-indigo-200 px-3 py-1 rounded-full text-sm font-semibold">
                {person.known_for_department || person.department || 'Writer'}
              </span>
            </div>
          </div>
          <div className="flex items-center">
            {user ? (
              <button
                type="button"
                onClick={toggleFavorite}
                className={`flex items-center gap-2 px-4 py-2 rounded-full border-2 transition font-semibold text-sm duration-200 ${
                  isFavorited ? 'bg-violet-50 border-violet-200/80 text-violet-700 hover:bg-violet-100' : 'bg-gray-50 border-gray-200 text-gray-600 hover:border-indigo-300 hover:bg-indigo-50 hover:text-indigo-600'
                }`}
              >
                <span>{isFavorited ? '💜' : '🩶'}</span>
                <span>{isFavorited ? 'Favorited' : 'Add to favorites'}</span>
              </button>
            ) : (
              <Link
                to="/login"
                className="flex items-center gap-2 px-4 py-2 rounded-full border-2 border-gray-200 bg-gray-50 text-gray-600 hover:border-indigo-300 hover:bg-indigo-50 hover:text-indigo-600 transition font-semibold text-sm duration-200"
              >
                <span>🩶</span>
                <span>Add to favorites</span>
              </Link>
            )}
          </div>
        </div>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
        <div className="bg-gray-50 px-8 py-6 border-b border-gray-100">
          <h2 className="text-2xl font-bold text-gray-800">Writing Credits</h2>
          <p className="text-gray-500 mt-1">Shows written by {person.name}</p>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 p-8 gap-8">
          {credits.map((credit) => (
            <div key={credit.credit_id || credit.id} className="flex flex-col h-full bg-white rounded-lg border border-gray-100 hover:border-indigo-300 hover:shadow-lg transition duration-300 group overflow-hidden">
              <Link to={`/show?id=${credit.id}`} className="block relative overflow-hidden aspect-video bg-gray-100">
                {credit.poster_path ? (
                  <>
                    <img src={POSTER + credit.poster_path} alt="" className="object-cover w-full h-full opacity-90 group-hover:opacity-100 transition duration-500" />
                    <div className="absolute inset-0 bg-gradient-to-t from-black/80 to-transparent flex items-end p-4">
                      <h3 className="text-white font-bold text-lg leading-tight shadow-black drop-shadow-md">{credit.name}</h3>
                    </div>
                  </>
                ) : (
                  <div className="flex items-center justify-center h-full w-full p-4 text-center">
                    <h3 className="font-bold text-lg text-gray-900">{credit.name}</h3>
                  </div>
                )}
              </Link>
              <div className="p-5 flex-grow flex flex-col">
                <div className="flex flex-wrap gap-2 mb-3">
                  <span className="bg-indigo-50 text-indigo-700 text-xs px-2 py-1 rounded font-semibold">{credit.job}</span>
                  <span className="bg-gray-100 text-gray-600 text-xs px-2 py-1 rounded">{credit.episode_count} Episodes</span>
                </div>
                <div className="flex items-center text-xs text-gray-400 mb-3">
                  <span>First Aired: {formatDate(credit.first_air_date)}</span>
                </div>
                {credit.overview && <p className="text-sm text-gray-500 line-clamp-2 mb-4">{credit.overview}</p>}
                {credit.episodes && credit.episodes.length > 0 && (
                  <div className="mt-4 border-t pt-3 flex-grow">
                    <p className="text-xs font-bold text-gray-500 uppercase mb-2">Credited Episodes:</p>
                    <ul className="space-y-1 max-h-48 overflow-y-auto episode-list-scroll pr-2">
                      {credit.episodes.map((ep) => (
                        <li key={ep.id}>
                          <Link
                            to={`/episode?show_id=${credit.id}&season=${ep.season_number}&episode=${ep.episode_number}`}
                            className="text-sm text-gray-700 truncate hover:text-indigo-600 transition flex items-center"
                          >
                            S{ep.season_number} E{ep.episode_number}: {ep.name}
                          </Link>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
