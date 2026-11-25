import { FormEvent, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../auth'
import { useI18n } from '../i18n'
import LanguageSwitcher from '../components/LanguageSwitcher'

export default function LoginPage() {
  const { login, register } = useAuth()
  const { t } = useI18n()
  const navigate = useNavigate()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      if (mode === 'login') {
        await login(email, password)
      } else {
        await register(email, name, password)
      }
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('common.error'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="page-pad auth-wrap">
      <div className="card auth-card">
        <h1 className="title">Splitty</h1>
        <p className="muted">{t('login.tagline')}</p>
        <form onSubmit={handleSubmit} className="stack">
          {mode === 'register' && (
            <label className="field">
              {t('login.name')}
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t('login.namePlaceholder')}
              />
            </label>
          )}
          <label className="field">
            {t('login.email')}
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t('login.emailPlaceholder')}
            />
          </label>
          <label className="field">
            {t('login.password')}
            <input
              type="password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t('login.passwordPlaceholder')}
            />
          </label>
          {error && <p className="error">{error}</p>}
          <button className="btn btn-primary" disabled={busy}>
            {busy ? '…' : mode === 'login' ? t('login.signIn') : t('login.createAccount')}
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => setMode(mode === 'login' ? 'register' : 'login')}
          >
            {mode === 'login' ? t('login.noAccount') : t('login.hasAccount')}
          </button>
        </form>
      </div>
      <LanguageSwitcher />
    </div>
  )
}
