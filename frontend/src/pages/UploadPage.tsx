import { useState, useRef, type DragEvent, type ChangeEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import apiClient from '../api/client'

interface UploadResult {
  id: string
  filename: string
  row_count: number
  col_count: number
  sheet_names: string[]
}

export default function UploadPage() {
  const [dragOver, setDragOver] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [uploadResult, setUploadResult] = useState<UploadResult | null>(null)
  const [selectedSheet, setSelectedSheet] = useState('')
  const [error, setError] = useState('')
  const [assessing, setAssessing] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()

  const handleFile = async (file: File) => {
    const validTypes = [
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      'text/csv',
      'application/vnd.ms-excel',
    ]
    const ext = file.name.split('.').pop()?.toLowerCase()
    if (!validTypes.includes(file.type) && ext !== 'xlsx' && ext !== 'csv') {
      setError('僅支援 .xlsx 或 .csv 格式檔案')
      return
    }

    setError('')
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
      setError(axiosErr.response?.data?.error?.message || '上傳失敗，請稍後再試')
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
      navigate('/assessment?id=' + res.data.id)
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: { message?: string } } } }
      setError(axiosErr.response?.data?.error?.message || '無法啟動評估')
    } finally {
      setAssessing(false)
    }
  }

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
          <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>上傳檔案</h2>
          <p style={{ color: 'var(--ink-soft)', fontSize: 14, marginTop: 5 }}>
            拖曳或點擊選取 Excel / CSV 檔案，系統將解析後進行品質評估
          </p>
        </div>
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
                拖曳檔案至此處
              </h3>
              <p style={{ fontSize: 13, color: 'var(--ink-faint)' }}>
                或點擊選取檔案（支援 .xlsx、.csv，上限 50MB）
              </p>
            </div>
            <input
              ref={fileInputRef}
              type="file"
              accept=".xlsx,.csv"
              onChange={onFileSelect}
              style={{ display: 'none' }}
            />

            {/* Upload progress */}
            {uploading && (
              <div style={{ marginTop: 18 }}>
                <div style={{
                  height: 8, borderRadius: 5, background: 'var(--line-soft)', overflow: 'hidden',
                }}>
                  <div style={{
                    height: '100%', background: 'var(--accent)',
                    width: `${progress}%`, transition: 'width 0.3s ease',
                  }} />
                </div>
                <p style={{ fontSize: 12, color: 'var(--ink-faint)', marginTop: 6, fontFamily: 'var(--mono)' }}>
                  上傳中... {progress}%
                </p>
              </div>
            )}
          </>
        ) : (
          <>
            {/* File info chip */}
            <div style={{
              display: 'flex', alignItems: 'center', gap: 12,
              border: '1px solid var(--line)', borderRadius: 'var(--radius)',
              padding: '13px 16px', background: 'var(--paper)',
            }}>
              <div style={{
                width: 34, height: 34, borderRadius: 7,
                background: 'var(--green-soft)', color: 'var(--green)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontWeight: 700, fontSize: 11, fontFamily: 'var(--mono)',
              }}>
                {uploadResult.filename.split('.').pop()?.toUpperCase()}
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 14, fontWeight: 600 }}>{uploadResult.filename}</div>
                <div style={{ fontSize: 12, color: 'var(--ink-faint)', fontFamily: 'var(--mono)' }}>
                  {uploadResult.row_count} 列 × {uploadResult.col_count} 欄
                </div>
              </div>
              <span className="pill ready">✓ 上傳完成</span>
            </div>

            {/* Sheet selection */}
            {uploadResult.sheet_names.length > 1 && (
              <div style={{ marginTop: 18 }}>
                <p style={{ fontSize: 13, fontWeight: 550, marginBottom: 8 }}>選擇工作表：</p>
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
                評估中，正在分析資料品質...
              </span>
              <style>{`@keyframes spin { to { transform: rotate(360deg) } }`}</style>
            </div>
          ) : (
            <>
              <span style={{ fontSize: 12.5, color: 'var(--ink-faint)' }}>
                選定工作表後將進行 AI Readiness 品質評估
              </span>
              <button
                className="btn btn-primary"
                onClick={handleStartAssessment}
                disabled={!selectedSheet && uploadResult.sheet_names.length > 1}
              >
                開始評估 →
              </button>
            </>
          )}
        </div>
      )}
    </div>
  )
}
