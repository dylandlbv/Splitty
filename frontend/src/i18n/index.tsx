import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import en from './en'
import fr from './fr'

export type Locale = 'en' | 'fr'
export type Translation = typeof en

const dictionaries: Record<Locale, Translation> = {
  en,
  fr: fr as unknown as Translation,
}

const LOCALE_KEY = 'splitty_locale'
export const DEFAULT_LOCALE: Locale = 'en'

export function detectLocale(): Locale {
  const stored = localStorage.getItem(LOCALE_KEY)
  if (stored === 'en' || stored === 'fr') return stored
  return DEFAULT_LOCALE
}

type I18nState = {
  locale: Locale
  setLocale: (locale: Locale) => void
  t: (key: keyof Translation | string, vars?: Record<string, string>) => string
}

const I18nContext = createContext<I18nState | null>(null)

function lookup(dict: unknown, key: string): unknown {
  return key.split('.').reduce<unknown>((acc, part) => {
    if (acc && typeof acc === 'object' && part in (acc as Record<string, unknown>)) {
      return (acc as Record<string, unknown>)[part]
    }
    return undefined
  }, dict)
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(detectLocale)

  useEffect(() => {
    document.documentElement.lang = locale
  }, [locale])

  function setLocale(next: Locale) {
    setLocaleState(next)
    localStorage.setItem(LOCALE_KEY, next)
  }

  function t(key: string, vars?: Record<string, string>): string {
    let value = lookup(dictionaries[locale], key)
    if (value === undefined) value = lookup(dictionaries[DEFAULT_LOCALE], key)
    if (typeof value !== 'string') return key
    if (vars) {
      for (const [name, replacement] of Object.entries(vars)) {
        value = (value as string).replace(`{${name}}`, replacement)
      }
    }
    return value as string
  }

  return (
    <I18nContext.Provider value={{ locale, setLocale, t }}>{children}</I18nContext.Provider>
  )
}

export function useI18n(): I18nState {
  const ctx = useContext(I18nContext)
  if (!ctx) {
    throw new Error('useI18n must be used within I18nProvider')
  }
  return ctx
}
