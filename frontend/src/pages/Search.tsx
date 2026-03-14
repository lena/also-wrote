import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { api } from '../api'
import { formatDate } from '../AuthContext'
import type { TVShow, Person } from '../types'

const IMAGE_BASE = 'https://image.tmdb.org/t/p/w500'
const PROFILE_BASE = 'https://image.tmdb.org/t/p/w200'

export default function Search() {
  const [searchParams] = useSearchParams()
  const q = searchParams.get('q') || ''
  const [type, setType] = useState<'shows' | 'people' | 'none' | null>(null)
  const [shows, setShows] = useState<TVShow[]>([])
  const [people, setPeople] = useState<Person[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!q.trim()) {
      setLoading(false)
      setType('none')
      return
    }
    setLoading(true)
    setError(null)
    api
      .search(q)
      .then((d) => {
        setType(d.results_type as 'shows' | 'people' | 'none')
        setShows(d.shows || [])
        setPeople(d.people || [])
        if (d.results_type === 'shows' && d.shows?.length === 1) {
          window.location.href = `/show?id=${d.shows[0].id}`
          return
        }
        if (d.results_type === 'people' && d.people?.length === 1) {
          window.location.href = `/writer?id=${d.people[0].id}`
          return
        }
      })
      .catch((e) => {
        setError(e instanceof Error ? e.message : 'Search failed')
        setType('none')
      })
      .finally(() => setLoading(false))
  }, [q])

  if (!q.trim()) {
    return (
      <div className="container mx-auto px-4 py-8">
        <p className="text-slate-300">Enter a search query.</p>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="animate-pulse text-slate-400">Searching…</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="container mx-auto px-4 py-8">
        <p className="text-red-300">{error}</p>
      </div>
    )
  }

  if (type === 'none') {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="flex flex-col items-center justify-center min-h-[50vh] text-center px-4">
          <div className="bg-slate-700 rounded-full p-6 mb-6">
            <svg xmlns="http://www.w3.org/2000/svg" className="h-16 w-16 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <h2 className="text-3xl font-bold text-white mb-2">No Results Found</h2>
          <p className="text-slate-300 mb-8 max-w-md">
            We couldn&apos;t find any TV shows matching &quot;<strong className="text-white">{q}</strong>&quot;. Try searching for a different title.
          </p>
          <Link to="/" className="bg-indigo-600 text-white px-6 py-3 rounded-lg font-semibold hover:bg-indigo-700 transition shadow-md">
            Back to Search
          </Link>
        </div>
      </div>
    )
  }

  if (type === 'shows') {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="mb-8 border-b border-slate-600 pb-4">
          <h2 className="text-3xl font-bold text-white">
            Results for &quot;<span className="text-indigo-300">{q}</span>&quot;
          </h2>
          <p className="text-slate-400 mt-2">Found {shows.length} shows matching your search.</p>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-5 gap-8">
          {shows.map((show) => (
            <Link to={`/show?id=${show.id}`} key={show.id} className="group block bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden hover:shadow-xl hover:-translate-y-1 transition duration-300">
              <div className="relative overflow-hidden aspect-[2/3]">
                {show.poster_path ? (
                  <img src={IMAGE_BASE + show.poster_path} alt={show.name} className="w-full h-full object-cover group-hover:scale-105 transition duration-500" />
                ) : (
                  <div className="w-full h-full bg-slate-600 flex flex-col items-center justify-center text-slate-400 p-4 text-center">
                    <span className="text-sm font-medium">No Poster</span>
                  </div>
                )}
                <div className="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent opacity-0 group-hover:opacity-100 transition duration-300 flex items-end p-4">
                  <span className="text-white font-semibold text-sm">View Details →</span>
                </div>
              </div>
              <div className="p-4">
                <h3 className="font-bold text-lg mb-1 leading-tight text-gray-900 group-hover:text-indigo-600 transition">{show.name}</h3>
                <p className="text-sm text-gray-500 font-medium">{formatDate(show.first_air_date)}</p>
              </div>
            </Link>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8 border-b border-slate-600 pb-4">
        <h2 className="text-3xl font-bold text-white">
          Results for &quot;<span className="text-indigo-300">{q}</span>&quot;
        </h2>
        <p className="text-slate-400 mt-2">Found {people.length} people matching your search.</p>
      </div>
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-5 gap-8">
        {people.map((person) => (
          <Link to={`/writer?id=${person.id}`} key={person.id} className="group block bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden hover:shadow-xl hover:-translate-y-1 transition duration-300">
            <div className="relative overflow-hidden aspect-[2/3]">
              {person.profile_path ? (
                <img src={PROFILE_BASE + person.profile_path} alt={person.name} className="w-full h-full object-cover group-hover:scale-105 transition duration-500" />
              ) : (
                <div className="w-full h-full bg-slate-600 flex items-center justify-center text-slate-400 text-4xl">?</div>
              )}
            </div>
            <div className="p-4">
              <h3 className="font-bold text-lg leading-tight text-gray-900 group-hover:text-indigo-600 transition">{person.name}</h3>
              {(person.known_for_department || person.department) && (
                <p className="text-sm text-gray-500">{person.known_for_department || person.department}</p>
              )}
            </div>
          </Link>
        ))}
      </div>
    </div>
  )
}
