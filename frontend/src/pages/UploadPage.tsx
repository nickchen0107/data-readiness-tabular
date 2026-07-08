import { useState, useRef, useEffect, type DragEvent, type ChangeEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import apiClient from '../api/client'
import { useStepper } from '../contexts/StepperContext'

interface UploadResult {
  id: string
  filename: string
  row_count: number
  col_count: number
  sheet_names: string[]
}

interface QuotaInfo {
  remaining: number
  max_assessments: number
  used_count: number
}

export default function UploadPage() {
  const { t } = useTranslation()
  const [dragOver, setDragOver] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [uploadResult, setUploadResult] = useState<UploadResult | null>(null)
  const [selectedSheet, setSelectedSheet] = useState('')
  const [error, setError] = useState('')
  const [assessing, setAssessing] = useState(false)
  const [quota, setQuota] = useState<QuotaInfo | null>(null)
  const [isHistoryView, setIsHistoryView] = useState(false)
  const hasUserInteracted = useRef(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()
  const { resetProgress, maxReachedStep } = useStepper()

  useEffect(() => {
    apiClient.get('/quota/me').then((res) => {
      setQuota(res.data)
    }).catch(() => {})

    // Load latest assessment to show upload info on revisit — only if user has reached step 2+
    if (maxReachedStep >= 2) {
      apiClient.get('/assess/latest').then((res) => {
        if (hasUserInteracted.current) return
        if (res.data && res.data.id && res.data.upload_id && res.data.filename) {
          setUploadResult((current) => {
            if (current !== null) return current
            setSelectedSheet(res.data.selected_sheet || '')
            setIsHistoryView(true)
            return {
              id: res.data.upload_id,
              filename: res.data.filename,
              row_count: res.data.total_rows || 0,
              col_count: 0,
              sheet_names: [],
            }
          })
        }
      }).catch(() => {})
    }
  }, [])

  const handleFile = async (file: File) => {
    const validTypes = [
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      'text/csv',
      'application/vnd.ms-excel',
    ]
    const ext = file.name.split('.').pop()?.toLowerCase()
    if (!validTypes.includes(file.type) && ext !== 'xlsx' && ext !== 'csv') {
      setError(t('error.file_format'))
      return
    }

    setError('')
    setUploadResult(null)
    setSelectedSheet('')
    setIsHistoryView(false)
    hasUserInteracted.current = true
    setUploading(true)
    setProgress(0)

    const formData = new FormData()
    formData.append('file', file)

    try {
      const res = await apiClient.post('/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (e) => {
          if (e.total) setProgress(Math.round((e.loaded / e.total) * 100))
        },
      })
      setUploadResult(res.data)
      if (res.data.sheet_names?.length === 1) {
        setSelectedSheet(res.data.sheet_names[0])
      }
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: { message?: string } } } }
      setError(axiosErr.response?.data?.error?.message || t('error.upload_failed'))
    } finally {
      setUploading(false)
    }
  }

  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    const file = e.dataTransfer.files[0]
    if (file) handleFile(file)
  }

  const onFileSelect = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) handleFile(file)
  }

  const handleStartAssessment = async () => {
    if (!uploadResult) return
    setAssessing(true)
    setError('')
    try {
      const sheet = selectedSheet || uploadResult.sheet_names[0]
      const res = await apiClient.post('/assess', {
        upload_id: uploadResult.id,
        sheet_name: sheet,
      })
      // Reset stepper progress — new assessment = new round
      resetProgress(2)
      navigate('/assessment?id=' + res.data.id)
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: { message?: string } } } }
      setError(axiosErr.response?.data?.error?.message || t('error.cannot_start_assessment'))
    } finally {
      setAssessing(false)
    }
  }

  const quotaExhausted = quota !== null && quota.remaining === 0

  return (
    <div style={{
      background: 'var(--paper)', border: '1px solid var(--line)',
      borderRadius: 14, overflow: 'hidden',
    }}>
      {/* Stage header */}
      <div style={{
        padding: '20px 28px', borderBottom: '1px solid var(--line-soft)',
        display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 20,
      }}>
        <div>
          <div style={{
            fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--accent)',
            letterSpacing: '0.08em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 5,
          }}>STEP 1</div>
          <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>{t('page.upload.title')}</h2>
          <p style={{ color: 'var(--ink-soft)', fontSize: 14, marginTop: 5 }}>
            {t('page.upload.desc')}
          </p>
        </div>
        {quota !== null && (
          <div style={{
            fontSize: 12, color: 'var(--ink-faint)', fontFamily: 'var(--mono)',
            background: 'var(--panel)', padding: '6px 12px', borderRadius: 8,
            whiteSpace: 'nowrap',
          }}>
            {t('admin.remaining_quota')}: {quota.remaining}
          </div>
        )}
      </div>

      {/* Stage body */}
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

        {!uploadResult ? (
          <>
            {/* Drop zone */}
            <div
              onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
              onDragLeave={() => setDragOver(false)}
              onDrop={onDrop}
              onClick={() => fileInputRef.current?.click()}
              style={{
                border: `2px dashed ${dragOver ? 'var(--accent)' : 'var(--line)'}`,
                borderRadius: 12, padding: 40, textAlign: 'center',
                background: dragOver ? 'var(--accent-soft)' : 'var(--panel)',
                cursor: 'pointer', transition: 'all 0.2s',
              }}
            >
              <div style={{ fontSize: 34, color: 'var(--ink-faint)', marginBottom: 10 }}>📄</div>
              <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 4 }}>
                {t('upload.drop_title')}
              </h3>
              <p style={{ fontSize: 13, color: 'var(--ink-faint)' }}>
                {t('upload.drop_desc')}
              </p>
            </div>
            <input
              ref={fileInputRef}
              type="file"
              accept=".xlsx,.csv"
              onChange={onFileSelect}
              style={{ display: 'none' }}
            />

            {/* Demo sample button */}
            {!uploadResult && !uploading && (
              <div style={{ marginTop: 14, textAlign: 'center' }}>
                <button
                  type="button"
                  onClick={async () => {
                    try {
                      const res = await fetch('/data-readiness-tabular/demo-sample.xlsx')
                      const blob = await res.blob()
                      const file = new File([blob], 'demo-sample.xlsx', { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
                      handleFile(file)
                    } catch {
                      alert('Failed to load demo data')
                    }
                  }}
                  style={{
                    background: 'var(--accent-soft)', border: '1.5px solid var(--accent)', borderRadius: 8,
                    padding: '8px 16px', fontSize: 13, fontWeight: 550, color: 'var(--accent)',
                    cursor: 'pointer', transition: 'all 0.15s',
                  }}
                >
                  📋 {t('upload.use_demo')}
                </button>
              </div>
            )}

            {/* Upload progress */}
            {uploading && (
              <div style={{ marginTop: 18 }}>
                <div style={{
                  height: 8, borderRadius: 5, background: 'var(--line-soft)', overflow: 'hidden',
                }}>
                  <div style={{
                    height: '100%', background: 'var(--accent)',
                    width: `${progress}%`, transition: 'width 0.3s ease',
                    animation: progress >= 100 ? 'pulse 1.5s infinite' : 'none',
                  }} />
                </div>
                <p style={{ fontSize: 12, color: 'var(--ink-faint)', marginTop: 6, fontFamily: 'var(--mono)' }}>
                  {progress >= 100 ? 'Processing file...' : `${t('common.upload_progress')} ${progress}%`}
                </p>
              </div>
            )}
          </>
        ) : (
          <>
            {/* File info chip */}
            <div
              onClick={() => fileInputRef.current?.click()}
              style={{
                display: 'flex', alignItems: 'center', gap: 12,
                border: '1px solid var(--line)', borderRadius: 'var(--radius)',
                padding: '13px 16px', background: 'var(--paper)',
                cursor: 'pointer',
              }}
            >
              <div style={{
                width: 34, height: 34, borderRadius: 7,
                background: 'var(--green-soft)', color: 'var(--green)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontWeight: 700, fontSize: 11, fontFamily: 'var(--mono)',
                flexShrink: 0, overflow: 'hidden',
              }}>
                {(() => {
                  const parts = uploadResult.filename.split('.')
                  const ext = parts.length > 1 ? parts.pop()?.toUpperCase() : 'XLSX'
                  return ext
                })()}
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 14, fontWeight: 600 }}>{uploadResult.filename}</div>
                <div style={{ fontSize: 12, color: 'var(--ink-faint)', fontFamily: 'var(--mono)' }}>
                  {t('upload.file_info', { rows: uploadResult.row_count, cols: uploadResult.col_count })}
                </div>
              </div>
              <span className="pill ready">✓ {t('status.upload_complete')}</span>
              <a
                href={`/data-readiness-tabular/api/upload/${uploadResult.id}/download`}
                download={uploadResult.filename}
                className="pill"
                style={{ fontSize: 11, textDecoration: 'none', marginLeft: 8 }}
                onClick={(e) => e.stopPropagation()}
              >
                ↓ Download
              </a>
              <button
                type="button"
                onClick={(e) => { e.stopPropagation(); setUploadResult(null); setSelectedSheet(''); setIsHistoryView(false); setError(''); }}
                style={{
                  background: 'none', border: 'none', cursor: 'pointer',
                  fontSize: 16, color: 'var(--ink-faint)', marginLeft: 8,
                  padding: '2px 6px', borderRadius: 4,
                }}
                title="Clear"
              >
                ✕
              </button>
            </div>
            <input
              ref={fileInputRef}
              type="file"
              accept=".xlsx,.csv"
              onChange={onFileSelect}
              style={{ display: 'none' }}
            />

            {/* Sheet selection */}
            {uploadResult.sheet_names.length > 1 && (
              <div style={{ marginTop: 18 }}>
                <p style={{ fontSize: 13, fontWeight: 550, marginBottom: 8 }}>{t('common.select_sheet')}</p>
                <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
                  {uploadResult.sheet_names.map((sheet) => (
                    <button
                      key={sheet}
                      onClick={() => setSelectedSheet(sheet)}
                      style={{
                        fontFamily: 'var(--mono)', fontSize: 12,
                        border: `1px solid ${selectedSheet === sheet ? 'var(--accent)' : 'var(--line)'}`,
                        borderRadius: 20, padding: '6px 13px',
                        color: selectedSheet === sheet ? 'var(--accent)' : 'var(--ink-soft)',
                        background: selectedSheet === sheet ? 'var(--accent-soft)' : 'var(--paper)',
                        transition: 'all 0.15s',
                      }}
                    >
                      {sheet}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {/* Stage footer */}
      {uploadResult && (
        <div style={{
          padding: '16px 28px', borderTop: '1px solid var(--line-soft)',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          background: 'var(--panel)',
        }}>
          {assessing ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, width: '100%' }}>
              <div style={{
                width: 18, height: 18, border: '2.5px solid var(--line-soft)',
                borderTopColor: 'var(--accent)', borderRadius: '50%',
                animation: 'spin 0.8s linear infinite',
              }} />
              <span style={{ fontSize: 14, color: 'var(--ink-soft)', fontWeight: 500 }}>
                {t('common.assessing')}
              </span>
              <style>{`@keyframes spin { to { transform: rotate(360deg) } }`}</style>
            </div>
          ) : (
            <>
              <span style={{ fontSize: 12.5, color: 'var(--ink-faint)' }}>
                {isHistoryView ? t('status.upload_complete') : t('common.ready_for_assess')}
              </span>
              <div style={{ position: 'relative' }}>
                {isHistoryView ? (
                  <button
                    className="btn btn-primary"
                    onClick={() => navigate('/assessment')}
                  >
                    View Assessment →
                  </button>
                ) : (
                  <button
                    className="btn btn-primary"
                    onClick={handleStartAssessment}
                    disabled={quotaExhausted || (!selectedSheet && uploadResult.sheet_names.length > 1)}
                    title={quotaExhausted ? t('tooltip.quota_exhausted') : undefined}
                  >
                    {t('btn.start_assess')} →
                  </button>
                )}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  )
}
