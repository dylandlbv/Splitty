import { Navigate, Route, Routes, Link } from 'react-router-dom'
import type { ReactElement } from 'react'
import { useAuth } from './auth'
import HouseholdsPage from './pages/Households'
import HouseholdPage from './pages/Household'
import LoginPage from './pages/Login'

function RequireAuth({ children }: { children: ReactElement }) {
  const { user, loading } = useAuth()
  if (loading) {
    return <p className="muted page-pad">Chargement…</p>
  }
  if (!user) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

export default function App() {
  const { user, loading, logout } = useAuth()

  return (
    <div>
      {user && (
        <nav className="topbar">
          <Link to="/" className="brand">
            Splitty
          </Link>
          <div className="topbar-right">
            <span className="muted">{user.name || user.email}</span>
            <button className="btn btn-ghost" onClick={logout}>
              Déconnexion
            </button>
          </div>
        </nav>
      )}
      <Routes>
        <Route
          path="/login"
          element={
            loading ? (
              <p className="muted page-pad">Chargement…</p>
            ) : !user ? (
              <LoginPage />
            ) : (
              <Navigate to="/" replace />
            )
          }
        />
        <Route
          path="/"
          element={
            <RequireAuth>
              <HouseholdsPage />
            </RequireAuth>
          }
        />
        <Route
          path="/households/:id"
          element={
            <RequireAuth>
              <HouseholdPage />
            </RequireAuth>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </div>
  )
}
