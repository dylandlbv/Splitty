import { FormEvent, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, type Household } from '../api'

export default function HouseholdsPage() {
  const [households, setHouseholds] = useState<Household[]>([])
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api
      .households()
      .then(setHouseholds)
      .catch((err) => setError(err instanceof Error ? err.message : 'Erreur'))
      .finally(() => setLoading(false))
  }, [])

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    setError('')
    try {
      const created = await api.createHousehold(name.trim())
      setHouseholds((prev) => [...prev, created])
      setName('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erreur')
    }
  }

  return (
    <div className="page-pad">
      <h1 className="title">Mes foyers</h1>
      {loading && <p className="muted">Chargement…</p>}
      {!loading && households.length === 0 && (
        <p className="muted">Aucun foyer pour l'instant. Crée le tien pour commencer.</p>
      )}
      <div className="grid-cards">
        {households.map((h) => (
          <Link key={h.id} to={`/households/${h.id}`} className="card household-card">
            <span className="household-name">{h.name}</span>
            <span className="muted">Ouvrir →</span>
          </Link>
        ))}
      </div>
      <form onSubmit={handleCreate} className="card row-form">
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Nom du nouveau foyer (ex. Famille Dupont)"
        />
        <button className="btn btn-primary" type="submit">
          Créer le foyer
        </button>
      </form>
      {error && <p className="error">{error}</p>}
    </div>
  )
}
