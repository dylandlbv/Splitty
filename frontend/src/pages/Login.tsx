import { FormEvent, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../auth'

export default function LoginPage() {
  const { login, register } = useAuth()
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
      setError(err instanceof Error ? err.message : 'Une erreur est survenue')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="page-pad auth-wrap">
      <div className="card auth-card">
        <h1 className="title">Splitty</h1>
        <p className="muted">Le budget familial, chacun paie sa juste part — au prorata de ses revenus.</p>
        <form onSubmit={handleSubmit} className="stack">
          {mode === 'register' && (
            <label className="field">
              Nom
              <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Dylan" />
            </label>
          )}
          <label className="field">
            Email
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="moi@exemple.fr"
            />
          </label>
          <label className="field">
            Mot de passe
            <input
              type="password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="8 caractères minimum"
            />
          </label>
          {error && <p className="error">{error}</p>}
          <button className="btn btn-primary" disabled={busy}>
            {busy ? '…' : mode === 'login' ? 'Se connecter' : 'Créer mon compte'}
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            onClick={() => setMode(mode === 'login' ? 'register' : 'login')}
          >
            {mode === 'login' ? "Pas de compte ? S'inscrire" : 'Déjà un compte ? Se connecter'}
          </button>
        </form>
      </div>
    </div>
  )
}
