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
import { useI18n } from '../i18n'

function memberName(members: HouseholdDetail['members'], userID: string): string {
  const m = members.find((x) => x.user.id === userID)
  if (!m) return '—'
  return m.user.name || m.user.email
}

export default function HouseholdPage() {
  const { id = '' } = useParams()
  const { user } = useAuth()
  const { t } = useI18n()

  const [detail, setDetail] = useState<HouseholdDetail | null>(null)
  const [month, setMonth] = useState(currentMonth())
  const [incomeDraft, setIncomeDraft] = useState<Record<string, string>>({})
  const [expenses, setExpenses] = useState<Expense[]>([])
  const [settlement, setSettlement] = useState<Settlement | null>(null)
  const [error, setError] = useState('')

  const [memberEmail, setMemberEmail] = useState('')
  const [managedName, setManagedName] = useState('')

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
      setError(err instanceof Error ? err.message : t('household.loadError'))
    }
  }, [id, month, user?.id, t])

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
      setError(err instanceof Error ? err.message : t('common.error'))
    }
  }

  async function handleAddManagedMember(e: FormEvent) {
    e.preventDefault()
    if (!managedName.trim()) return
    try {
      await api.addManagedMember(id, managedName.trim())
      setManagedName('')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('common.error'))
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
      setError(err instanceof Error ? err.message : t('common.error'))
    }
  }

  async function handleDeleteExpense(expenseId: string) {
    try {
      await api.deleteExpense(id, expenseId)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('common.error'))
    }
  }

  if (!detail) {
    return (
      <div className="page-pad">
        <p className="muted">{error || t('common.loading')}</p>
      </div>
    )
  }

  return (
    <div className="page-pad">
      <Link to="/" className="muted">
        {t('household.back')}
      </Link>
      <h1 className="title">{detail.household.name}</h1>

      <label className="field month-field">
        {t('household.month')}
        <input type="month" value={month} onChange={(e) => setMonth(e.target.value)} />
      </label>

      {error && <p className="error">{error}</p>}

      <div className="section-grid">
        <section className="card">
          <h2>{t('household.membersTitle')}</h2>
          <table className="table">
            <thead>
              <tr>
                <th>{t('household.colMember')}</th>
                <th>{t('household.colIncome')}</th>
              </tr>
            </thead>
            <tbody>
              {detail.members.map((m) => (
                <tr key={m.user.id}>
                  <td>
                    {m.user.name || m.user.email || '—'}
                    {m.role === 'admin' && <span className="badge">admin</span>}
                    {!m.user.has_login && <span className="badge">{t('household.badgeNoLogin')}</span>}
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
            {t('household.saveIncomes')}
          </button>
          <p className="muted small">
            {t('household.noIncomeHint')}
          </p>
        </section>

        <section className="card">
          <h2>{t('household.expensesTitle')}</h2>
          <form onSubmit={handleAddExpense} className="stack">
            <label className="field">
              {t('household.expenseName')}
              <input
                value={expenseName}
                onChange={(e) => setExpenseName(e.target.value)}
                placeholder={t('household.expenseNamePlaceholder')}
              />
            </label>
            <div className="row-form">
              <label className="field">
                {t('household.amount')}
                <input
                  className="euro-input"
                  inputMode="decimal"
                  value={expenseEuros}
                  onChange={(e) => setExpenseEuros(e.target.value)}
                  placeholder="0,00"
                />
              </label>
              <label className="field">
                {t('household.paidBy')}
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
              {t('household.commonExpense')}
            </label>
            {expenseCommon && (
              <label className="field">
                {t('household.split')}
                <select value={expenseSplit} onChange={(e) => setExpenseSplit(e.target.value as SplitMode)}>
                  <option value="proportional">{t('household.splitProportional')}</option>
                  <option value="equal">{t('household.splitEqual')}</option>
                </select>
              </label>
            )}
            <button className="btn btn-primary" type="submit">
              {t('household.addExpense')}
            </button>
          </form>

          <ul className="expense-list">
            {expenses.map((x) => (
              <li key={x.id} className={x.is_common ? '' : 'personal'}>
                <div>
                  <strong>{x.name}</strong>
                  <span className="muted small">
                    {' '}
                    {t('household.paidByLabel', { amount: formatEuros(x.amount_cents), name: memberName(detail.members, x.paid_by) })}{' '}
                    {x.is_common
                      ? x.split_mode === 'equal'
                        ? t('household.modeEqual')
                        : t('household.modeProportional')
                      : t('household.modePersonal')}
                  </span>
                </div>
                <button className="btn btn-ghost" onClick={() => void handleDeleteExpense(x.id)}>
                  ✕
                </button>
              </li>
            ))}
            {expenses.length === 0 && <li className="muted">{t('household.noExpenses')}</li>}
          </ul>
        </section>

        <section className="card">
          <h2>{t('household.balancesTitle')}</h2>
          <p className="muted small">{t('household.commonTotal', { amount: formatEuros(settlement?.total_common_cents ?? 0) })}</p>
          <table className="table">
            <thead>
              <tr>
                <th>{t('household.colMember')}</th>
                <th>{t('household.colPaid')}</th>
                <th>{t('household.colShare')}</th>
                <th>{t('household.colBalance')}</th>
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
          <h3>{t('household.whoRepaysTitle')}</h3>
          <ul>
            {(settlement?.transfers ?? []).map((tr, i) => (
              <li key={i}>
                {t('household.transfer', {
                  from: memberName(detail.members, tr.from_user_id),
                  amount: formatEuros(tr.amount_cents),
                  to: memberName(detail.members, tr.to_user_id),
                })}
              </li>
            ))}
            {(settlement?.transfers ?? []).length === 0 && <li className="muted">{t('household.allSettled')}</li>}
          </ul>
        </section>

        {isAdmin && (
          <section className="card">
            <h2>{t('household.addMemberTitle')}</h2>
            <p className="muted small">
              {t('household.addMemberHint')}
            </p>
            <form onSubmit={handleAddMember} className="row-form">
              <input
                type="email"
                value={memberEmail}
                onChange={(e) => setMemberEmail(e.target.value)}
                placeholder={t('household.emailPlaceholder')}
              />
              <button className="btn btn-primary" type="submit">
                {t('household.invite')}
              </button>
            </form>
            <h3>{t('household.managedTitle')}</h3>
            <p className="muted small">
              {t('household.managedHint')}
            </p>
            <form onSubmit={handleAddManagedMember} className="row-form">
              <input
                value={managedName}
                onChange={(e) => setManagedName(e.target.value)}
                placeholder={t('household.managedNamePlaceholder')}
              />
              <button className="btn btn-primary" type="submit">
                {t('household.create')}
              </button>
            </form>
          </section>
        )}
      </div>
    </div>
  )
}
