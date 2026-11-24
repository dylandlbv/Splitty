export const TOKEN_KEY = 'splitty_token'

export type User = {
  id: string
  email: string
  name: string
  has_login: boolean
}

export type Household = {
  id: string
  name: string
}

export type Member = {
  user: User
  role: 'admin' | 'member'
}

export type HouseholdDetail = {
  household: Household
  members: Member[]
}

export type Income = {
  user_id: string
  month: string
  amount_cents: number
}

export type SplitMode = 'equal' | 'proportional'

export type Expense = {
  id: string
  household_id: string
  name: string
  amount_cents: number
  paid_by: string
  is_common: boolean
  split_mode: SplitMode
  month: string
  created_at: string
}

export type MemberBalance = {
  user_id: string
  paid_cents: number
  share_cents: number
  balance_cents: number
}

export type Transfer = {
  from_user_id: string
  to_user_id: string
  amount_cents: number
}

export type Settlement = {
  month: string
  total_common_cents: number
  balances: MemberBalance[]
  transfers: Transfer[]
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, opts: RequestInit = {}, authenticated = true): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  const token = localStorage.getItem(TOKEN_KEY)
  if (authenticated && token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const res = await fetch(path, { ...opts, headers })
  if (!res.ok) {
    let message = res.statusText
    try {
      const body = await res.json()
      if (body && typeof body.error === 'string') {
        message = body.error
      }
    } catch {
      /* keep statusText */
    }
    if (res.status === 401 && authenticated) {
      localStorage.removeItem(TOKEN_KEY)
    }
    throw new ApiError(res.status, message)
  }
  if (res.status === 204) {
    return undefined as T
  }
  return res.json()
}

function jsonBody(payload: unknown): RequestInit {
  return { method: 'POST', body: JSON.stringify(payload) }
}

export const api = {
  register: (email: string, name: string, password: string) =>
    request<{ token: string; user: User }>('/api/auth/register', jsonBody({ email, name, password }), false),
  login: (email: string, password: string) =>
    request<{ token: string; user: User }>('/api/auth/login', jsonBody({ email, password }), false),
  me: () => request<User>('/api/me'),
  households: () => request<Household[]>('/api/households'),
  createHousehold: (name: string) => request<Household>('/api/households', jsonBody({ name })),
  household: (id: string) => request<HouseholdDetail>(`/api/households/${id}`),
  addMember: (id: string, email: string) =>
    request<Member>(`/api/households/${id}/members`, jsonBody({ email })),
  addManagedMember: (id: string, name: string) =>
    request<Member>(`/api/households/${id}/members/managed`, jsonBody({ name })),
  incomes: (id: string, month: string) => request<Income[]>(`/api/households/${id}/incomes?month=${month}`),
  setIncomes: (id: string, month: string, incomes: Income[]) =>
    request<Income[]>(`/api/households/${id}/incomes/${month}`, {
      method: 'PUT',
      body: JSON.stringify({ incomes }),
    }),
  expenses: (id: string, month: string) => request<Expense[]>(`/api/households/${id}/expenses?month=${month}`),
  createExpense: (id: string, expense: Omit<Expense, 'id' | 'household_id' | 'created_at'>) =>
    request<Expense>(`/api/households/${id}/expenses`, jsonBody(expense)),
  deleteExpense: (id: string, expenseId: string) =>
    request<void>(`/api/households/${id}/expenses/${expenseId}`, { method: 'DELETE' }),
  balances: (id: string, month: string) => request<Settlement>(`/api/households/${id}/balances/${month}`),
}

export function formatEuros(cents: number): string {
  return (cents / 100).toLocaleString('fr-FR', { style: 'currency', currency: 'EUR' })
}

export function eurosToCents(euros: string): number {
  const value = Number(euros.replace(',', '.'))
  return Math.round(value * 100)
}

export function currentMonth(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}
