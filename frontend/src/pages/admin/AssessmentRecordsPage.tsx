import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import apiClient from '../../api/client'

interface AssessmentRecord {
  id: string
  filename: string
  total_score: number
  status: string
  created_at: string
}

interface AssessmentListResponse {
  assessments: AssessmentRecord[]
  total: number
  page: number
  page_size: number
}

export default function AssessmentRecordsPage() {
  const { t } = useTranslation()
  const [userFilter, setUserFilter] = useState('')
  const [assessments, setAssessments] = useState<AssessmentRecord[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const pageSize = 20

  useEffect(() => {
    setLoading(true)
    const params = new URLSearchParams({
      page: String(page),
      page_size: String(pageSize),
    })
    if (userFilter.trim()) params.set('user_id', userFilter.trim())

    apiClient.get<AssessmentListResponse>(`/admin/assessments?${params.toString()}`)
      .then((res) => {
        setAssessments(res.data.assessments || [])
        setTotal(res.data.total || 0)
      })
      .catch(() => {
        setAssessments([])
      })
      .finally(() => setLoading(false))
  }, [page, userFilter])

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div>
      <h1 style={{ fontSize: 22, fontWeight: 650, marginBottom: 24 }}>{t('admin.records')}</h1>

      {/* User filter */}
      <div style={{ marginBottom: 20 }}>
        <input
          type="text"
          placeholder="User ID or email..."
          value={userFilter}
          onChange={(e) => { setUserFilter(e.target.value); setPage(1) }}
          style={{
            padding: '8px 12px', fontSize: 13, width: 300,
            border: '1px solid var(--line, #e0e0e0)', borderRadius: 8,
          }}
        />
      </div>

      {loading ? (
        <p style={{ color: 'var(--ink-faint)' }}>{t('common.loading')}...</p>
      ) : (
        <>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
              <thead>
                <tr style={{ borderBottom: '2px solid var(--line, #e0e0e0)' }}>
                  <th style={{ textAlign: 'left', padding: '10px 12px', fontWeight: 600 }}>Timestamp</th>
                  <th style={{ textAlign: 'left', padding: '10px 12px', fontWeight: 600 }}>Filename</th>
                  <th style={{ textAlign: 'right', padding: '10px 12px', fontWeight: 600 }}>{t('common.total_score')}</th>
                  <th style={{ textAlign: 'center', padding: '10px 12px', fontWeight: 600 }}>Status</th>
                </tr>
              </thead>
              <tbody>
                {assessments.map((a) => (
                  <tr key={a.id} style={{ borderBottom: '1px solid var(--line-soft, #f0f0f0)' }}>
                    <td style={{ padding: '10px 12px', fontFamily: 'var(--mono)', fontSize: 12 }}>
                      {new Date(a.created_at).toLocaleString()}
                    </td>
                    <td style={{ padding: '10px 12px' }}>{a.filename}</td>
                    <td style={{ padding: '10px 12px', textAlign: 'right', fontFamily: 'var(--mono)' }}>
                      {a.total_score}
                    </td>
                    <td style={{ padding: '10px 12px', textAlign: 'center' }}>
                      <span style={{
                        fontSize: 11, fontWeight: 600,
                        padding: '2px 8px', borderRadius: 4,
                        background: a.status === 'ready' ? '#dcfce7' : a.status === 'conditional' ? '#fef9c3' : '#fee2e2',
                        color: a.status === 'ready' ? '#166534' : a.status === 'conditional' ? '#854d0e' : '#991b1b',
                      }}>
                        {a.status}
                      </span>
                    </td>
                  </tr>
                ))}
                {assessments.length === 0 && (
                  <tr>
                    <td colSpan={4} style={{ padding: 20, textAlign: 'center', color: 'var(--ink-faint)' }}>
                      No records found
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
