import { Link, Outlet, useNavigate } from 'react-router-dom'
import { useAuth, initial, avatarColor } from './AuthContext'
import { useState } from 'react'
import { api } from './api'

export default function Layout() {
  const { user, admin, loading } = useAuth()
  const [searchQ, setSearchQ] = useState('')
  const [dropdownOpen, setDropdownOpen] = useState(false)
  const navigate = useNavigate()

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    const q = searchQ.trim()
    if (q) navigate(`/search?q=${encodeURIComponent(q)}`)
  }

  const handleLogout = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await api.logout()
      window.location.href = '/'
    } catch {
      window.location.href = '/'
    }
  }

  return (
    <div className="bg-gradient-to-br from-slate-900 to-slate-800 text-white min-h-screen flex flex-col">
      <div className="fixed inset-0 overflow-hidden opacity-20 pointer-events-none z-0" aria-hidden="true">
        <div className="absolute -top-24 -left-24 w-96 h-96 bg-indigo-600 rounded-full blur-3xl mix-blend-multiply filter" />
        <div className="absolute top-1/2 right-0 w-72 h-72 bg-purple-600 rounded-full blur-3xl mix-blend-multiply filter" />
        <div className="absolute -bottom-32 left-1/3 w-80 h-80 bg-pink-600 rounded-full blur-3xl mix-blend-multiply filter" />
      </div>
      <nav className="relative z-10 w-full min-w-0 bg-white shadow-sm sticky top-0 z-50 border-b border-gray-100">
        <div className="container mx-auto px-4 py-3 flex justify-between items-center min-w-0">
          <Link to="/" className="text-base sm:text-xl font-bold text-indigo-600 tracking-tight flex items-center gap-1.5 sm:gap-2 shrink-0 mr-3 sm:mr-2">
            <svg xmlns="http://www.w3.org/2000/svg" className="h-8 w-8 sm:h-6 sm:w-6 shrink-0" fill="none" viewBox="2 4 20 14" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.2" d="M3.5 4.5h17v11h-17v-11z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1" d="M10.5 7.8l4 2.2-4 2.2V7.8z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="0.9" d="M5.5 13.8h13" />
              <circle cx="7.5" cy="13.8" r="0.6" fill="currentColor" stroke="none" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.2" d="M8.5 15.5l-2 1.2" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.2" d="M15.5 15.5l2 1.2" />
            </svg>
            Also Wrote
          </Link>
          <div className="flex items-center gap-2 sm:gap-4 min-w-0 flex-1 justify-end">
            <form onSubmit={handleSearch} className="flex items-center min-w-0 flex-1 sm:flex-initial max-w-[200px] sm:max-w-none">
              <div className="relative w-full min-w-0">
                <input
                  type="text"
                  value={searchQ}
                  onChange={(e) => setSearchQ(e.target.value)}
                  placeholder="Search shows/writers"
                  className="w-full min-w-0 pl-3 pr-8 py-1.5 bg-gray-100 border-transparent rounded-lg text-sm text-gray-900 placeholder-gray-500 focus:ring-2 focus:ring-indigo-500 focus:bg-white focus:border-indigo-500 transition-colors md:w-64 outline-none"
                />
                <button type="submit" className="absolute right-2 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-indigo-600">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                </button>
              </div>
            </form>
            {!loading &&
              (user ? (
                <div className="relative group">
                  <button
                    type="button"
                    onClick={() => setDropdownOpen(!dropdownOpen)}
                    className="flex items-center gap-1.5 pl-1 pr-2 py-1.5 rounded-full hover:bg-gray-100 text-gray-700 transition"
                  >
                    <span className={`flex items-center justify-center w-8 h-8 rounded-full text-white font-semibold text-sm shrink-0 ${avatarColor(user?.email)}`}>
                      {initial(user?.email)}
                    </span>
                    <svg xmlns="http://www.w3.org/2000/svg" className={`h-4 w-4 text-gray-400 shrink-0 transition-transform ${dropdownOpen ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
                    </svg>
                  </button>
                  {dropdownOpen && (
                    <>
                      <div className="fixed inset-0 z-40" onClick={() => setDropdownOpen(false)} aria-hidden="true" />
                      <div className="absolute right-0 top-full mt-1 py-1 min-w-[12rem] max-w-[20rem] bg-white rounded-lg shadow-lg border border-gray-100 z-50">
                        <Link to="/favorite-writers" className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 hover:text-indigo-600" onClick={() => setDropdownOpen(false)}>
                          Favorite Writers
                        </Link>
                        {admin && (
                          <Link to="/admin" className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 hover:text-indigo-600" onClick={() => setDropdownOpen(false)}>
                            Admin
                          </Link>
                        )}
                        <form onSubmit={handleLogout} className="border-t border-gray-100">
                          <button type="submit" className="block w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 hover:text-red-600">
                            Log out <span className="text-gray-500 font-normal">({user?.email ?? ''})</span>
                          </button>
                        </form>
                      </div>
                    </>
                  )}
                </div>
              ) : (
                <Link to="/login" className="text-sm font-medium text-indigo-600 hover:text-indigo-700">
                  Log in
                </Link>
              ))}
          </div>
        </div>
      </nav>
      <main className="relative z-10 flex-grow flex flex-col min-w-0">
        <Outlet />
      </main>
      <footer className="relative z-10 bg-slate-800/90 border-t border-slate-700 py-8">
        <div className="container mx-auto px-4 text-center text-slate-400 text-sm">
          <p className="flex flex-wrap justify-center items-center gap-x-3 gap-y-1">
            <span>Also Wrote by Lena Feldberg &copy; 2026</span>
            <span className="text-slate-500 select-none" aria-hidden="true">·</span>
            <span>
              Data provided by{' '}
              <a href="https://www.themoviedb.org/" target="_blank" rel="noopener noreferrer" className="hover:text-indigo-400 underline">
                TMDB
              </a>
            </span>
            <span className="text-slate-500 select-none" aria-hidden="true">·</span>
            <a href="https://github.com/lena/also-wrote" target="_blank" rel="noopener noreferrer" className="inline-flex items-center hover:text-indigo-400 text-slate-400" aria-label="GitHub repository">
              <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
              </svg>
            </a>
          </p>
        </div>
      </footer>
    </div>
  )
}
