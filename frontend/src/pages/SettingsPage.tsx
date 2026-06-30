import { useState, useEffect } from 'react'
import Slider from 'rc-slider'
import 'rc-slider/assets/index.css'
import apiClient from '../api/client'

interface Weights {
  row_completeness: number
  column_completeness: number
  format_consistency: number
  duplicate_similar: number
  table_structure: number
  ai_query_readiness: number
}

const WEIGHT_FIELDS: { key: keyof Weights; label: string; labelEn: string }[] = [
  { key: 'row_completeness', label: '列完整度', labelEn: 'Row Completeness' },
  { key: 'column_completeness', label: '欄完整度', labelEn: 'Column Completeness' },
  { key: 'format_consistency', label: '格式一致性', labelEn: 'Format Consistency' },
  { key: 'duplicate_similar', label: '重複/近似', labelEn: 'Duplicate/Similar' },
  { key: 'table_structure', label: '表格結構', labelEn: 'Table Structure' },
  { key: 'ai_query_readiness', label: 'AI 查詢準備度', labelEn: 'AI Query Readiness' },
]

const DEFAULT_WEIGHTS: Weights = {
  row_completeness: 20,
  column_completeness: 20,
  format_consistency: 15,
  duplicate_similar: 10,
  table_structure: 15,
  ai_query_readiness: 20,
}

export default function SettingsPage() {
  const [weights, setWeights] = useState<Weights>(DEFAULT_WEIGHTS)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    apiClient.get('/settings/weights').then((res) => {
      const data = res.data
      setWeights({
        row_completeness: (data.row_completeness ?? 0.2) * 100,
        column_completeness: (data.column_completeness ?? 0.2) * 100,
        format_consistency: (data.format_consistency ?? 0.15) * 100,
        duplicate_similar: (data.duplicate_similar ?? 0.1) * 100,
        table_structure: (data.table_structure ?? 0.15) * 100,
        ai_query_readiness: (data.ai_query_readiness ?? 0.2) * 100,
      })
    }).catch(() => {})
  }, [])

  const sum = Object.values(weights).reduce((a, b) => a + b, 0)
  const isValid = Math.abs(sum - 100) < 0.5

  const handleChange = (key: keyof Weights, value: number) => {
    setWeights((prev) => ({ ...prev, [key]: value }))
    setSaved(false)
  }

  const handleSave = async () => {
    setError('')
    setSaving(true)
    try {
      await apiClient.put('/settings/weights', {
        row_completeness: weights.row_completeness / 100,
        column_completeness: weights.column_completeness / 100,
        format_consistency: weights.format_consistency / 100,
        duplicate_similar: weights.duplicate_similar / 100,
        table_structure: weights.table_structure / 100,
        ai_query_readiness: weights.ai_query_readiness / 100,
      })
      setSaved(true)
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: { message?: string } } } }
      setError(axiosErr.response?.data?.error?.message || '儲存失敗')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div style={{
      background: 'var(--paper)', border: '1px solid var(--line)',
      borderRadius: 14, overflow: 'hidden',
    }}>
      {/* Header */}
      <div style={{ padding: '20px 28px', borderBottom: '1px solid var(--line-soft)' }}>
        <div style={{
          fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--accent)',
          letterSpacing: '0.08em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 5,
        }}>SETTINGS</div>
        <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>權重設定</h2>
        <p style={{ color: 'var(--ink-soft)', fontSize: 14, marginTop: 5 }}>
          調整各項指標在總分計算中的權重比例
        </p>
      </div>

      {/* Body */}
      <div style={{ padding: 28 }}>
        {error && (
          <div style={{
            background: 'var(--rose-soft)', color: 'var(--rose)',
            padding: '10px 14px', borderRadius: 'var(--radius-sm)',
            fontSize: 13, fontWeight: 500, marginBottom: 16,
          }}>
            {error}
          </div>
        )}

        {/* Sliders */}
        <div style={{ marginBottom: 24 }}>
          {WEIGHT_FIELDS.map((field) => (
            <div key={field.key} style={{ marginBottom: 24 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 10 }}>
                <div>
                  <span style={{ fontSize: 14, fontWeight: 600 }}>{field.label}</span>
                  <span style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)', marginLeft: 8 }}>
                    {field.labelEn}
                  </span>
                </div>
                <span style={{ fontFamily: 'var(--mono)', fontSize: 14, fontWeight: 600, color: 'var(--ink)' }}>
                  {weights[field.key].toFixed(0)}%
                </span>
              </div>
              <Slider
                min={0}
                max={50}
                step={1}
                value={weights[field.key]}
                onChange={(val) => handleChange(field.key, val as number)}
                styles={{
                  track: { background: 'var(--accent)', height: 6 },
                  rail: { background: 'var(--line-soft)', height: 6 },
                  handle: {
                    borderColor: 'var(--accent)',
                    width: 18,
                    height: 18,
                    marginTop: -6,
                    background: 'var(--paper)',
                    boxShadow: '0 1px 4px rgba(0,0,0,0.15)',
                    opacity: 1,
                  },
                }}
              />
            </div>
          ))}
        </div>

        {/* Sum display */}
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '14px 18px', borderRadius: 'var(--radius)',
          background: isValid ? 'var(--green-soft)' : 'var(--amber-soft)',
          border: `1px solid ${isValid ? '#cfe8d8' : '#f0dbb8'}`,
        }}>
          <span style={{ fontSize: 14, fontWeight: 600, color: isValid ? 'var(--green)' : 'var(--amber)' }}>
            權重總和
          </span>
          <span style={{
            fontFamily: 'var(--mono)', fontSize: 18, fontWeight: 700,
            color: isValid ? 'var(--green)' : 'var(--amber)',
          }}>
            {sum.toFixed(0)}%
          </span>
        </div>

        {!isValid && (
          <div style={{
            fontSize: 12.5, color: 'var(--amber)', marginTop: 8, fontWeight: 500,
          }}>
            ⚠ 權重總和必須為 100%，目前為 {sum.toFixed(0)}%
          </div>
        )}

        {saved && (
          <div style={{
            fontSize: 12.5, color: 'var(--green)', marginTop: 8, fontWeight: 500,
          }}>
            ✓ 權重已儲存成功
          </div>
        )}
      </div>

      {/* Footer */}
      <div style={{
        padding: '16px 28px', borderTop: '1px solid var(--line-soft)',
        display: 'flex', alignItems: 'center', justifyContent: 'flex-end',
        background: 'var(--panel)',
      }}>
        <button
          className="btn btn-primary"
          onClick={handleSave}
          disabled={!isValid || saving}
        >
          {saving ? '儲存中...' : '儲存設定'}
        </button>
      </div>
    </div>
  )
}
