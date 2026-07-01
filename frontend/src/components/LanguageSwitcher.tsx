import { useTranslation } from 'react-i18next'

export default function LanguageSwitcher() {
  const { i18n } = useTranslation()

  const currentLang = i18n.language
  const label = currentLang === 'zh-TW' ? '中' : 'EN'

  const toggle = () => {
    const nextLang = currentLang === 'zh-TW' ? 'en' : 'zh-TW'
    i18n.changeLanguage(nextLang)
    localStorage.setItem('language', nextLang)
  }

  return (
    <button
      onClick={toggle}
      style={{
        fontSize: 12,
        fontWeight: 600,
        padding: '5px 10px',
        borderRadius: 6,
        border: '1px solid var(--line)',
        background: 'var(--paper)',
        color: 'var(--ink-soft)',
        cursor: 'pointer',
        fontFamily: 'var(--mono)',
      }}
    >
      {label}
    </button>
  )
}
