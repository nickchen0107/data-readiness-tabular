import { useLocation, useNavigate, Outlet } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../contexts/AuthContext'
import { useStepper } from '../contexts/StepperContext'
import LanguageSwitcher from './LanguageSwitcher'

const STEP_KEYS = [
  'stepper.landing',
  'stepper.upload',
  'stepper.assess',
  'stepper.route',
  'stepper.clean',
  'stepper.export',
  'stepper.evidence',
] as const

const STEP_PATHS = [
  '/landing',
  '/upload',
  '/assessment',
  '/routing',
  '/cleaning',
  '/export',
  '/evidence',
]

const STEP_SUBS = ['Landing', 'Upload', 'Assess', 'Route', 'Clean', 'Export', 'Evidence']

export default function Layout() {
  const { user, logout } = useAuth()
  const { t } = useTranslation()
  const { canNavigateTo } = useStepper()
  const location = useLocation()
  const navigate = useNavigate()

  const currentStepIndex = STEP_PATHS.findIndex((p) => location.pathname.startsWith(p))

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <header style={{
        maxWidth: 1180,
        width: '100%',
        margin: '0 auto',
        padding: '18px 24px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        borderBottom: '1px solid var(--line)',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 13 }}>
          <div style={{
            width: 38, height: 38, borderRadius: 9,
            background: 'var(--ink)', color: '#fff',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontWeight: 700, fontSize: 17, fontFamily: 'var(--mono)', letterSpacing: '-0.04em',
          }}>S</div>
          <div>
            <div style={{ fontSize: 16, fontWeight: 650, letterSpacing: '-0.01em', lineHeight: 1.2 }}>
              {t('header.platform_name')}
            </div>
            <div style={{ fontSize: 12.5, color: 'var(--ink-faint)', fontWeight: 500 }}>
              {t('header.tool_version')}
            </div>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          {user && user.role === 'admin' && (
            <button
              className="btn btn-ghost"
              onClick={() => navigate('/admin')}
              style={{ fontSize: 12, padding: '5px 10px' }}
            >
              {t('header.admin')}
            </button>
          )}
          <LanguageSwitcher />
          {user && (
            <span style={{ fontFamily: 'var(--mono)', fontSize: 12, color: 'var(--ink-faint)' }}>
              {user.email}
            </span>
          )}
          <button className="btn btn-ghost" onClick={logout} style={{ fontSize: 13, padding: '7px 14px' }}>
            {t('btn.logout')}
          </button>
        </div>
      </header>

      {/* Stepper */}
      <nav style={{
        maxWidth: 1180,
        width: '100%',
        margin: '0 auto',
        padding: '18px 24px 4px',
        display: 'flex',
        justifyContent: 'center',
        gap: 0,
        overflowX: 'auto',
      }}>
        {STEP_PATHS.map((path, i) => {
          const isActive = i === currentStepIndex
          const isDone = i < currentStepIndex
          const isDisabled = !canNavigateTo(i)
          return (
            <div key={path} style={{ display: 'flex', alignItems: 'center' }}>
              <div
                onClick={() => {
                  if (!isDisabled) navigate(path)
                }}
                title={isDisabled ? t('nav.stepper_disabled') : undefined}
                style={{
                  display: 'flex', alignItems: 'center', gap: 9,
                  paddingRight: 8, whiteSpace: 'nowrap',
                  cursor: isDisabled ? 'not-allowed' : 'pointer',
                  opacity: isActive ? 1 : isDisabled ? 0.3 : isDone ? 0.8 : 0.45,
                  transition: 'opacity 0.25s',
                }}
              >
                <div style={{
                  width: 24, height: 24, borderRadius: '50%',
                  border: isActive ? '1.5px solid var(--accent)' : isDone ? '1.5px solid var(--ink)' : '1.5px solid var(--ink-faint)',
                  background: isActive ? 'var(--accent)' : isDone ? 'var(--ink)' : 'transparent',
                  color: (isActive || isDone) ? '#fff' : 'var(--ink-faint)',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  fontSize: 12, fontFamily: 'var(--mono)', flexShrink: 0,
                }}>
                  {i + 1}
                </div>
                <div>
                  <div style={{ fontSize: 13, fontWeight: 550 }}>{t(STEP_KEYS[i])}</div>
                  <div style={{ fontSize: 10.5, color: 'var(--ink-faint)', fontWeight: 500, fontFamily: 'var(--mono)' }}>
                    {STEP_SUBS[i]}
                  </div>
                </div>
              </div>
              {i < STEP_PATHS.length - 1 && (
                <div style={{ width: 26, height: 1.5, background: 'var(--line)', margin: '0 4px', flexShrink: 0 }} />
              )}
            </div>
          )
        })}
      </nav>

      {/* Main content */}
      <main style={{ maxWidth: 1180, width: '100%', margin: '0 auto', padding: '30px 24px 60px', flex: 1 }}>
        <Outlet />
      </main>
    </div>
  )
}
