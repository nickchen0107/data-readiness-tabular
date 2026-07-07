/**
 * Translate dynamic backend strings (issue labels, descriptions, units)
 * using the i18n translation function.
 * 
 * Looks up "dynamic.{originalText}" in the translation table.
 * If found → returns translated text
 * If not found → returns original text unchanged
 * 
 * This allows backend Chinese strings to be translated on the frontend
 * without modifying backend code. Translations are managed via the
 * Translation Editor in admin panel.
 */
export function td(text: string | undefined | null, t: (key: string, options?: Record<string, unknown>) => string, lang: string): string {
  if (!text || lang !== 'en') return text || ''
  const key = `dynamic.${text}`
  const translated = t(key, { defaultValue: '__NOTFOUND__' })
  return translated === '__NOTFOUND__' ? text : translated
}
