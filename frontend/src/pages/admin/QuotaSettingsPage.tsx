import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import apiClient from '../../api/client'

interface QuotaSettings {
  max_assessments: number
  reset_period: string
}

export default function QuotaSettingsPage() {
  const { t } = useTranslation()
  const [maxAssessments, setMaxAssessments] = useState(5)
  const [resetPeriod, setResetPeriod] = useState('daily')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')

  useEffect(() => {
    apiClient.get<QuotaSettings>('/admin/quota')
      .then((res) => {
        setMaxAssessments(res.data.max_assessments)
        setResetPeriod(res.data.reset_period)
      })
      .catch(() => {
        // use defaults
      })
      .finally(() => setLoading(false))
  }, [])

  const handleSave = async () => {
    setSaving(true)
    setMessage('')
    try {
      await apiClient.put('/admin/quota', {
        max_assessments: maxAssessments,
        reset_period: resetPeriod,
      })
      setMessage(t('admin.save_success'))
    } catch {
      setMessage(t('admin.save_failed'))
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <p style={{ color: 'var(--ink-faint)' }}>{t('common.loading')}...</p>
  }

  return (
    <div>
      <h1 style={{ fontSize: 22, fontWeight: 650, marginBottom: 24 }}>{t('admin.quota')}</h1>

      <div style={{
        background: 'var(--paper, #fff)',
        border: '1px solid var(--line, #e0e0e0)',
        borderRadius: 12,
        padding: 24,
        maxWidth: 420,
      }}>
        <div style={{ marginBottom: 20 }}>
          <label style={{ display: 'block', fontSize: 13, fontWeight: 550, marginBottom: 6 }}>
            {t('admin.max_assessments')}
          </label>
          <input
            type="number"
            min={1}
            value={maxAssessments}
            onChange={(e) => setMaxAssessments(Number(e.target.value))}
            style={{
              width: '100%', padding: '8px 12px', fontSize: 14,
              border: '1px solid var(--line, #e0e0e0)', borderRadius: 8,
              fontFamily: 'var(--mono)',
            }}
          />
        </div>

        <div style={{ marginBottom: 24 }}>
          <label style={{ display: 'block', fontSize: 13, fontWeight: 550, marginBottom: 6 }}>
            {t('admin.reset_period')}
          </label>
          <select
            value={resetPeriod}
            onChange={(e) => setResetPeriod(e.target.value)}
            style={{
              width: '100%', padding: '8px 12px', fontSize: 14,
              border: '1px solid var(--line, #e0e0e0)', borderRadius: 8,
            }}
          >
            <option value="daily">{t('admin.daily')}</option>
            <option value="weekly">{t('admin.weekly')}</option>
          </select>
        </div>

        <button
          onClick={handleSave}
          disabled={saving}
          style={{
            padding: '10px 20px', fontSize: 14, fontWeight: 600,
            background: 'var(--accent, #2563eb)', color: '#fff',
            border: 'none', borderRadius: 8, cursor: saving ? 'not-allowed' : 'pointer',
            opacity: saving ? 0.7 : 1,
          }}
        >
          {saving ? t('common.running') : t('btn.save')}
        </button>

        {message && (
          <p style={{
            marginTop: 12, fontSize: 13, fontWeight: 500,
            color: message === t('admin.save_success') ? 'var(--green, #16a34a)' : 'var(--rose, #dc2626)',
          }}>
            {message}
          </p>
        )}
      </div>
    </div>
  )
}
