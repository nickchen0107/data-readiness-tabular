import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export default function DashboardLandingPage() {
  const navigate = useNavigate()
  const { t } = useTranslation()

  return (
    <div style={{
      background: 'var(--paper)', border: '1px solid var(--line)',
      borderRadius: 14, overflow: 'hidden', minHeight: 440,
      display: 'flex', flexDirection: 'column',
    }}>
      {/* Stage header */}
      <div style={{
        padding: '20px 28px', borderBottom: '1px solid var(--line-soft)',
        display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 20,
      }}>
        <div>
          <div style={{
            fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--accent)',
            letterSpacing: '0.08em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 5,
          }}>LANDING</div>
          <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>
            {t('dashboard.title')}
          </h2>
          <p style={{ color: 'var(--ink-soft)', fontSize: 14, marginTop: 5, maxWidth: 560 }}>
            {t('dashboard.desc')}
          </p>
        </div>
      </div>

      {/* Stage body */}
      <div style={{ padding: 28, flex: 1 }}>
        {/* AICM card */}
        <div style={{
          display: 'flex', gap: 16, alignItems: 'flex-start',
          border: '1px dashed var(--line)', borderRadius: 'var(--radius)', padding: '16px 18px',
        }}>
          <div style={{ fontSize: 22 }}>🛡️</div>
          <div>
            <div style={{ fontWeight: 650, marginBottom: 3 }}>{t('dashboard.aicm_title')}</div>
            <div style={{ color: 'var(--ink-faint)', fontSize: 13 }}>
              {t('dashboard.aicm_desc')}
            </div>
          </div>
        </div>

        {/* Excel tool card */}
        <div style={{
          marginTop: 16, border: '1px solid var(--accent)', background: 'var(--accent-soft)',
          borderRadius: 'var(--radius)', padding: '16px 18px',
          display: 'flex', gap: 14, alignItems: 'center', cursor: 'pointer',
        }}
          onClick={() => navigate('/upload')}
        >
          <div style={{
            width: 46, height: 46, borderRadius: 10, background: 'var(--accent)', color: '#fff',
            display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 22,
          }}>📊</div>
          <div style={{ flex: 1 }}>
            <div style={{ fontWeight: 650, fontSize: 16 }}>{t('dashboard.excel_tool_title')}</div>
            <div style={{ fontSize: 13, color: '#1c4e7a' }}>
              {t('dashboard.excel_tool_desc')}
            </div>
          </div>
        </div>

        <p style={{ color: 'var(--ink-faint)', fontSize: 13, marginTop: 18, textAlign: 'center' }}>
          {t('dashboard.hint')}
        </p>
      </div>

      {/* Stage footer */}
      <div style={{
        padding: '16px 28px', borderTop: '1px solid var(--line-soft)',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        background: 'var(--panel)',
      }}>
        <div />
        <div style={{ fontSize: 12.5, color: 'var(--ink-faint)' }}>1 / 8</div>
        <button className="btn btn-primary" onClick={() => navigate('/upload')}>
          {t('btn.next_step')} →
        </button>
      </div>
    </div>
  )
}
