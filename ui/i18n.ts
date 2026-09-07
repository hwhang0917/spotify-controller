// Framework-free i18n shared by the admin and guest UIs. Each app keeps its
// own messages.ts and wraps `locale` in a Vue ref.
export type Locale = 'en' | 'ko'
export type Messages = Record<Locale, Record<string, string>>

export const LOCALES: Locale[] = ['en', 'ko']
const STORAGE_KEY = 'vibe-music.locale'

export function detectLocale(): Locale {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'en' || saved === 'ko') return saved
  } catch { /* storage unavailable */ }
  return navigator.language?.toLowerCase().startsWith('ko') ? 'ko' : 'en'
}

export function saveLocale(l: Locale) {
  try { localStorage.setItem(STORAGE_KEY, l) } catch { /* storage unavailable */ }
}

// Looks up key in the locale, falls back to English, then to the key itself.
// `{name}` placeholders are replaced from vars.
export function format(messages: Messages, locale: Locale, key: string, vars?: Record<string, string | number>): string {
  let s = messages[locale][key] ?? messages.en[key] ?? key
  if (vars) for (const [k, v] of Object.entries(vars)) s = s.replaceAll(`{${k}}`, String(v))
  return s
}
