/**
 * Translate dynamic backend strings (issue labels, descriptions, units)
 * using the i18n translation function.
 * 
 * Looks up "dynamic.{originalText}" in the translation table.
 * If found → returns translated text
 * If not found → tries pattern-based translation
 * If still not found → returns original text unchanged
 */
export function td(text: string | undefined | null, t: (key: string, options?: Record<string, unknown>) => string, lang: string): string {
  if (!text || lang !== 'en') return text || ''
  const key = `dynamic.${text}`
  const translated = t(key, { defaultValue: '__NOTFOUND__' })
  if (translated !== '__NOTFOUND__') return translated
  
  // Pattern-based translations for dynamic strings with variables
  const patterns: Array<[RegExp, (m: RegExpMatchArray) => string]> = [
    [/^欄位「(.+?)」超過一半為空$/, (m) => `Column "${m[1]}" is over 50% empty`],
    [/^偵測到\s*(.+)/, (m) => `Detected: ${m[1]}`],
    [/^共\s*(\d+)\s*列/, () => text], // keep as-is, description_en handles this
  ]
  
  for (const [re, fn] of patterns) {
    const match = text.match(re)
    if (match) return fn(match)
  }
  
  return text
}
