import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import apiClient from '../../api/client'

interface UserRecord {
  id: string
  username: string
  role: string
  used_count: number
  remaining: number
}

interface UsersResponse {
  users: UserRecord[]
  total: number
  page: number
  page_size: number
}

export default function UsersPage() {
  const { t } = useTranslation()
  const [users, setUsers] = useState<UserRecord[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const pageSize = 20

  useEffect(() => {
    setLoading(true)
    apiClient.get<UsersResponse>(`/admin/users?page=${page}&page_size=${pageSize}`)
      .then((res) => {
        setUsers(res.data.users || [])
        setTotal(res.data.total || 0)
      })
      .catch(() => {
        setUsers([])
      })
      .finally(() => setLoading(false))
  }, [page])

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div>
      <h1 style={{ fontSize: 22, fontWeight: 650, marginBottom: 24 }}>{t('admin.users')}</h1>

      {loading ? (
        <p style={{ color: 'var(--ink-faint)' }}>{t('common.loading')}...</p>
      ) : (
        <>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
              <thead>
                <tr style={{ borderBottom: '2px solid var(--line, #e0e0e0)' }}>
                  <th style={{ textAlign: 'left', padding: '10px 12px', fontWeight: 600 }}>Account</th>
                  <th style={{ textAlign: 'left', padding: '10px 12px', fontWeight: 600 }}>Role</th>
                  <th style={{ textAlign: 'right', padding: '10px 12px', fontWeight: 600 }}>{t('admin.used_quota')}</th>
                  <th style={{ textAlign: 'right', padding: '10px 12px', fontWeight: 600 }}>{t('admin.remaining_quota')}</th>
                </tr>
              </thead>
              <tbody>
                {users.map((u) => (
                  <tr key={u.id} style={{ borderBottom: '1px solid var(--line-soft, #f0f0f0)' }}>
                    <td style={{ padding: '10px 12px', fontFamily: 'var(--mono)' }}>{u.username}</td>
                    <td style={{ padding: '10px 12px' }}>
                      <span style={{
                        fontSize: 11, fontWeight: 600, textTransform: 'uppercase',
                        padding: '2px 8px', borderRadius: 4,
                        background: u.role === 'admin' ? '#fef3c7' : '#e0f2fe',
                        color: u.role === 'admin' ? '#92400e' : '#0369a1',
                      }}>
                        {u.role}
                      </span>
                    </td>
                    <td style={{ padding: '10px 12px', textAlign: 'right', fontFamily: 'var(--mono)' }}>{u.used_count}</td>
                    <td style={{ padding: '10px 12px', textAlign: 'right', fontFamily: 'var(--mono)' }}>{u.remaining}</td>
                  </tr>
                ))}
                {users.length === 0 && (
                  <tr>
                    <td colSpan={4} style={{ padding: 20, textAlign: 'center', color: 'var(--ink-faint)' }}>
                      No users found
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div style={{ display: 'flex', justifyContent: 'center', gap: 8, marginTop: 20 }}>
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                style={{ padding: '6px 12px', fontSize: 13, borderRadius: 6, border: '1px solid var(--line)', cursor: page === 1 ? 'not-allowed' : 'pointer', opacity: page === 1 ? 0.5 : 1 }}
              >
                ←
              </button>
              <span style={{ padding: '6px 12px', fontSize: 13, fontFamily: 'var(--mono)' }}>
                {page} / {totalPages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
                style={{ padding: '6px 12px', fontSize: 13, borderRadius: 6, border: '1px solid var(--line)', cursor: page === totalPages ? 'not-allowed' : 'pointer', opacity: page === totalPages ? 0.5 : 1 }}
              >
                →
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
