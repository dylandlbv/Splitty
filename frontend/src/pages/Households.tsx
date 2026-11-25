import { FormEvent, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, type Household } from '../api'
import { useI18n } from '../i18n'

export default function HouseholdsPage() {
  const { t } = useI18n()
  const [households, setHouseholds] = useState<Household[]>([])
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api
      .households()
      .then(setHouseholds)
      .catch((err) => setError(err instanceof Error ? err.message : t('common.error')))
      .finally(() => setLoading(false))
  }, [t])

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    setError('')
    try {
      const created = await api.createHousehold(name.trim())
      setHouseholds((prev) => [...prev, created])
      setName('')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('common.error'))
    }
  }

  return (
    <div className="page-pad">
      <h1 className="title">{t('households.title')}</h1>
      {loading && <p className="muted">{t('common.loading')}</p>}
      {!loading && households.length === 0 && <p className="muted">{t('households.empty')}</p>}
      <div className="grid-cards">
        {households.map((h) => (
          <Link key={h.id} to={`/households/${h.id}`} className="card household-card">
            <span className="household-name">{h.name}</span>
            <span className="muted">{t('households.open')}</span>
          </Link>
        ))}
      </div>
      <form onSubmit={handleCreate} className="card row-form">
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('households.newNamePlaceholder')}
        />
        <button className="btn btn-primary" type="submit">
          {t('households.create')}
        </button>
      </form>
      {error && <p className="error">{error}</p>}
    </div>
  )
}
