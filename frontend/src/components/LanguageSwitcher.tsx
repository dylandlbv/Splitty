import { useI18n, type Locale } from '../i18n'

const LOCALES: Locale[] = ['en', 'fr']
const LABELS: Record<Locale, string> = { en: 'EN', fr: 'FR' }

export default function LanguageSwitcher() {
  const { locale, setLocale } = useI18n()

  return (
    <select
      className="lang-switcher"
      value={locale}
      onChange={(e) => setLocale(e.target.value as Locale)}
      aria-label="Language"
    >
      {LOCALES.map((l) => (
        <option key={l} value={l}>
          {LABELS[l]}
        </option>
      ))}
    </select>
  )
}
