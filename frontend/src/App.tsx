import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { AuthProvider } from './AuthContext'
import Layout from './Layout'
import Home from './pages/Home'
import Search from './pages/Search'
import Show from './pages/Show'
import Writer from './pages/Writer'
import Episode from './pages/Episode'
import Login from './pages/Login'
import FavoriteWriters from './pages/FavoriteWriters'
import Admin from './pages/Admin'
import NotFound from './pages/NotFound'

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route element={<Layout />}>
            <Route path="/" element={<Home />} />
            <Route path="/search" element={<Search />} />
            <Route path="/show" element={<Show />} />
            <Route path="/writer" element={<Writer />} />
            <Route path="/episode" element={<Episode />} />
            <Route path="/login" element={<Login />} />
            <Route path="/favorite-writers" element={<FavoriteWriters />} />
            <Route path="/admin" element={<Admin />} />
            <Route path="*" element={<NotFound />} />
          </Route>
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}
