import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

const NAV_ITEMS = [
  { path: '/admin/users', key: 'admin.users' },
  { path: '/admin/quota', key: 'admin.quota' },
  { path: '/admin/translations', key: 'admin.translations' },
  { path: '/admin/records', key: 'admin.records' },
] as const

export default function AdminLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      {/* Sidebar */}
      <aside style={{
        width: 220,
        background: 'var(--panel, #f8f9fa)',
        borderRight: '1px solid var(--line, #e0e0e0)',
        padding: '24px 0',
        display: 'flex',
        flexDirection: 'column',
      }}>
        <div style={{ padding: '0 20px', marginBottom: 24 }}>
          <button
            onClick={() => navigate('/landing')}
            style={{
              fontSize: 12, color: 'var(--ink-faint, #888)',
              background: 'none', border: 'none', cursor: 'pointer',
              padding: 0, fontFamily: 'var(--mono)',
            }}
          >
            ← {t('header.platform_name')}
          </button>
          <h2 style={{ fontSize: 16, fontWeight: 650, marginTop: 12 }}>
            {t('header.admin')}
          </h2>
        </div>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              style={({ isActive }) => ({
                display: 'block',
                padding: '10px 20px',
                fontSize: 14,
                fontWeight: isActive ? 600 : 400,
                color: isActive ? 'var(--accent, #2563eb)' : 'var(--ink-soft, #555)',
                background: isActive ? 'var(--accent-soft, #eff6ff)' : 'transparent',
                textDecoration: 'none',
                borderLeft: isActive ? '3px solid var(--accent, #2563eb)' : '3px solid transparent',
              })}
            >
              {t(item.key)}
            </NavLink>
          ))}
        </nav>
      </aside>

      {/* Main content */}
      <main style={{ flex: 1, padding: 32, maxWidth: 960 }}>
        <Outlet />
      </main>
    </div>
  )
}
