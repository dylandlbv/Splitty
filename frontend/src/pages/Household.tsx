import { FormEvent, useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  api,
  currentMonth,
  eurosToCents,
  formatEuros,
  type Expense,
  type HouseholdDetail,
  type Income,
  type Settlement,
  type SplitMode,
} from '../api'
import { useAuth } from '../auth'

function memberName(members: HouseholdDetail['members'], userID: string): string {
  const m = members.find((x) => x.user.id === userID)
  if (!m) return '—'
  return m.user.name || m.user.email
}

export default function HouseholdPage() {
  const { id = '' } = useParams()
  const { user } = useAuth()

  const [detail, setDetail] = useState<HouseholdDetail | null>(null)
  const [month, setMonth] = useState(currentMonth())
  const [incomeDraft, setIncomeDraft] = useState<Record<string, string>>({})
  const [expenses, setExpenses] = useState<Expense[]>([])
  const [settlement, setSettlement] = useState<Settlement | null>(null)
  const [error, setError] = useState('')

  const [memberEmail, setMemberEmail] = useState('')

  const [expenseName, setExpenseName] = useState('')
  const [expenseEuros, setExpenseEuros] = useState('')
  const [expensePaidBy, setExpensePaidBy] = useState('')
  const [expenseCommon, setExpenseCommon] = useState(true)
  const [expenseSplit, setExpenseSplit] = useState<SplitMode>('proportional')

  const isAdmin = detail?.members.some((m) => m.user.id === user?.id && m.role === 'admin') ?? false

  const load = useCallback(async () => {
    setError('')
    try {
      const [d, inc, exp, bal] = await Promise.all([
        api.household(id),
        api.incomes(id, month),
        api.expenses(id, month),
        api.balances(id, month),
      ])
      setDetail(d)
      setExpenses(exp)
      setSettlement(bal)
      setExpensePaidBy((prev) => prev || user?.id || '')
      setIncomeDraft((prev) => {
        const next: Record<string, string> = { ...prev }
        for (const m of d.members) {
          const existing = inc.find((i) => i.user_id === m.user.id)
          next[m.user.id] =
            next[m.user.id] ??
            (existing !== undefined ? (existing.amount_cents / 100).toFixed(2).replace('.', ',') : '')
        }
        return next
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erreur de chargement')
    }
  }, [id, month, user?.id])

  useEffect(() => {
    void load()
  }, [load])

  async function handleAddMember(e: FormEvent) {
    e.preventDefault()
    if (!memberEmail.trim()) return
    try {
      await api.addMember(id, memberEmail.trim())
      setMemberEmail('')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erreur')
    }
  }

  async function handleSaveIncomes() {
    try {
      const payload: Income[] = detail!.members.map((m) => {
        const raw = (incomeDraft[m.user.id] ?? '').trim()
        return {
          user_id: m.user.id,
          month,
          amount_cents: raw === '' ? 0 : eurosToCents(raw),
        }
      })
      await api.setIncomes(id, month, payload)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erreur')
    }
  }

  async function handleAddExpense(e: FormEvent) {
    e.preventDefault()
    if (!expenseName.trim() || expenseEuros.trim() === '') return
    try {
      await api.createExpense(id, {
        name: expenseName.trim(),
        amount_cents: eurosToCents(expenseEuros),
        paid_by: expensePaidBy || user!.id,
        is_common: expenseCommon,
        split_mode: expenseSplit,
        month,
      })
      setExpenseName('')
      setExpenseEuros('')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erreur')
    }
  }

  async function handleDeleteExpense(expenseId: string) {
    try {
      await api.deleteExpense(id, expenseId)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erreur')
    }
  }

  if (!detail) {
    return (
      <div className="page-pad">
        <p className="muted">{error || 'Chargement…'}</p>
      </div>
    )
  }

  return (
    <div className="page-pad">
      <Link to="/" className="muted">
        ← Mes foyers
      </Link>
      <h1 className="title">{detail.household.name}</h1>

      <label className="field month-field">
        Mois
        <input type="month" value={month} onChange={(e) => setMonth(e.target.value)} />
      </label>

      {error && <p className="error">{error}</p>}

      <div className="section-grid">
        <section className="card">
          <h2>Membres & revenus</h2>
          <table className="table">
            <thead>
              <tr>
                <th>Membre</th>
                <th>Revenu du mois</th>
              </tr>
            </thead>
            <tbody>
              {detail.members.map((m) => (
                <tr key={m.user.id}>
                  <td>
                    {m.user.name || m.user.email}
                    {m.role === 'admin' && <span className="badge">admin</span>}
                  </td>
                  <td>
                    <input
                      className="euro-input"
                      inputMode="decimal"
                      placeholder="0,00"
                      value={incomeDraft[m.user.id] ?? ''}
                      onChange={(e) => setIncomeDraft({ ...incomeDraft, [m.user.id]: e.target.value })}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <button className="btn btn-primary" onClick={() => void handleSaveIncomes()}>
            Enregistrer les revenus
          </button>
          <p className="muted small">
            Un membre sans revenu déclaré (0 €) n'est pas compté dans les dépenses proportionnelles du mois.
          </p>
        </section>

        <section className="card">
          <h2>Dépenses du mois</h2>
          <form onSubmit={handleAddExpense} className="stack">
            <label className="field">
              Nom
              <input
                value={expenseName}
                onChange={(e) => setExpenseName(e.target.value)}
                placeholder="Courses, loyer, électricité…"
              />
            </label>
            <div className="row-form">
              <label className="field">
                Montant (€)
                <input
                  className="euro-input"
                  inputMode="decimal"
                  value={expenseEuros}
                  onChange={(e) => setExpenseEuros(e.target.value)}
                  placeholder="0,00"
                />
              </label>
              <label className="field">
                Payé par
                <select value={expensePaidBy} onChange={(e) => setExpensePaidBy(e.target.value)}>
                  {detail.members.map((m) => (
                    <option key={m.user.id} value={m.user.id}>
                      {m.user.name || m.user.email}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            <label className="check">
              <input
                type="checkbox"
                checked={expenseCommon}
                onChange={(e) => setExpenseCommon(e.target.checked)}
              />
              Dépense commune (sinon personne ne participe)
            </label>
            {expenseCommon && (
              <label className="field">
                Répartition
                <select value={expenseSplit} onChange={(e) => setExpenseSplit(e.target.value as SplitMode)}>
                  <option value="proportional">Proportionnelle aux revenus</option>
                  <option value="equal">Égalitaire</option>
                </select>
              </label>
            )}
            <button className="btn btn-primary" type="submit">
              Ajouter la dépense
            </button>
          </form>

          <ul className="expense-list">
            {expenses.map((x) => (
              <li key={x.id} className={x.is_common ? '' : 'personal'}>
                <div>
                  <strong>{x.name}</strong>
                  <span className="muted small">
                    {' '}
                    — {formatEuros(x.amount_cents)} payé par {memberName(detail.members, x.paid_by)}{' '}
                    {x.is_common ? (x.split_mode === 'equal' ? '· égalitaire' : '· proportionnelle') : '· perso'}
                  </span>
                </div>
                <button className="btn btn-ghost" onClick={() => void handleDeleteExpense(x.id)}>
                  ✕
                </button>
              </li>
            ))}
            {expenses.length === 0 && <li className="muted">Aucune dépense ce mois-ci.</li>}
          </ul>
        </section>

        <section className="card">
          <h2>Soldes du mois</h2>
          <p className="muted small">Commun : {formatEuros(settlement?.total_common_cents ?? 0)}</p>
          <table className="table">
            <thead>
              <tr>
                <th>Membre</th>
                <th>Payé</th>
                <th>Part</th>
                <th>Solde</th>
              </tr>
            </thead>
            <tbody>
              {(settlement?.balances ?? []).map((b) => (
                <tr key={b.user_id}>
                  <td>{memberName(detail.members, b.user_id)}</td>
                  <td>{formatEuros(b.paid_cents)}</td>
                  <td>{formatEuros(b.share_cents)}</td>
                  <td className={b.balance_cents < 0 ? 'negative' : b.balance_cents > 0 ? 'positive' : ''}>
                    {formatEuros(b.balance_cents)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <h3>Qui rembourse qui</h3>
          <ul>
            {(settlement?.transfers ?? []).map((t, i) => (
              <li key={i}>
                <strong>{memberName(detail.members, t.from_user_id)}</strong> doit{' '}
                <strong>{formatEuros(t.amount_cents)}</strong> à{' '}
                <strong>{memberName(detail.members, t.to_user_id)}</strong>
              </li>
            ))}
            {(settlement?.transfers ?? []).length === 0 && <li className="muted">Tout est à l'équilibre 🎉</li>}
          </ul>
        </section>

        {isAdmin && (
          <section className="card">
            <h2>Ajouter un membre</h2>
            <p className="muted small">
              La personne doit d'abord créer un compte Splitty avec cet email.
            </p>
            <form onSubmit={handleAddMember} className="row-form">
              <input
                type="email"
                value={memberEmail}
                onChange={(e) => setMemberEmail(e.target.value)}
                placeholder="email@exemple.fr"
              />
              <button className="btn btn-primary" type="submit">
                Inviter
              </button>
            </form>
          </section>
        )}
      </div>
    </div>
  )
}
