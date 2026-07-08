import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
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

function getAvailableRules(issues: DetectedIssue[], t: (key: string) => string): Rule[] {
  const indicators = new Set(issues.map(i => i.indicator))
  const descriptions = issues.map(i => i.description).join(' ')
  
  const rules: Rule[] = [
    {
      id: 'date_normalize', label: t('rule.date_normalize'),
      desc: t('rule.date_normalize.desc'),
      enabled: true,
    },
    {
      id: 'dedup', label: t('rule.dedup'),
      desc: t('rule.dedup.desc'),
      enabled: indicators.has('duplicate_similar'),
    },
    {
      id: 'name_normalize', label: t('rule.name_normalize'),
      desc: t('rule.name_normalize.desc'),
      enabled: indicators.has('name_variants'),
    },
    {
      id: 'subtotal_remove', label: t('rule.subtotal_remove'),
      desc: t('rule.subtotal_remove.desc'),
      enabled: descriptions.includes('小計'),
    },
    {
      id: 'newline_remove', label: t('rule.newline_remove'),
      desc: t('rule.newline_remove.desc'),
      enabled: descriptions.includes('換行'),
    },
    {
      id: 'bracket_note_remove', label: t('rule.bracket_note_remove'),
      desc: t('rule.bracket_note_remove.desc'),
      enabled: descriptions.includes('備註'),
    },
    {
      id: 'empty_row_remove', label: t('rule.empty_row_remove'),
      desc: t('rule.empty_row_remove.desc'),
      enabled: descriptions.includes('多表格') || indicators.has('completeness'),
    },
    {
      id: 'multi_table_keep_main', label: t('rule.multi_table_keep_main'),
      desc: t('rule.multi_table_keep_main.desc'),
      enabled: descriptions.includes('多表格'),
    },
    {
      id: 'empty_col_remove', label: t('rule.empty_col_remove'),
      desc: t('rule.empty_col_remove.desc'),
      enabled: indicators.has('completeness') || descriptions.includes('缺漏'),
    },
  ]

  return rules.filter(r => r.enabled)
}

export default function CleaningPage() {
  const { t } = useTranslation()
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
    // Check if cleaning was already done for the CURRENT assessment
    apiClient.get('/clean/latest').then((res) => {
      if (res.data && (res.data.session_id || res.data.id)) {
        // Only show result if it matches current assessment
        apiClient.get('/assess/latest').then((assessRes) => {
          if (assessRes.data && res.data.assessment_id === assessRes.data.id) {
            setResult({
              session_id: res.data.session_id || res.data.id,
              rows_before: res.data.rows_before || 0,
              rows_after: res.data.rows_after || 0,
              score_before: res.data.score_before || 0,
              score_after: res.data.score_after || 0,
            })
          }
        }).catch(() => {})
      }
    }).catch(() => {})
  }, [])

  const loadAssessment = async () => {
    try {
      const res = await apiClient.get('/assess/latest')
      setAssessmentId(res.data.id)
      const issues: DetectedIssue[] = res.data.issues || []
      const rules = getAvailableRules(issues, t)
      setAvailableRules(rules)
      setSelectedRules(rules.map(r => r.id))
    } catch {
      const allRules = getAvailableRules([], t)
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
      setCleaning(true)
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
        setError(axiosErr.response?.data?.error?.message || t('error.preview_failed'))
      } finally {
        setCleaning(false)
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
      setError(axiosErr.response?.data?.error?.message || t('error.cleaning_failed'))
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
        <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>{t('page.cleaning.title')}</h2>
        <p style={{ color: 'var(--ink-soft)', fontSize: 14, marginTop: 5 }}>
          {t('page.cleaning.desc')}
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
          <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 12 }}>{t('assessment.batch_rules')}</h3>
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
            <h4 style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>{t('assessment.confirm_removal')}</h4>

            {/* Empty columns - Excel-style data grid */}
            {previewData.empty_columns?.length > 0 && selectedRules.includes('empty_col_remove') && (
              <div style={{ marginBottom: 20 }}>
                <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 8 }}>
                  {t('assessment.empty_col_hint')}
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
                                  {isSelected ? `☑ ${t('clean.remove_label')}` : `☐ ${t('clean.keep_label')}`}
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
                                  {Math.round(col.empty_rate * 100)}% {t('clean.empty_rate')}
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
                        <td style={{ padding: '6px 6px', border: '1px solid #e5e7eb', background: '#f9fafb', fontWeight: 600, textAlign: 'center', fontSize: 9, color: 'var(--ink-faint)', position: 'sticky', left: 0, zIndex: 1 }}>{t('clean.empty_rate')}</td>
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
                              {pct}% {t('clean.empty_rate')}
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
                  {t('assessment.data_blocks_hint')}
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
                          <span style={{ fontWeight: 650, fontSize: 14 }}>{t('clean.block_range', { start: block.start_row, end: block.end_row })}</span>
                          <span style={{ color: 'var(--ink-faint)', fontSize: 12, fontFamily: 'var(--mono)' }}>({block.row_count} {t('common.rows')})</span>
                          {block.is_main && (
                            <span style={{ color: 'var(--green)', fontSize: 11, fontWeight: 600, background: 'rgba(21,128,61,0.08)', padding: '2px 8px', borderRadius: 10 }}>{t('assessment.recommend_keep')}</span>
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
              {t('common.cleaning_progress')}
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
              ✓ {t('clean.complete')}
            </h4>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
              <div style={{
                background: 'var(--paper)', borderRadius: 'var(--radius-sm)', padding: '12px 14px',
              }}>
                <div style={{ fontSize: 12, color: 'var(--ink-faint)', fontFamily: 'var(--mono)', marginBottom: 4 }}>
                  {t('clean.row_count')}
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
                  {t('clean.quality_score')}
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
          {result ? t('common.after_clean_hint') : t('common.select_rules_hint')}
        </span>
        {!result ? (
          <button
            className="btn btn-primary"
            onClick={handleClean}
            disabled={cleaning || selectedRules.length === 0}
          >
            {cleaning ? t('common.running') : t('btn.run_clean')}
          </button>
        ) : (
          <button className="btn btn-primary" onClick={() => navigate('/export')}>
            {t('btn.next_step')} →
          </button>
        )}
      </div>
    </div>
  )
}
