import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api'
import type { TVShow } from '../types'

const IMAGE_BASE = 'https://image.tmdb.org/t/p/w500'

export default function Home() {
  const navigate = useNavigate()
  const [shows, setShows] = useState<TVShow[]>([])
  const [heroQ, setHeroQ] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api
      .home()
      .then((d) => setShows(d.suggested_shows || []))
      .catch(() => setShows([]))
      .finally(() => setLoading(false))
  }, [])

  return (
    <div className="relative min-h-[calc(100vh-theme(spacing.16)-theme(spacing.16))] flex flex-col items-center justify-center overflow-hidden flex-grow">
      <div className="absolute top-0 left-0 w-full h-full overflow-hidden opacity-20 pointer-events-none">
        <div className="absolute -top-24 -left-24 w-96 h-96 bg-indigo-600 rounded-full blur-3xl mix-blend-multiply filter" />
        <div className="absolute top-1/2 right-0 w-72 h-72 bg-purple-600 rounded-full blur-3xl mix-blend-multiply filter" />
        <div className="absolute -bottom-32 left-1/3 w-80 h-80 bg-pink-600 rounded-full blur-3xl mix-blend-multiply filter" />
      </div>

      <div className="relative z-10 w-full max-w-4xl px-4 py-12 md:py-16 flex flex-col items-center flex-grow justify-center">
        <h1 className="text-5xl md:text-7xl font-black mb-6 text-center tracking-tight leading-tight">
          Who wrote your <br />
          <span className="text-transparent bg-clip-text bg-gradient-to-r from-indigo-400 to-pink-400">favorite episode?</span>
        </h1>

        <p className="text-xl md:text-2xl text-slate-300 mb-12 text-center max-w-2xl font-light">
          Discover the creative minds behind the TV series you love. Search for a show or writer to see episode-by-episode writing credits.
        </p>

        <div className="w-full max-w-2xl mb-16">
          <form
            className="relative group"
            onSubmit={(e) => {
              e.preventDefault()
              const q = heroQ.trim()
              if (q) navigate(`/search?q=${encodeURIComponent(q)}`)
            }}
          >
            <input
              type="text"
              value={heroQ}
              onChange={(e) => setHeroQ(e.target.value)}
              placeholder="Search for a show or writer (e.g., The Wire)"
              className="hero-search-input w-full p-6 pl-8 text-xl text-slate-900 bg-white/95 backdrop-blur-sm border-0 rounded-full shadow-2xl focus:ring-4 focus:ring-indigo-500/50 transition duration-300 placeholder-slate-400 outline-none"
            />
            <button type="submit" className="absolute right-3 top-2 bottom-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-full px-8 font-bold transition duration-300 flex items-center justify-center">
              Search
            </button>
          </form>
        </div>

        <div className="w-full">
          <span className="text-sm font-bold text-slate-400 uppercase tracking-widest mb-6 block text-center">Suggested Searches</span>
          {loading ? (
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
              {[1, 2, 3].map((i) => (
                <div key={i} className="aspect-[2/3] w-full bg-slate-700 rounded-xl animate-pulse" />
              ))}
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
              {shows.map((show) => (
                <Link to={`/show?id=${show.id}`} key={show.id} className="group block relative rounded-xl overflow-hidden shadow-lg hover:shadow-2xl transition duration-500 hover:-translate-y-2">
                  <div className="aspect-[2/3] w-full bg-slate-700 relative">
                    {show.poster_path ? (
                      <img src={IMAGE_BASE + show.poster_path} alt={show.name} className="w-full h-full object-cover transition duration-700 group-hover:scale-110" />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center text-slate-500 p-2 text-center text-xs">No Image</div>
                    )}
                    <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition duration-300 flex items-end p-4">
                      <span className="text-white font-bold text-sm line-clamp-2">{show.name}</span>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
