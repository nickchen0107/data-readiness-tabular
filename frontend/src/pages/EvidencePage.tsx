import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import apiClient from '../api/client'

interface EvidenceRecord {
  record_id: string
  dataset_hash: string
  log_hash: string
  report_hash: string
  signature_status: string
  timestamp: string
  transaction_hash?: string
}

export default function EvidencePage() {
  const navigate = useNavigate()
  const [sessionId, setSessionId] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [record, setRecord] = useState<EvidenceRecord | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    apiClient.get('/clean/latest').then((res) => {
      setSessionId(res.data.session_id || res.data.id)
    }).catch(() => {})
  }, [])

  const handleSubmit = async () => {
    if (!sessionId) return
    setSubmitting(true)
    setError('')
    try {
      const res = await apiClient.post('/evidence/submit', { session_id: sessionId })
      setRecord(res.data)
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: { message?: string } } } }
      setError(axiosErr.response?.data?.error?.message || '存證提交失敗')
    } finally {
      setSubmitting(false)
    }
  }

  const statusLabel = record?.signature_status === 'confirmed' ? '已確認'
    : record?.signature_status === 'pending' ? '待確認' : 'Demo 模式'
  const isDemo = record?.signature_status === 'demo'

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
        }}>STEP 6</div>
        <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>存證紀錄</h2>
        <p style={{ color: 'var(--ink-soft)', fontSize: 14, marginTop: 5 }}>
          將梳理結果的 Hash 提交至區塊鏈進行完整性存證
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

        {!record ? (
          /* Submit button */
          <div style={{ textAlign: 'center', padding: '40px 20px' }}>
            <div style={{ fontSize: 48, marginBottom: 16 }}>🛡️</div>
            <h3 style={{ fontSize: 18, fontWeight: 650, marginBottom: 8 }}>準備提交存證</h3>
            <p style={{ fontSize: 13.5, color: 'var(--ink-soft)', marginBottom: 24, maxWidth: 400, margin: '0 auto 24px' }}>
              系統將計算資料集、梳理紀錄及報告的 SHA-256 雜湊值，並提交至區塊鏈存證
            </p>
            <button
              className="btn btn-primary"
              onClick={handleSubmit}
              disabled={submitting || !sessionId}
              style={{ padding: '12px 24px' }}
            >
              {submitting ? '提交中...' : '提交存證'}
            </button>
            <div style={{
              marginTop: 16, fontSize: 12, color: 'var(--ink-faint)',
              fontFamily: 'var(--mono)',
            }}>
              Demo Mode — 存證功能為展示用途
            </div>
          </div>
        ) : (
          /* Evidence card */
          <div style={{ border: '1px solid var(--line)', borderRadius: 14, overflow: 'hidden' }}>
            {/* Dark header */}
            <div style={{
              background: 'var(--ink)', color: '#fff', padding: '24px 26px', textAlign: 'center',
            }}>
              <div style={{
                width: 54, height: 54, borderRadius: '50%',
                background: 'rgba(255,255,255,0.12)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: 26, margin: '0 auto 12px',
              }}>🛡️</div>
              <h3 style={{ fontSize: 18, fontWeight: 650, letterSpacing: '-0.01em' }}>
                資料完整性存證
              </h3>
              <div style={{
                fontFamily: 'var(--mono)', fontSize: 12, color: '#9fb8d4',
                marginTop: 8, lineHeight: 1.6,
              }}>
                Data Integrity Evidence Record
                {isDemo && (
                  <span style={{
                    display: 'inline-block', marginLeft: 8,
                    background: 'rgba(255,210,122,0.2)', color: '#ffd27a',
                    padding: '2px 8px', borderRadius: 4, fontSize: 10,
                  }}>
                    DEMO
                  </span>
                )}
              </div>
            </div>

            {/* Evidence rows */}
            <div style={{ padding: '8px 26px 20px' }}>
              {[
                { key: 'Record ID', value: record.record_id },
                { key: 'Dataset Hash', value: record.dataset_hash },
                { key: 'Log Hash', value: record.log_hash },
                { key: 'Report Hash', value: record.report_hash },
                { key: '時間戳', value: record.timestamp },
                { key: '簽章狀態', value: statusLabel },
              ].map((row) => (
                <div key={row.key} style={{
                  display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                  padding: '12px 0', borderBottom: '1px solid var(--line-soft)',
                  fontSize: 13,
                }}>
                  <span style={{ color: 'var(--ink-soft)', fontWeight: 500 }}>{row.key}</span>
                  <span style={{
                    fontFamily: 'var(--mono)', fontSize: 12, color: 'var(--ink)',
                    maxWidth: 320, overflow: 'hidden', textOverflow: 'ellipsis',
                  }}>
                    {row.value}
                  </span>
                </div>
              ))}
            </div>

            {/* Flags */}
            <div style={{
              display: 'flex', gap: 9, flexWrap: 'wrap',
              padding: '16px 26px', background: 'var(--green-soft)',
              borderTop: '1px solid #cfe8d8',
            }}>
              <span style={{
                fontFamily: 'var(--mono)', fontSize: 11.5, color: 'var(--green)',
                fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6,
              }}>
                ✓ No sensitive data on-chain
              </span>
              <span style={{
                fontFamily: 'var(--mono)', fontSize: 11.5, color: 'var(--green)',
                fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6,
              }}>
                ✓ Integrity verifiable
              </span>
            </div>
          </div>
        )}
      </div>

      {/* Footer */}
      {record && (
        <div style={{
          padding: '16px 28px', borderTop: '1px solid var(--line-soft)',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          background: 'var(--panel)',
        }}>
          <span style={{ fontSize: 12.5, color: 'var(--ink-faint)' }}>
            存證完成，可進行問答對比
          </span>
          <button className="btn btn-primary" onClick={() => navigate('/qa')}>
            下一步 →
          </button>
        </div>
      )}
    </div>
  )
}
