import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import apiClient from '../api/client'

interface CleaningResult {
  session_id: string
  rows_before: number
  rows_after: number
  score_before: number
  score_after: number
}

interface DetectedIssue {
  title: string
  description: string
  indicator: string
  severity?: string
  affected_rows?: number
  unit?: string
  examples?: IssueExample[]
}

interface IssueExample {
  label?: string
  headers: string[]
  row_number: number
  cells: string[]
  highlights: number[]
}

interface Rule {
  id: string
  label: string
  desc: string
  enabled: boolean
}

function getAvailableRules(issues: DetectedIssue[]): Rule[] {
  const indicators = new Set(issues.map(i => i.indicator))
  const descriptions = issues.map(i => i.description).join(' ')
  
  const rules: Rule[] = [
    {
      id: 'date_normalize', label: '統一日期格式',
      desc: '將各種日期寫法統一為 yyyy-MM-dd',
      enabled: true,
    },
    {
      id: 'dedup', label: '移除重複列',
      desc: '刪除完全相同的資料列',
      enabled: indicators.has('duplicate_similar'),
    },
    {
      id: 'name_normalize', label: '客戶名正規化',
      desc: '統一公司名稱的不同寫法為最常用版本',
      enabled: indicators.has('name_variants'),
    },
    {
      id: 'subtotal_remove', label: '移除小計列',
      desc: '刪除含「小計」「合計」的非資料列',
      enabled: descriptions.includes('小計'),
    },
    {
      id: 'newline_remove', label: '移除儲存格內換行',
      desc: '將儲存格內的換行符號替換為空格',
      enabled: descriptions.includes('換行'),
    },
    {
      id: 'bracket_note_remove', label: '移除中文括號備註',
      desc: '刪除儲存格內的中文括號備註內容',
      enabled: descriptions.includes('備註'),
    },
    {
      id: 'empty_row_remove', label: '移除全空列',
      desc: '刪除所有欄位都為空的資料列',
      enabled: descriptions.includes('多表格') || indicators.has('completeness'),
    },
    {
      id: 'multi_table_keep_main', label: '移除多餘資料區塊',
      desc: '保留最大的連續資料區塊，移除其他被空白列隔開的段落',
      enabled: descriptions.includes('多表格'),
    },
    {
      id: 'empty_col_remove', label: '移除高度空缺欄位',
      desc: '移除超過 80% 為空值的欄位，提升資料完整度',
      enabled: indicators.has('completeness') || descriptions.includes('缺漏'),
    },
  ]

  return rules.filter(r => r.enabled)
}

export default function CleaningPage() {
  const [availableRules, setAvailableRules] = useState<Rule[]>([])
  const [selectedRules, setSelectedRules] = useState<string[]>([])
  const [assessmentId, setAssessmentId] = useState<string>('')
  const [cleaning, setCleaning] = useState(false)
  const [result, setResult] = useState<CleaningResult | null>(null)
  const [error, setError] = useState('')
  const [showConfirmPanel, setShowConfirmPanel] = useState(false)
  const [previewData, setPreviewData] = useState<any>(null)
  const [selectedColRemovals, setSelectedColRemovals] = useState<number[]>([])
  const [selectedBlockKeep, setSelectedBlockKeep] = useState<number>(-1)
  const navigate = useNavigate()

  useEffect(() => {
    loadAssessment()
  }, [])

  const loadAssessment = async () => {
    try {
      const res = await apiClient.get('/assess/latest')
      setAssessmentId(res.data.id)
      const issues: DetectedIssue[] = res.data.issues || []
      const rules = getAvailableRules(issues)
      setAvailableRules(rules)
      setSelectedRules(rules.map(r => r.id))
    } catch {
      const allRules = getAvailableRules([])
      setAvailableRules(allRules)
    }
  }

  const toggleRule = (id: string) => {
    setSelectedRules((prev) =>
      prev.includes(id) ? prev.filter((r) => r !== id) : [...prev, id]
    )
  }

  const handleClean = async () => {
    if (!assessmentId) return
    setError('')

    // If user selected rules that need confirmation, show preview first
    const needsConfirm = selectedRules.includes('empty_col_remove') || selectedRules.includes('multi_table_keep_main')

    if (needsConfirm && !showConfirmPanel) {
      // Fetch preview
      try {
        const res = await apiClient.post('/clean/preview-removals', { assessment_id: assessmentId })
        setPreviewData(res.data)
        // Auto-select all empty columns for removal
        if (res.data.empty_columns?.length > 0) {
          setSelectedColRemovals(res.data.empty_columns.map((c: any) => c.col_index))
        }
        // Auto-select the main (largest) block to keep
        if (res.data.data_blocks?.length > 1) {
          const mainIdx = res.data.data_blocks.findIndex((b: any) => b.is_main)
          setSelectedBlockKeep(mainIdx >= 0 ? mainIdx : 0)
        }
        setShowConfirmPanel(true)
      } catch (err: unknown) {
        const axiosErr = err as { response?: { data?: { error?: { message?: string } } } }
        setError(axiosErr.response?.data?.error?.message || '無法取得預覽資料')
      }
      return
    }

    setCleaning(true)
    try {
      const res = await apiClient.post('/clean/apply', {
        assessment_id: assessmentId,
        rules: selectedRules,
        remove_columns: showConfirmPanel ? selectedColRemovals : undefined,
        keep_block_index: showConfirmPanel ? selectedBlockKeep : -1,
      })
      setResult(res.data)
      setShowConfirmPanel(false)
      setPreviewData(null)
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: { message?: string } } } }
      setError(axiosErr.response?.data?.error?.message || '梳理執行失敗')
    } finally {
      setCleaning(false)
    }
  }

  return (
    <div style={{
      background: 'var(--paper)', border: '1px solid var(--line)',
      borderRadius: 14, overflow: 'hidden',
    }}>
      {/* Header */}
      <div style={{
        padding: '20px 28px', borderBottom: '1px solid var(--line-soft)',
      }}>
        <div style={{
          fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--accent)',
          letterSpacing: '0.08em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 5,
        }}>STEP 5</div>
        <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>資料梳理</h2>
        <p style={{ color: 'var(--ink-soft)', fontSize: 14, marginTop: 5 }}>
          選擇梳理規則並執行批次資料清理
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

        {/* Rules checkboxes */}
        <div style={{ marginBottom: 24 }}>
          <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 12 }}>批次規則</h3>
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            {availableRules.map((rule) => {
              const isSelected = selectedRules.includes(rule.id)
              return (
                <button
                  key={rule.id}
                  onClick={() => toggleRule(rule.id)}
                  title={rule.desc}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 8,
                    padding: '8px 14px', borderRadius: 20,
                    border: `1.5px solid ${isSelected ? 'var(--accent)' : 'var(--line)'}`,
                    background: isSelected ? 'var(--accent-soft)' : 'var(--paper)',
                    color: isSelected ? 'var(--accent)' : 'var(--ink-soft)',
                    fontSize: 13, fontWeight: 550, transition: 'all 0.15s',
                  }}
                >
                  <span style={{
                    width: 16, height: 16, borderRadius: 4,
                    border: `1.5px solid ${isSelected ? 'var(--accent)' : 'var(--line)'}`,
                    background: isSelected ? 'var(--accent)' : 'transparent',
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    color: '#fff', fontSize: 10,
                  }}>
                    {isSelected && '✓'}
                  </span>
                  {rule.label}
                </button>
              )
            })}
          </div>
        </div>

        {/* Confirmation Panel */}
        {showConfirmPanel && previewData && (
          <div style={{ marginBottom: 24, border: '1px solid var(--line)', borderRadius: 10, padding: 18 }}>
            <h4 style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>確認要移除的項目</h4>

            {/* Empty columns - Excel-style data grid */}
            {previewData.empty_columns?.length > 0 && selectedRules.includes('empty_col_remove') && (
              <div style={{ marginBottom: 20 }}>
                <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 8 }}>
                  高度空缺欄位（紅色欄位建議移除，點擊表頭切換）
                </div>
                <div style={{ overflowX: 'auto', border: '1px solid var(--line-soft)', borderRadius: 8, maxHeight: 400, overflowY: 'auto' }}>
                  <table style={{ borderCollapse: 'collapse', fontSize: 11, fontFamily: 'var(--mono)', minWidth: 'max-content' }}>
                    <thead>
                      <tr>
                        <th style={{ padding: '8px 6px', border: '1px solid #e5e7eb', background: '#f3f4f6', color: 'var(--ink-faint)', fontWeight: 600, minWidth: 36, textAlign: 'center', position: 'sticky', left: 0, zIndex: 2 }}>#</th>
                        {(previewData.all_columns || []).map((col: any) => {
                          const isCandidate = previewData.empty_columns.some((ec: any) => ec.col_index === col.col_index)
                          const isSelected = selectedColRemovals.includes(col.col_index)
                          return (
                            <th
                              key={col.col_index}
                              onClick={() => {
                                if (!isCandidate) return
                                if (isSelected) {
                                  setSelectedColRemovals(prev => prev.filter(i => i !== col.col_index))
                                } else {
                                  setSelectedColRemovals(prev => [...prev, col.col_index])
                                }
                              }}
                              style={{
                                padding: '8px 6px',
                                border: '1px solid #e5e7eb',
                                background: isCandidate
                                  ? (isSelected ? 'rgba(220, 38, 38, 0.1)' : 'rgba(220, 38, 38, 0.03)')
                                  : '#f3f4f6',
                                color: isCandidate ? 'var(--rose)' : 'var(--ink-soft)',
                                fontWeight: 600,
                                whiteSpace: 'nowrap',
                                cursor: isCandidate ? 'pointer' : 'default',
                                minWidth: 80,
                                textAlign: 'center',
                                position: 'relative',
                              }}
                            >
                              {isCandidate && (
                                <div style={{ fontSize: 11, marginBottom: 3, fontWeight: 700 }}>
                                  {isSelected ? '☑ 移除' : '☐ 保留'}
                                </div>
                              )}
                              <div style={{ fontSize: 11, lineHeight: 1.3 }}>{col.col_name}</div>
                              {isCandidate && (
                                <div style={{
                                  marginTop: 4,
                                  padding: '2px 6px',
                                  borderRadius: 10,
                                  background: 'rgba(220, 38, 38, 0.15)',
                                  color: '#dc2626',
                                  fontSize: 11,
                                  fontWeight: 700,
                                }}>
                                  {Math.round(col.empty_rate * 100)}% 空值
                                </div>
                              )}
                            </th>
                          )
                        })}
                      </tr>
                    </thead>
                    <tbody>
                      {(previewData.sample_rows || []).map((row: any) => (
                        <tr key={row.row_number}>
                          <td style={{ padding: '4px 6px', border: '1px solid #e5e7eb', background: '#f9fafb', color: 'var(--ink-faint)', fontWeight: 500, textAlign: 'center', fontSize: 10, position: 'sticky', left: 0, zIndex: 1 }}>{row.row_number}</td>
                          {row.cells.map((cell: string, k: number) => {
                            const isCandidate = previewData.empty_columns.some((ec: any) => ec.col_index === k)
                            const isSelected = selectedColRemovals.includes(k)
                            return (
                              <td key={k} style={{
                                padding: '4px 8px',
                                border: '1px solid #f0f0f0',
                                background: isCandidate && isSelected ? 'rgba(220, 38, 38, 0.04)' : 'white',
                                whiteSpace: 'nowrap',
                                maxWidth: 160,
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                color: cell ? 'var(--ink)' : '#ccc',
                              }}>
                                {cell || '—'}
                              </td>
                            )
                          })}
                        </tr>
                      ))}
                    </tbody>
                    {/* Footer row showing empty rate */}
                    <tfoot>
                      <tr>
                        <td style={{ padding: '6px 6px', border: '1px solid #e5e7eb', background: '#f9fafb', fontWeight: 600, textAlign: 'center', fontSize: 9, color: 'var(--ink-faint)', position: 'sticky', left: 0, zIndex: 1 }}>空率</td>
                        {(previewData.all_columns || []).map((col: any) => {
                          const isCandidate = previewData.empty_columns.some((ec: any) => ec.col_index === col.col_index)
                          const isSelected = selectedColRemovals.includes(col.col_index)
                          const pct = Math.round(col.empty_rate * 100)
                          return (
                            <td key={col.col_index} style={{
                              padding: '6px 6px',
                              border: '1px solid #e5e7eb',
                              background: isCandidate && isSelected ? 'rgba(220, 38, 38, 0.08)' : '#f9fafb',
                              textAlign: 'center',
                              fontSize: 10,
                              fontWeight: 600,
                              color: isCandidate ? 'var(--rose)' : 'var(--ink-faint)',
                            }}>
                              {pct}% 空
                            </td>
                          )
                        })}
                      </tr>
                    </tfoot>
                  </table>
                </div>
              </div>
            )}

            {/* Data blocks */}
            {previewData.data_blocks?.length > 1 && selectedRules.includes('multi_table_keep_main') && (
              <div>
                <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 8 }}>
                  資料區塊（選擇要保留的區塊，其他將被移除）
                </div>
                {previewData.data_blocks.map((block: any, idx: number) => {
                  const isSelected = selectedBlockKeep === idx
                  return (
                    <div
                      key={idx}
                      onClick={() => setSelectedBlockKeep(idx)}
                      style={{
                        marginBottom: 14,
                        border: isSelected ? '2.5px solid var(--accent)' : '1.5px solid var(--line)',
                        borderRadius: 10,
                        overflow: 'hidden',
                        cursor: 'pointer',
                        transition: 'border-color 0.15s, box-shadow 0.15s',
                        boxShadow: isSelected ? '0 2px 8px rgba(59,130,246,0.1)' : 'none',
                      }}
                    >
                      {/* Description header */}
                      <div style={{
                        padding: '12px 16px',
                        background: isSelected ? 'var(--accent-soft)' : 'var(--panel)',
                        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                      }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                          <span style={{ fontWeight: 650, fontSize: 14 }}>第 {block.start_row}–{block.end_row} 列</span>
                          <span style={{ color: 'var(--ink-faint)', fontSize: 12, fontFamily: 'var(--mono)' }}>({block.row_count} 列)</span>
                          {block.is_main && (
                            <span style={{ color: 'var(--green)', fontSize: 11, fontWeight: 600, background: 'rgba(21,128,61,0.08)', padding: '2px 8px', borderRadius: 10 }}>推薦保留</span>
                          )}
                        </div>
                        <div style={{
                          width: 20, height: 20, borderRadius: '50%',
                          border: isSelected ? '6px solid var(--accent)' : '2px solid var(--line)',
                          background: 'white', flexShrink: 0,
                        }} />
                      </div>
                      {/* Sample rows */}
                      {block.sample_rows?.length > 0 && (
                        <div style={{ overflowX: 'auto', borderTop: '1px solid var(--line-soft)' }}>
                          <table style={{ borderCollapse: 'collapse', fontSize: 11, fontFamily: 'var(--mono)', width: '100%', minWidth: 'max-content' }}>
                            <thead>
                              <tr>
                                <th style={{ padding: '4px 8px', background: '#f3f4f6', border: '1px solid #e5e7eb', fontWeight: 600, color: 'var(--ink-faint)', textAlign: 'center', fontSize: 10 }}>#</th>
                                {previewData.headers?.map((h: string, hIdx: number) => (
                                  <th key={hIdx} style={{ padding: '4px 8px', background: '#f3f4f6', border: '1px solid #e5e7eb', fontWeight: 600, color: 'var(--ink-soft)', whiteSpace: 'nowrap', fontSize: 10 }}>{h}</th>
                                ))}
                              </tr>
                            </thead>
                            <tbody>
                              {block.sample_rows.map((row: any) => (
                                <tr key={row.row_number}>
                                  <td style={{ padding: '4px 8px', background: '#f9fafb', border: '1px solid #eee', color: 'var(--ink-faint)', fontWeight: 600, textAlign: 'center', fontSize: 10 }}>{row.row_number}</td>
                                  {row.cells.map((cell: string, k: number) => (
                                    <td key={k} style={{ padding: '4px 8px', border: '1px solid #eee', whiteSpace: 'nowrap', color: cell ? 'var(--ink-soft)' : '#ccc' }}>{cell || '—'}</td>
                                  ))}
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        )}

        {/* Progress */}
        {cleaning && (
          <div style={{ marginBottom: 20 }}>
            <div style={{
              height: 8, borderRadius: 5, background: 'var(--line-soft)', overflow: 'hidden',
            }}>
              <div style={{
                height: '100%', background: 'var(--accent)',
                width: '60%', transition: 'width 0.9s ease',
                animation: 'pulse 1.5s infinite',
              }} />
            </div>
            <p style={{ fontSize: 12, color: 'var(--ink-faint)', marginTop: 6, fontFamily: 'var(--mono)' }}>
              梳理執行中...
            </p>
          </div>
        )}

        {/* Results */}
        {result && (
          <div style={{
            background: 'var(--green-soft)', border: '1px solid #cfe8d8',
            borderRadius: 'var(--radius)', padding: '18px 20px', marginBottom: 20,
          }}>
            <h4 style={{ fontSize: 14, fontWeight: 600, color: 'var(--green)', marginBottom: 12 }}>
              ✓ 梳理完成
            </h4>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
              <div style={{
                background: 'var(--paper)', borderRadius: 'var(--radius-sm)', padding: '12px 14px',
              }}>
                <div style={{ fontSize: 12, color: 'var(--ink-faint)', fontFamily: 'var(--mono)', marginBottom: 4 }}>
                  資料列數
                </div>
                <div style={{ fontSize: 22, fontWeight: 700 }}>
                  {result.rows_before} → {result.rows_after}
                  <span style={{ fontSize: 12, color: 'var(--ink-faint)', marginLeft: 8 }}>
                    (-{result.rows_before - result.rows_after})
                  </span>
                </div>
              </div>
              <div style={{
                background: 'var(--paper)', borderRadius: 'var(--radius-sm)', padding: '12px 14px',
              }}>
                <div style={{ fontSize: 12, color: 'var(--ink-faint)', fontFamily: 'var(--mono)', marginBottom: 4 }}>
                  品質分數
                </div>
                <div style={{ fontSize: 22, fontWeight: 700 }}>
                  {result.score_before.toFixed(1)} → {result.score_after.toFixed(1)}
                  <span style={{
                    fontSize: 12, marginLeft: 8,
                    color: result.score_after > result.score_before ? 'var(--green)' : 'var(--ink-faint)',
                  }}>
                    (+{(result.score_after - result.score_before).toFixed(1)})
                  </span>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Footer */}
      <div style={{
        padding: '16px 28px', borderTop: '1px solid var(--line-soft)',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        background: 'var(--panel)',
      }}>
        <span style={{ fontSize: 12.5, color: 'var(--ink-faint)' }}>
          {result ? '梳理完成，可進入下一步產出檔案' : '選擇規則後點擊執行'}
        </span>
        {!result ? (
          <button
            className="btn btn-primary"
            onClick={handleClean}
            disabled={cleaning || selectedRules.length === 0}
          >
            {cleaning ? '執行中...' : '執行梳理'}
          </button>
        ) : (
          <button className="btn btn-primary" onClick={() => navigate('/export')}>
            下一步 →
          </button>
        )}
      </div>
    </div>
  )
}
