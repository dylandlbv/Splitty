import { Navigate, Route, Routes, Link } from 'react-router-dom'
import type { ReactElement } from 'react'
import { useAuth } from './auth'
import { useI18n } from './i18n'
import LanguageSwitcher from './components/LanguageSwitcher'
import HouseholdsPage from './pages/Households'
import HouseholdPage from './pages/Household'
import LoginPage from './pages/Login'

function RequireAuth({ children }: { children: ReactElement }) {
  const { user, loading } = useAuth()
  const { t } = useI18n()
  if (loading) {
    return <p className="muted page-pad">{t('common.loading')}</p>
  }
  if (!user) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

export default function App() {
  const { user, loading, logout } = useAuth()
  const { t } = useI18n()

  return (
    <div>
      {user && (
        <nav className="topbar">
          <Link to="/" className="brand">
            Splitty
          </Link>
          <div className="topbar-right">
            <span className="muted">{user.name || user.email}</span>
            <LanguageSwitcher />
            <button className="btn btn-ghost" onClick={logout}>
              {t('app.logout')}
            </button>
          </div>
        </nav>
      )}
      <Routes>
        <Route
          path="/login"
          element={
            loading ? (
              <p className="muted page-pad">{t('common.loading')}</p>
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
