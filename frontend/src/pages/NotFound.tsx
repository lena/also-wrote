import { Link } from 'react-router-dom'

export default function NotFound() {
  return (
    <div className="container mx-auto px-4 py-16 flex flex-col items-center justify-center min-h-[50vh] text-center">
      <h1 className="text-4xl font-bold text-white mb-2">Page not found</h1>
      <p className="text-slate-300 mb-8">The page you&apos;re looking for doesn&apos;t exist.</p>
      <Link to="/" className="bg-indigo-600 text-white px-6 py-3 rounded-lg font-semibold hover:bg-indigo-700 transition">
        Back to home
      </Link>
    </div>
  )
}
