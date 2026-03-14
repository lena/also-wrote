import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import { formatDate } from '../AuthContext'
import type { TVShowDetails, Season } from '../types'

const IMAGE_BASE = 'https://image.tmdb.org/t/p'
const BACKDROP = IMAGE_BASE + '/original'
const STILL = IMAGE_BASE + '/w300'

export default function Show() {
  const [searchParams] = useSearchParams()
  const id = parseInt(searchParams.get('id') || '0', 10)
  const [show, setShow] = useState<TVShowDetails | null>(null)
  const [seasons, setSeasons] = useState<Season[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) {
      setError('Invalid show ID')
      setLoading(false)
      return
    }
    api
      .show(id)
      .then((d) => {
        setShow(d.show)
        setSeasons(d.seasons || [])
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'Show not found'))
      .finally(() => setLoading(false))
  }, [id])

  if (loading) return <div className="container mx-auto px-4 py-8 animate-pulse text-slate-400">Loading…</div>
  if (error || !show) return <div className="container mx-auto px-4 py-8 text-red-300">{error || 'Show not found'}</div>

  const writerJobs = ['Writer', 'Screenplay', 'Teleplay', 'Story']

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="relative w-full h-96 bg-gray-900 overflow-hidden mb-8 rounded-2xl shadow-2xl">
        {show.backdrop_path ? (
          <img src={BACKDROP + show.backdrop_path} alt="" className="w-full h-full object-cover opacity-40" />
        ) : (
          <div className="w-full h-full bg-gradient-to-r from-gray-900 to-indigo-900 opacity-80" />
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-gray-900 via-transparent to-transparent" />
        <div className="absolute bottom-0 left-0 p-8 w-full md:w-2/3">
          <h1 className="text-4xl md:text-6xl font-extrabold text-white mb-2 drop-shadow-md">{show.name}</h1>
          <div className="flex items-center gap-4 text-gray-300 text-sm md:text-base mb-4">
            <span className="bg-indigo-600 text-white px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider">Series</span>
            <span>{show.number_of_seasons} Seasons</span>
            <span>Started {formatDate(show.first_air_date)}</span>
          </div>
          <p className="text-gray-200 line-clamp-3 text-lg drop-shadow-sm max-w-3xl">{show.overview}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-10">
        <div className="lg:col-span-8 space-y-12">
          {seasons.map((season) => (
            <div key={season.id} className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
              <div className="bg-gray-50 px-6 py-4 border-b border-gray-100 flex justify-between items-center">
                <h2 className="text-2xl font-bold text-gray-800">Season {season.season_number}</h2>
                <span className="text-gray-500 text-sm">{season.episodes?.length ?? 0} Episodes</span>
              </div>
              <div className="divide-y divide-gray-100">
                {(season.episodes || []).map((ep) => (
                  <div key={ep.id} className="p-6 hover:bg-gray-50 transition duration-150">
                    <div className="flex flex-col md:flex-row gap-6">
                      <div className="md:w-48 flex-shrink-0 relative group">
                        <Link to={`/episode?show_id=${show.id}&season=${ep.season_number}&episode=${ep.episode_number}`} className="block">
                          {ep.still_path ? (
                            <img src={STILL + ep.still_path} alt={ep.name} className="rounded-lg shadow-md w-full h-28 object-cover transition transform group-hover:scale-105 duration-300" />
                          ) : (
                            <div className="w-full h-28 bg-gray-200 rounded-lg flex items-center justify-center text-xs text-gray-400">No Image</div>
                          )}
                          <div className="absolute top-2 right-2 bg-black/70 text-white text-xs px-1.5 py-0.5 rounded font-mono">E{ep.episode_number}</div>
                        </Link>
                      </div>
                      <div className="flex-grow">
                        <div className="flex justify-between items-start mb-2">
                          <Link to={`/episode?show_id=${show.id}&season=${ep.season_number}&episode=${ep.episode_number}`} className="hover:underline">
                            <h3 className="font-bold text-lg text-gray-900">{ep.name}</h3>
                          </Link>
                          <span className="text-xs text-gray-500 font-medium bg-gray-100 px-2 py-1 rounded">{formatDate(ep.air_date)}</span>
                        </div>
                        <p className="text-gray-600 text-sm mb-4 line-clamp-2 leading-relaxed">{ep.overview}</p>
                        <div className="flex flex-wrap gap-2 items-center">
                          <span className="text-xs font-bold text-gray-400 uppercase tracking-wider mr-1">Written by</span>
                          {(ep.crew || []).filter((c) => c.id > 0 && writerJobs.includes(c.job || '')).map((c) => (
                            <Link to={`/writer?id=${c.id}`} key={c.credit_id ?? c.id} className="inline-flex items-center px-2.5 py-1 bg-indigo-50 text-indigo-700 text-xs font-semibold rounded-full hover:bg-indigo-100 hover:text-indigo-800 transition border border-indigo-100">
                              {c.name}
                              {c.job && c.job !== 'Writer' && <span className="ml-1 text-indigo-400 font-normal">({c.job})</span>}
                            </Link>
                          ))}
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        <div className="lg:col-span-4 space-y-8">
          <div className="bg-white p-6 rounded-xl shadow-sm border border-gray-100 sticky top-24">
            <h3 className="text-lg font-bold mb-4 text-gray-800 border-b pb-2">Show Creators</h3>
            <div className="space-y-4">
              {(show.created_by || []).length > 0 ? (
                show.created_by.map((c) => (
                  <Link to={`/writer?id=${c.id}`} key={c.id} className="flex items-center gap-3 group hover:bg-gray-50 p-2 rounded-lg transition -mx-2">
                    {c.profile_path ? (
                      <img src={IMAGE_BASE + '/w200' + c.profile_path} alt="" className="w-12 h-12 rounded-full object-cover shadow-sm group-hover:ring-2 ring-indigo-500 transition" />
                    ) : (
                      <div className="w-12 h-12 rounded-full bg-indigo-100 flex items-center justify-center text-indigo-500 font-bold text-lg">?</div>
                    )}
                    <div>
                      <p className="font-bold text-gray-900 group-hover:text-indigo-600 transition">{c.name}</p>
                      <p className="text-xs text-gray-500">Creator</p>
                    </div>
                  </Link>
                ))
              ) : (
                <p className="text-sm text-gray-500 italic">No creator information available.</p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
