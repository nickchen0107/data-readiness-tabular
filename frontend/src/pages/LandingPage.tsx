import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export default function LandingPage() {
  const { t } = useTranslation()

  return (
    <div style={{
      minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: 'var(--canvas)', padding: 24,
    }}>
      <div style={{ textAlign: 'center', maxWidth: 480 }}>
        <div style={{
          width: 56, height: 56, borderRadius: 14, background: 'var(--ink)', color: '#fff',
          display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
          fontWeight: 700, fontSize: 24, fontFamily: 'var(--mono)', marginBottom: 20,
        }}>S</div>
        <h1 style={{ fontSize: 28, fontWeight: 700, marginBottom: 8, letterSpacing: '-0.02em' }}>
          {t('header.platform_name')}
        </h1>
        <p style={{ fontSize: 15, color: 'var(--ink-soft)', marginBottom: 32, lineHeight: 1.6 }}>
          {t('landing.subtitle')}<br />
          {t('landing.flow')}
        </p>
        <div style={{ display: 'flex', gap: 12, justifyContent: 'center' }}>
          <Link to="/login" className="btn btn-primary" style={{ textDecoration: 'none', padding: '12px 28px' }}>
            {t('btn.login')}
          </Link>
          <Link to="/register" className="btn btn-ghost" style={{ textDecoration: 'none', padding: '12px 28px' }}>
            {t('btn.register')}
          </Link>
        </div>
      </div>
    </div>
  )
}
