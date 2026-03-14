import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { api } from '../api'

export default function Login() {
  const [searchParams] = useSearchParams()
  const errorParam = searchParams.get('error')
  const defaultError =
    errorParam === 'missing' || errorParam === 'invalid'
      ? 'Invalid or missing sign-in link. Request a new one below.'
      : errorParam === 'expired'
        ? 'That link has expired. Request a new one below.'
        : errorParam
          ? 'Something went wrong. Please try again.'
          : ''
  const [email, setEmail] = useState('')
  const [error, setError] = useState(defaultError)
  const [submitting, setSubmitting] = useState(false)
  const [sent, setSent] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    const trimmed = email.trim()
    if (!trimmed) {
      setError('Please enter your email.')
      return
    }
    setSubmitting(true)
    try {
      await api.login(trimmed)
      setSent(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong.')
    } finally {
      setSubmitting(false)
    }
  }

  if (sent) {
    return (
      <div className="container mx-auto px-4 py-16 flex flex-col items-center justify-center min-h-[60vh]">
        <div className="w-full max-w-md bg-white rounded-xl shadow-lg border border-gray-100 p-8 text-center">
          <h1 className="text-2xl font-bold text-gray-900 mb-2">Check your email</h1>
          <p className="text-gray-500 mb-6">
            We sent a magic link to <strong>{email}</strong>. Click the link to sign in.
          </p>
          <Link to="/" className="text-indigo-600 font-semibold hover:text-indigo-700">
            ← Back to home
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-16 flex flex-col items-center justify-center min-h-[60vh]">
      <div className="w-full max-w-md bg-white rounded-xl shadow-lg border border-gray-100 p-8">
        <h1 className="text-2xl font-bold text-gray-900 mb-2">Sign in</h1>
        <p className="text-gray-500 mb-6">Enter your email and we&apos;ll send you a magic link to sign in.</p>
        {error && <div className="mb-4 p-3 bg-amber-50 border border-amber-200 rounded-lg text-amber-800 text-sm">{error}</div>}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-1">
              Email
            </label>
            <input
              type="email"
              id="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoFocus
              className="w-full px-4 py-2.5 border border-gray-200 rounded-lg text-gray-900 placeholder-gray-500 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              placeholder="you@example.com"
            />
          </div>
          <button type="submit" disabled={submitting} className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white font-semibold py-2.5 rounded-lg transition">
            {submitting ? 'Sending…' : 'Send magic link'}
          </button>
        </form>
        <p className="mt-6 text-center text-sm text-gray-400">
          <Link to="/" className="hover:text-indigo-600">
            ← Back to home
          </Link>
        </p>
      </div>
    </div>
  )
}
