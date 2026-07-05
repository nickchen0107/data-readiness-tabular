import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import apiClient from '../api/client'

interface EvidenceRecord {
  record_id: string
  dataset_hash: string
  log_hash: string
  report_hash: string
  signature_status: string
  timestamp: string
  transaction_hash?: string
  t3_cid?: string
  t3_token_id?: string
  t3_tx_id?: string
  t3_minted_at?: string
}

export default function EvidencePage() {
  const navigate = useNavigate()
  const { t } = useTranslation()
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
      setError(axiosErr.response?.data?.error?.message || t('error.evidence_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  const statusLabel = record?.signature_status === 'confirmed' ? t('evidence.confirmed')
    : record?.signature_status === 'pending' ? t('evidence.pending') : t('evidence.demo_mode')
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
        <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>{t('page.evidence.title')}</h2>
        <p style={{ color: 'var(--ink-soft)', fontSize: 14, marginTop: 5 }}>
          {t('page.evidence.desc')}
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
            <h3 style={{ fontSize: 18, fontWeight: 650, marginBottom: 8 }}>{t('evidence.submit_title')}</h3>
            <p style={{ fontSize: 13.5, color: 'var(--ink-soft)', marginBottom: 24, maxWidth: 400, margin: '0 auto 24px' }}>
              {t('evidence.submit_desc')}
            </p>
            <button
              className="btn btn-primary"
              onClick={handleSubmit}
              disabled={submitting || !sessionId}
              style={{ padding: '12px 24px' }}
            >
              {submitting ? t('evidence.submitting') : t('evidence.submit_btn')}
            </button>
            <div style={{
              marginTop: 16, fontSize: 12, color: 'var(--ink-faint)',
              fontFamily: 'var(--mono)',
            }}>
              {t('evidence.demo_note')}
            </div>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
            {/* Evidence card */}
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
                  {t('evidence.integrity_title')}
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

              {/* Core evidence info */}
              <div style={{ padding: '8px 26px 20px' }}>
                {[
                  { key: t('evidence.record_id'), value: record.record_id },
                  { key: t('evidence.transaction_id'), value: record.transaction_hash || record.t3_tx_id || '-' },
                  { key: t('evidence.signature_status'), value: statusLabel },
                  { key: t('evidence.timestamp'), value: record.timestamp ? new Date(record.timestamp).toLocaleString('zh-TW') : '-' },
                ].map((row) => (
                  <div key={row.key} style={{
                    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                    padding: '12px 0', borderBottom: '1px solid var(--line-soft)',
                    fontSize: 13,
                  }}>
                    <span style={{ color: 'var(--ink-soft)', fontWeight: 500 }}>{row.key}</span>
                    <span style={{
                      fontFamily: 'var(--mono)', fontSize: 12, color: 'var(--ink)',
                      maxWidth: 360, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                    }}>
                      {row.value}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            {/* Lineage: Data Artifacts */}
            <div style={{ border: '1px solid var(--line)', borderRadius: 14, overflow: 'hidden' }}>
              <div style={{
                padding: '16px 26px', borderBottom: '1px solid var(--line-soft)',
                background: 'var(--panel)',
              }}>
                <h4 style={{ fontSize: 15, fontWeight: 650, margin: 0 }}>
                  📦 {t('evidence.lineage_title')}
                </h4>
                <p style={{ fontSize: 12.5, color: 'var(--ink-soft)', margin: '4px 0 0' }}>
                  {t('evidence.lineage_desc')}
                </p>
              </div>

              <div style={{ padding: '4px 26px 16px' }}>
                {/* Raw Dataset (processed Excel) */}
                <div style={{ padding: '14px 0', borderBottom: '1px solid var(--line-soft)' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                    <span style={{ fontSize: 16 }}>📊</span>
                    <span style={{ fontSize: 13, fontWeight: 600 }}>{t('evidence.artifact_raw_dataset')}</span>
                    <span style={{
                      fontSize: 10, fontFamily: 'var(--mono)', color: 'var(--accent)',
                      background: 'var(--accent-soft)', padding: '2px 6px', borderRadius: 4,
                    }}>IPFS</span>
                  </div>
                  <div style={{ fontSize: 12, fontFamily: 'var(--mono)', color: 'var(--ink-soft)', lineHeight: 1.8 }}>
                    <div><span style={{ color: 'var(--ink-faint)', marginRight: 8 }}>SHA-256:</span>{record.dataset_hash}</div>
                    {record.t3_cid && <div><span style={{ color: 'var(--ink-faint)', marginRight: 8 }}>IPFS CID:</span>{record.t3_cid}</div>}
                  </div>
                </div>

                {/* Processed Dataset (PDF report) */}
                <div style={{ padding: '14px 0', borderBottom: '1px solid var(--line-soft)' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                    <span style={{ fontSize: 16 }}>📄</span>
                    <span style={{ fontSize: 13, fontWeight: 600 }}>{t('evidence.artifact_processed_report')}</span>
                    <span style={{
                      fontSize: 10, fontFamily: 'var(--mono)', color: 'var(--accent)',
                      background: 'var(--accent-soft)', padding: '2px 6px', borderRadius: 4,
                    }}>IPFS</span>
                  </div>
                  <div style={{ fontSize: 12, fontFamily: 'var(--mono)', color: 'var(--ink-soft)', lineHeight: 1.8 }}>
                    <div><span style={{ color: 'var(--ink-faint)', marginRight: 8 }}>SHA-256:</span>{record.report_hash}</div>
                    {record.t3_token_id && <div><span style={{ color: 'var(--ink-faint)', marginRight: 8 }}>IPFS CID:</span>{record.t3_token_id}</div>}
                  </div>
                </div>

                {/* Cleaning Log (hash-only) */}
                <div style={{ padding: '14px 0' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                    <span style={{ fontSize: 16 }}>📋</span>
                    <span style={{ fontSize: 13, fontWeight: 600 }}>{t('evidence.artifact_cleaning_log')}</span>
                    <span style={{
                      fontSize: 10, fontFamily: 'var(--mono)', color: 'var(--ink-faint)',
                      background: 'var(--panel)', padding: '2px 6px', borderRadius: 4, border: '1px solid var(--line)',
                    }}>Hash Only</span>
                  </div>
                  <div style={{ fontSize: 12, fontFamily: 'var(--mono)', color: 'var(--ink-soft)', lineHeight: 1.8 }}>
                    <div><span style={{ color: 'var(--ink-faint)', marginRight: 8 }}>SHA-256:</span>{record.log_hash}</div>
                  </div>
                </div>
              </div>
            </div>

            {/* Flags */}
            <div style={{
              display: 'flex', gap: 9, flexWrap: 'wrap',
              padding: '16px 20px', background: 'var(--green-soft)',
              borderRadius: 'var(--radius)', border: '1px solid #cfe8d8',
            }}>
              <span style={{
                fontFamily: 'var(--mono)', fontSize: 11.5, color: 'var(--green)',
                fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6,
              }}>
                ✓ {t('evidence.flag_no_sensitive')}
              </span>
              <span style={{
                fontFamily: 'var(--mono)', fontSize: 11.5, color: 'var(--green)',
                fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6,
              }}>
                ✓ {t('evidence.flag_verifiable')}
              </span>
              <span style={{
                fontFamily: 'var(--mono)', fontSize: 11.5, color: 'var(--green)',
                fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6,
              }}>
                ✓ {t('evidence.flag_immutable')}
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
            {t('evidence.complete_hint')}
          </span>
          <button className="btn btn-primary" onClick={() => navigate('/qa')}>
            {t('btn.next_step')} →
          </button>
        </div>
      )}
    </div>
  )
}
