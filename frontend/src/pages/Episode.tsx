import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import { formatDate } from '../AuthContext'
import type { Episode as EpisodeType, TVShowDetails, AggregateCredit } from '../types'

const IMAGE_BASE = 'https://image.tmdb.org/t/p'
const STILL = IMAGE_BASE + '/original'
const POSTER = IMAGE_BASE + '/w500'
const PROFILE = IMAGE_BASE + '/w200'
const writerJobs = ['Writer', 'Screenplay', 'Teleplay', 'Story']

export default function Episode() {
  const [searchParams] = useSearchParams()
  const showId = parseInt(searchParams.get('show_id') || '0', 10)
  const season = parseInt(searchParams.get('season') || '0', 10)
  const episode = parseInt(searchParams.get('episode') || '0', 10)
  const [ep, setEp] = useState<EpisodeType | null>(null)
  const [show, setShow] = useState<TVShowDetails | null>(null)
  const [writingStaff, setWritingStaff] = useState<AggregateCredit[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!showId || !season || !episode) {
      setError('Invalid episode parameters')
      setLoading(false)
      return
    }
    api
      .episode(showId, season, episode)
      .then((d) => {
        setEp(d.episode)
        setShow(d.show || null)
        setWritingStaff(d.writing_staff || [])
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'Not found'))
      .finally(() => setLoading(false))
  }, [showId, season, episode])

  if (loading) return <div className="container mx-auto px-4 py-8 animate-pulse text-slate-400">Loading…</div>
  if (error || !ep) return <div className="container mx-auto px-4 py-8 text-red-300">{error || 'Episode not found'}</div>

  const writers = (ep.crew || []).filter((c) => writerJobs.includes(c.job || ''))
  const directors = (ep.crew || []).filter((c) => c.job === 'Director')

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="text-sm text-slate-400 mb-4 flex items-center gap-2">
        <Link to={`/show?id=${show?.id}`} className="hover:text-indigo-300 transition font-medium text-white">
          {show?.name}
        </Link>
        <svg xmlns="http://www.w3.org/2000/svg" className="h-3 w-3 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5l7 7-7 7" />
        </svg>
        <span>Season {ep.season_number}</span>
        <svg xmlns="http://www.w3.org/2000/svg" className="h-3 w-3 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5l7 7-7 7" />
        </svg>
        <span className="text-white font-semibold">Episode {ep.episode_number}</span>
      </div>

      <div className="relative w-full h-96 bg-gray-900 overflow-hidden mb-8 rounded-2xl shadow-2xl">
        {ep.still_path ? (
          <img src={STILL + ep.still_path} alt={ep.name} className="w-full h-full object-contain object-top opacity-40" />
        ) : (
          <div className="w-full h-full bg-gradient-to-r from-gray-900 to-indigo-900 opacity-80" />
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-gray-900 via-transparent to-transparent" />
        <div className="absolute bottom-0 left-0 p-8 w-full md:w-2/3">
          <h1 className="text-4xl md:text-6xl font-extrabold text-white mb-2 drop-shadow-md">{ep.name}</h1>
          <div className="flex items-center gap-4 text-gray-300 text-sm md:text-base mb-4">
            <span className="bg-indigo-600 text-white px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider">
              S{ep.season_number} E{ep.episode_number}
            </span>
            <span>Aired {formatDate(ep.air_date)}</span>
            {show && (
              <Link to={`/show?id=${show.id}`} className="hover:text-indigo-300 transition">
                {show.name}
              </Link>
            )}
          </div>
          <p className="text-gray-200 line-clamp-3 text-lg drop-shadow-sm max-w-3xl">{ep.overview}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-10">
        <div className="lg:col-span-8 space-y-8">
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
            <div className="bg-gray-50 px-6 py-4 border-b border-gray-100">
              <h2 className="text-2xl font-bold text-gray-800">Writers & Directors</h2>
              <p className="text-gray-500 text-sm mt-0.5">Episode credits</p>
            </div>
            <div className="p-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                <div>
                  <h3 className="text-lg font-bold text-gray-900 mb-4 border-b border-gray-100 pb-2">Writers</h3>
                  <div className="space-y-3">
                    {writers.map((c) => (
                      <Link to={`/writer?id=${c.id}`} key={c.id} className="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-50 transition group -mx-2">
                        {c.profile_path ? (
                          <img src={PROFILE + c.profile_path} alt="" className="w-10 h-10 rounded-full object-cover shadow-sm group-hover:ring-2 ring-indigo-500 transition" />
                        ) : (
                          <div className="w-10 h-10 rounded-full bg-indigo-100 flex items-center justify-center text-indigo-500 font-bold text-sm">{c.name?.slice(0, 1) || '?'}</div>
                        )}
                        <div>
                          <p className="font-bold text-gray-900 group-hover:text-indigo-600 transition">{c.name}</p>
                          <p className="text-xs text-gray-500">{c.job}</p>
                        </div>
                      </Link>
                    ))}
                  </div>
                </div>
                <div>
                  <h3 className="text-lg font-bold text-gray-900 mb-4 border-b border-gray-100 pb-2">Directors</h3>
                  <div className="space-y-3">
                    {directors.map((c) => (
                      <Link to={`/writer?id=${c.id}`} key={c.id} className="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-50 transition group -mx-2">
                        {c.profile_path ? (
                          <img src={PROFILE + c.profile_path} alt="" className="w-10 h-10 rounded-full object-cover shadow-sm group-hover:ring-2 ring-indigo-500 transition" />
                        ) : (
                          <div className="w-10 h-10 rounded-full bg-indigo-100 flex items-center justify-center text-indigo-500 font-bold text-sm">{c.name?.slice(0, 1) || '?'}</div>
                        )}
                        <div>
                          <p className="font-bold text-gray-900 group-hover:text-indigo-600 transition">{c.name}</p>
                          <p className="text-xs text-gray-500">Director</p>
                        </div>
                      </Link>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
            <div className="bg-gray-50 px-6 py-4 border-b border-gray-100 flex justify-between items-center">
              <h2 className="text-2xl font-bold text-gray-800">Season Writing Staff</h2>
              <span className="text-gray-500 text-sm">{writingStaff.length} writers</span>
            </div>
            <div className="p-6">
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                {writingStaff.map((c) => (
                  <Link to={`/writer?id=${c.id}`} key={c.id} className="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-50 transition group -mx-2">
                    {c.profile_path ? (
                      <img src={PROFILE + c.profile_path} alt="" className="w-10 h-10 rounded-full object-cover shadow-sm group-hover:ring-2 ring-indigo-500 transition" />
                    ) : (
                      <div className="w-10 h-10 rounded-full bg-indigo-100 flex items-center justify-center text-indigo-500 font-bold text-sm">{c.name?.slice(0, 1) || '?'}</div>
                    )}
                    <div className="min-w-0">
                      <p className="font-bold text-gray-900 group-hover:text-indigo-600 transition text-sm truncate">{c.name}</p>
                      <p className="text-xs text-gray-500 truncate max-w-[12rem]">{c.jobs?.map((j) => j.job).join(', ')}</p>
                    </div>
                  </Link>
                ))}
              </div>
            </div>
          </div>
        </div>

        <div className="lg:col-span-4 space-y-8">
          {show && (
            <div className="bg-white p-6 rounded-xl shadow-sm border border-gray-100 sticky top-24">
              <h3 className="text-lg font-bold mb-4 text-gray-800 border-b border-gray-100 pb-2">Show</h3>
              <Link to={`/show?id=${show.id}`} className="flex flex-col gap-3 group">
                {show.backdrop_path ? (
                  <img src={POSTER + show.backdrop_path} alt={show.name} className="w-full aspect-video object-cover rounded-lg shadow-sm group-hover:ring-2 ring-indigo-500 transition" />
                ) : (
                  <div className="w-full aspect-video bg-gray-200 rounded-lg flex items-center justify-center text-gray-400 text-sm">No image</div>
                )}
                <div>
                  <p className="font-bold text-gray-900 group-hover:text-indigo-600 transition">{show.name}</p>
                  <p className="text-xs text-gray-500">Back to series</p>
                </div>
              </Link>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
