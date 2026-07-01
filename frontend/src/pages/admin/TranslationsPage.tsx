import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import apiClient from '../../api/client'

interface TranslationEntry {
  id: string
  locale: string
  key: string
  value: string
  updated_at: string
}

interface TranslationsResponse {
  translations: TranslationEntry[]
  total: number
  page: number
  page_size: number
}

export default function TranslationsPage() {
  const { t } = useTranslation()
  const [locale, setLocale] = useState('zh-TW')
  const [search, setSearch] = useState('')
  const [translations, setTranslations] = useState<TranslationEntry[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [saveMessage, setSaveMessage] = useState('')
  const pageSize = 30

  const fetchTranslations = useCallback(() => {
    setLoading(true)
    const params = new URLSearchParams({
      locale,
      page: String(page),
      page_size: String(pageSize),
    })
    if (search) params.set('search', search)

    apiClient.get<TranslationsResponse>(`/admin/translations?${params.toString()}`)
      .then((res) => {
        setTranslations(res.data.translations || [])
        setTotal(res.data.total || 0)
      })
      .catch(() => {
        setTranslations([])
      })
      .finally(() => setLoading(false))
  }, [locale, page, search])

  useEffect(() => {
    fetchTranslations()
  }, [fetchTranslations])

  const handleSave = async (id: string) => {
    setSaveMessage('')
    try {
      await apiClient.put(`/admin/translations/${id}`, { value: editValue })
      setTranslations((prev) =>
        prev.map((tr) => tr.id === id ? { ...tr, value: editValue } : tr)
      )
      setEditingId(null)
      setSaveMessage(t('admin.save_success'))
      setTimeout(() => setSaveMessage(''), 2000)
    } catch {
      setSaveMessage(t('admin.save_failed'))
    }
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div>
      <h1 style={{ fontSize: 22, fontWeight: 650, marginBottom: 24 }}>{t('admin.translations')}</h1>

      {/* Filters */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap' }}>
        <select
          value={locale}
          onChange={(e) => { setLocale(e.target.value); setPage(1) }}
          style={{
            padding: '8px 12px', fontSize: 13,
            border: '1px solid var(--line, #e0e0e0)', borderRadius: 8,
          }}
        >
          <option value="zh-TW">zh-TW</option>
          <option value="en">en</option>
        </select>
        <input
          type="text"
          placeholder={t('admin.search_placeholder')}
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1) }}
          style={{
            flex: 1, minWidth: 200, padding: '8px 12px', fontSize: 13,
            border: '1px solid var(--line, #e0e0e0)', borderRadius: 8,
          }}
        />
      </div>

      {saveMessage && (
        <p style={{ fontSize: 13, fontWeight: 500, marginBottom: 12, color: 'var(--green, #16a34a)' }}>
          {saveMessage}
        </p>
      )}

      {loading ? (
        <p style={{ color: 'var(--ink-faint)' }}>{t('common.loading')}...</p>
      ) : (
        <>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: '2px solid var(--line, #e0e0e0)' }}>
                  <th style={{ textAlign: 'left', padding: '8px 10px', fontWeight: 600, width: '35%' }}>Key</th>
                  <th style={{ textAlign: 'left', padding: '8px 10px', fontWeight: 600, width: '50%' }}>Value</th>
                  <th style={{ textAlign: 'center', padding: '8px 10px', fontWeight: 600, width: '15%' }}></th>
                </tr>
              </thead>
              <tbody>
                {translations.map((tr) => (
                  <tr key={tr.id} style={{ borderBottom: '1px solid var(--line-soft, #f0f0f0)' }}>
                    <td style={{ padding: '8px 10px', fontFamily: 'var(--mono)', fontSize: 12, color: 'var(--ink-faint)' }}>
                      {tr.key}
                    </td>
                    <td style={{ padding: '8px 10px' }}>
                      {editingId === tr.id ? (
                        <input
                          type="text"
                          value={editValue}
                          onChange={(e) => setEditValue(e.target.value)}
                          style={{
                            width: '100%', padding: '4px 8px', fontSize: 13,
                            border: '1px solid var(--accent, #2563eb)', borderRadius: 4,
                          }}
                          onKeyDown={(e) => { if (e.key === 'Enter') handleSave(tr.id) }}
                        />
                      ) : (
                        <span>{tr.value}</span>
                      )}
                    </td>
                    <td style={{ padding: '8px 10px', textAlign: 'center' }}>
                      {editingId === tr.id ? (
                        <div style={{ display: 'flex', gap: 4, justifyContent: 'center' }}>
                          <button
                            onClick={() => handleSave(tr.id)}
                            style={{
                              fontSize: 12, padding: '4px 10px', borderRadius: 4,
                              background: 'var(--accent, #2563eb)', color: '#fff',
                              border: 'none', cursor: 'pointer',
                            }}
                          >
                            {t('btn.save')}
                          </button>
                          <button
                            onClick={() => setEditingId(null)}
                            style={{
                              fontSize: 12, padding: '4px 10px', borderRadius: 4,
                              background: 'var(--panel, #f5f5f5)', color: 'var(--ink-soft)',
                              border: '1px solid var(--line)', cursor: 'pointer',
                            }}
                          >
                            {t('btn.cancel')}
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => { setEditingId(tr.id); setEditValue(tr.value) }}
                          style={{
                            fontSize: 12, padding: '4px 10px', borderRadius: 4,
                            background: 'var(--panel, #f5f5f5)', color: 'var(--ink-soft)',
                            border: '1px solid var(--line)', cursor: 'pointer',
                          }}
                        >
                          ✏️
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
                {translations.length === 0 && (
                  <tr>
                    <td colSpan={3} style={{ padding: 20, textAlign: 'center', color: 'var(--ink-faint)' }}>
                      No translations found
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
