import { useLocation, useNavigate, Outlet } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

const STEPS = [
  { path: '/landing', num: '1', label: '進入', sub: 'Landing' },
  { path: '/upload', num: '2', label: '上傳', sub: 'Upload' },
  { path: '/assessment', num: '3', label: '評估', sub: 'Assess' },
  { path: '/routing', num: '4', label: '分流', sub: 'Route' },
  { path: '/cleaning', num: '5', label: '梳理', sub: 'Clean' },
  { path: '/export', num: '6', label: '產出', sub: 'Export' },
  { path: '/evidence', num: '7', label: '存證', sub: 'Evidence' },
  { path: '/qa', num: '8', label: '問答', sub: 'QA' },
]

export default function Layout() {
  const { user, logout } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()

  const currentStepIndex = STEPS.findIndex((s) => location.pathname.startsWith(s.path))

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
              SAFE-AI 資料梳理平台
            </div>
            <div style={{ fontSize: 12.5, color: 'var(--ink-faint)', fontWeight: 500 }}>
              Excel 梳理小工具 v0.1
            </div>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          {user && (
            <span style={{ fontFamily: 'var(--mono)', fontSize: 12, color: 'var(--ink-faint)' }}>
              {user.email}
            </span>
          )}
          <button className="btn btn-ghost" onClick={logout} style={{ fontSize: 13, padding: '7px 14px' }}>
            登出
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
        {STEPS.map((step, i) => {
          const isActive = i === currentStepIndex
          const isDone = i < currentStepIndex
          return (
            <div key={step.path} style={{ display: 'flex', alignItems: 'center' }}>
              <div
                onClick={() => navigate(step.path)}
                style={{
                  display: 'flex', alignItems: 'center', gap: 9,
                  paddingRight: 8, whiteSpace: 'nowrap', cursor: 'pointer',
                  opacity: isActive ? 1 : isDone ? 0.8 : 0.45,
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
                  {step.num}
                </div>
                <div>
                  <div style={{ fontSize: 13, fontWeight: 550 }}>{step.label}</div>
                  <div style={{ fontSize: 10.5, color: 'var(--ink-faint)', fontWeight: 500, fontFamily: 'var(--mono)' }}>
                    {step.sub}
                  </div>
                </div>
              </div>
              {i < STEPS.length - 1 && (
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
