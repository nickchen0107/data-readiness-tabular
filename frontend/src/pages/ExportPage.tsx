import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  RadarChart, Radar, PolarGrid, PolarAngleAxis, PolarRadiusAxis,
  ResponsiveContainer, Legend, PieChart, Pie, Cell,
} from 'recharts'
import apiClient from '../api/client'
import { getResolvedIssues, getRemainingIssues, IssueExample } from '../utils/issueDiff'

/* ─── Type Definitions ─── */

interface LogEntry {
  operation_type: string
  timestamp: string
  affected_rows: number[]
  details: string
}

interface AssessmentSummary {
  id: string
  total_score: number
  status: string
  row_completeness: number
  column_completeness: number
  format_consistency: number
  duplicate_similar: number
  table_structure: number
  ai_query_readiness: number
  issues: Array<{
    title: string
    severity: string
    description: string
    affected_rows: number
    unit: string
    indicator: string
    examples?: IssueExample[]
  }>
  row_distribution: { high: number; medium: number; low: number }
}

interface ComparisonData {
  session: {
    id: string
    rows_before: number
    rows_after: number
    score_before: number
    score_after: number
    rules_applied: string[]
    cleaning_log: LogEntry[]
  }
  original_assessment: AssessmentSummary
  post_clean_assessment: AssessmentSummary
}

/* ─── Helpers ─── */

function formatCount(count: number): string {
  if (count >= 10000) {
    return (count / 10000).toFixed(1).replace(/\.0$/, '') + '萬'
  }
  return count.toLocaleString()
}

/* ─── Excel Table Renderer ─── */

interface ExcelTableProps {
  examples: IssueExample[]
  highlightColor?: string       // border color for highlighted cells
  highlightBg?: string          // background for highlighted cells
  strikethrough?: boolean       // apply line-through to highlighted cells
}

function ExcelTable({ examples, highlightColor = 'var(--rose, #dc2626)', highlightBg = 'rgba(220, 38, 38, 0.06)', strikethrough = false }: ExcelTableProps) {
  // Group examples by label
  const groups: Array<{ label: string | undefined; items: IssueExample[] }> = []
  let currentLabel: string | undefined = undefined
  let currentItems: IssueExample[] = []
  for (const ex of examples) {
    if (ex.label !== currentLabel) {
      if (currentItems.length > 0) groups.push({ label: currentLabel, items: currentItems })
      currentLabel = ex.label
      currentItems = [ex]
    } else {
      currentItems.push(ex)
    }
  }
  if (currentItems.length > 0) groups.push({ label: currentLabel, items: currentItems })

  return (
    <>
      {groups.slice(0, 3).map((group, gIdx) => (
        <div key={gIdx}>
          {group.label && (
            <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--ink-soft)', margin: '12px 0 6px', paddingLeft: 4 }}>● {group.label}</div>
          )}
          <div style={{ marginTop: group.label ? 0 : 12, borderRadius: 8, overflow: 'auto', border: '1px solid var(--line-soft)' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12, fontFamily: 'var(--mono)', tableLayout: 'auto', minWidth: 'max-content' }}>
              <thead>
                <tr>
                  <th style={{ background: '#f3f4f6', padding: '6px 10px', border: '1px solid #e5e7eb', fontSize: 11, color: 'var(--ink-faint)', fontWeight: 600, textAlign: 'center', width: 40, minWidth: 40 }}>#</th>
                  {(() => {
                    const isFirstRowHeader = group.items[0].row_number === 1
                    const headerRow = isFirstRowHeader ? group.items[0] : null
                    const headerContent = isFirstRowHeader ? group.items[0].cells : group.items[0].headers
                    const hMergeSet = new Set<number>()
                    const hMergeMap = new Map<number, number>()
                    if (headerRow && headerRow.merges) {
                      for (const m of headerRow.merges) {
                        hMergeMap.set(m.start_col, m.span)
                        for (let s = m.start_col + 1; s < m.start_col + m.span; s++) {
                          hMergeSet.add(s)
                        }
                      }
                    }
                    return headerContent.map((h, hIdx) => {
                      if (hMergeSet.has(hIdx)) return null
                      const colspan = hMergeMap.get(hIdx) || 1
                      const isHighlighted = headerRow && headerRow.highlights && headerRow.highlights.includes(hIdx)
                      return (
                        <th key={hIdx} colSpan={colspan > 1 ? colspan : undefined} style={{
                          background: isHighlighted ? highlightBg : '#f3f4f6',
                          padding: '6px 10px',
                          border: isHighlighted ? `1.5px solid ${highlightColor}` : '1px solid #e5e7eb',
                          fontSize: 11,
                          color: isHighlighted ? highlightColor : 'var(--ink-soft)',
                          fontWeight: 600,
                          textAlign: 'left', whiteSpace: 'nowrap',
                        }}>{h}</th>
                      )
                    }).filter(Boolean)
                  })()}
                </tr>
              </thead>
              <tbody>
                {(() => {
                  const isFirstRowHeader = group.items[0].row_number === 1
                  const dataItems = isFirstRowHeader ? group.items.slice(1) : group.items
                  return dataItems.map((ex, j) => (
                    <tr key={j}>
                      <td style={{ background: '#f3f4f6', padding: '5px 8px', border: '1px solid #e5e7eb', color: 'var(--ink-faint)', fontWeight: 600, textAlign: 'center', fontSize: 11 }}>
                        {ex.row_number > 0 ? ex.row_number : ''}
                      </td>
                      {(() => {
                        const mergeSet = new Set<number>()
                        const mergeMap = new Map<number, number>()
                        if (ex.merges) {
                          for (const m of ex.merges) {
                            mergeMap.set(m.start_col, m.span)
                            for (let s = m.start_col + 1; s < m.start_col + m.span; s++) {
                              mergeSet.add(s)
                            }
                          }
                        }
                        return ex.cells.map((cell, k) => {
                          if (mergeSet.has(k)) return null
                          const colspan = mergeMap.get(k) || 1
                          const isHighlighted = ex.highlights && ex.highlights.includes(k)
                          const isEmpty = cell === '' && isHighlighted
                          const isMerged = colspan > 1
                          return (
                            <td key={k} colSpan={colspan} style={{
                              padding: '5px 10px',
                              border: isHighlighted ? `1.5px solid ${highlightColor}` : '1px solid #e5e7eb',
                              background: isMerged ? 'rgba(59, 130, 246, 0.06)' : isHighlighted ? highlightBg : 'var(--paper)',
                              color: isMerged ? 'var(--accent)' : isHighlighted ? highlightColor : 'var(--ink-soft)',
                              whiteSpace: 'pre-line',
                              fontWeight: isMerged ? 600 : isHighlighted ? 600 : 400,
                              textAlign: isMerged ? 'center' : 'left',
                              textDecoration: isHighlighted && strikethrough ? 'line-through' : 'none',
                            }}>
                              {isMerged ? `⬌ ${cell || '(合併儲存格)'}` : isEmpty ? '—' : cell}
                              {ex.format_labels?.[k] && (
                                <div style={{ marginTop: 2 }}>
                                  <span style={{
                                    fontSize: 10, fontFamily: 'var(--mono)',
                                    background: 'rgba(99, 102, 241, 0.1)', color: '#4338ca',
                                    borderRadius: 3, padding: '1px 5px', fontWeight: 500,
                                  }}>{ex.format_labels[k]}</span>
                                </div>
                              )}
                            </td>
                          )
                        }).filter(Boolean)
                      })()}
                    </tr>
                  ))
                })()}
              </tbody>
            </table>
          </div>
        </div>
      ))}
      {examples.length >= 5 && (
        <div style={{ marginTop: 8, fontSize: 11, color: 'var(--ink-faint)', textAlign: 'center', fontFamily: 'var(--mono)' }}>
          僅顯示前 5 筆，更多問題列請至梳理步驟查看
        </div>
      )}
    </>
  )
}

/* ─── Main Component ─── */

export default function ExportPage() {
  const navigate = useNavigate()
  const [data, setData] = useState<ComparisonData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [downloading, setDownloading] = useState('')
  const [expandedIssues, setExpandedIssues] = useState<Set<string>>(new Set())

  useEffect(() => { loadComparisonData() }, [])

  const loadComparisonData = async () => {
    try {
      const latestRes = await apiClient.get('/clean/latest')
      const sessionId = latestRes.data.session_id || latestRes.data.id
      if (!sessionId) { setError('找不到梳理記錄，請先執行資料梳理'); setLoading(false); return }
      const compareRes = await apiClient.get(`/compare/${sessionId}`)
      setData(compareRes.data)
    } catch (err: unknown) {
      const axiosErr = err as { response?: { status?: number; data?: { error?: { message?: string } } } }
      if (axiosErr.response?.status === 404) { setError('梳理記錄不存在，請先執行資料梳理') }
      else { setError(axiosErr.response?.data?.error?.message || '載入比較資料失敗') }
    } finally { setLoading(false) }
  }

  const toggleIssue = (key: string) => {
    setExpandedIssues(prev => {
      const next = new Set(prev)
      if (next.has(key)) { next.delete(key) } else { next.add(key) }
      return next
    })
  }

  const handleDownload = async (type: 'xlsx' | 'pdf' | 'log') => {
    if (!data) return
    setDownloading(type)
    try {
      const res = await apiClient.get(`/export/${data.session.id}/${type}`, { responseType: 'blob' })
      const blob = new Blob([res.data])
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = type === 'xlsx' ? 'refined.xlsx' : type === 'pdf' ? 'report.pdf' : 'cleaning.log'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(url)
    } catch { alert('下載失敗，請稍後再試') }
    finally { setDownloading('') }
  }

  if (loading) {
    return (
      <div style={{ background: 'var(--paper)', border: '1px solid var(--line)', borderRadius: 14, padding: 60, textAlign: 'center', color: 'var(--ink-faint)' }}>
        載入比較資料中...
      </div>
    )
  }

  if (error || !data) {
    return (
      <div style={{ background: 'var(--paper)', border: '1px solid var(--line)', borderRadius: 14, textAlign: 'center', padding: 60 }}>
        <p style={{ color: 'var(--rose)', marginBottom: 16 }}>{error || '載入失敗'}</p>
        <button className="btn btn-ghost" onClick={() => navigate('/cleaning')}>返回梳理步驟</button>
      </div>
    )
  }

  /* ─── Derived Data ─── */
  const { session, original_assessment, post_clean_assessment } = data
  const delta = session.score_after - session.score_before

  const statusLabel = post_clean_assessment.status === 'ready' ? 'AI Ready'
    : post_clean_assessment.status === 'conditional' ? 'Conditional' : 'Not Ready'
  const statusClass = post_clean_assessment.status === 'ready' ? 'ready'
    : post_clean_assessment.status === 'conditional' ? 'cond' : 'not'

  const origStatus = original_assessment.status
  const gaugeColor = origStatus === 'ready' ? '#15803d'
    : origStatus === 'conditional' ? '#b45309' : '#b42318'

  const postDist = post_clean_assessment.row_distribution
  const origDist = original_assessment.row_distribution
  const totalRows = (postDist?.high || 0) + (postDist?.medium || 0) + (postDist?.low || 0)
  const highDelta = (postDist?.high || 0) - (origDist?.high || 0)
  const medDelta = (postDist?.medium || 0) - (origDist?.medium || 0)
  const lowDelta = (postDist?.low || 0) - (origDist?.low || 0)

  const indicators = [
    { name: '列完整度', nameEn: 'Row Completeness', before: original_assessment.row_completeness, after: post_clean_assessment.row_completeness },
    { name: '欄完整度', nameEn: 'Column Completeness', before: original_assessment.column_completeness, after: post_clean_assessment.column_completeness },
    { name: '格式一致性', nameEn: 'Format Consistency', before: original_assessment.format_consistency, after: post_clean_assessment.format_consistency },
    { name: '資料唯一性', nameEn: 'Data Uniqueness', before: original_assessment.duplicate_similar, after: post_clean_assessment.duplicate_similar },
    { name: '表格結構', nameEn: 'Table Structure', before: original_assessment.table_structure, after: post_clean_assessment.table_structure },
    { name: 'AI 問答可用性', nameEn: 'AI Query Readiness', before: original_assessment.ai_query_readiness, after: post_clean_assessment.ai_query_readiness },
  ]

  const radarData = indicators.map(ind => ({ subject: ind.name, before: ind.before, after: ind.after }))

  const resolvedIssues = getResolvedIssues(original_assessment.issues, post_clean_assessment.issues)
  const remainingIssues = getRemainingIssues(post_clean_assessment.issues)

  // Find post-clean examples for resolved issues (by matching title in post_clean_assessment)
  const getPostCleanExamples = (title: string): IssueExample[] | undefined => {
    const postIssue = post_clean_assessment.issues.find(i => i.title === title)
    return postIssue?.examples
  }

  const downloads = [
    { type: 'xlsx' as const, icon: '📊', label: '梳理後資料', filename: 'refined.xlsx', desc: '清理完成的 Excel 檔案' },
    { type: 'pdf' as const, icon: '📋', label: '品質報告', filename: 'report.pdf', desc: '含圖表的品質評估報告' },
    { type: 'log' as const, icon: '📝', label: '梳理紀錄', filename: 'cleaning.log', desc: '所有操作的文字紀錄' },
  ]

  /* ─── Render (mirrors AssessmentPage layout) ─── */
  return (
    <div style={{ background: 'var(--paper)', border: '1px solid var(--line)', borderRadius: 14, overflow: 'hidden' }}>
      {/* Header — same as AssessmentPage */}
      <div style={{
        padding: '20px 28px', borderBottom: '1px solid var(--line-soft)',
        display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between',
      }}>
        <div>
          <div style={{
            fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--accent)',
            letterSpacing: '0.08em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 5,
          }}>STEP 5</div>
          <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>梳理成果總覽</h2>
        </div>
        <span className={`pill ${statusClass}`}>● {statusLabel}</span>
      </div>

      {/* Body */}
      <div style={{ padding: 28 }}>
        {/* ═══ Score + Row Distribution ═══ */}
        <div style={{ display: 'flex', gap: 26, alignItems: 'center', marginBottom: 28 }}>
          {/* Ring chart */}
          <div style={{ position: 'relative', width: 140, height: 140, flexShrink: 0 }}>
            <PieChart width={140} height={140}>
              <Pie
                data={[
                  { value: session.score_before, name: 'before' },
                  { value: delta > 0 ? delta : 0, name: 'improvement' },
                  { value: 100 - session.score_after, name: 'remaining' },
                ]}
                cx={65} cy={65} innerRadius={48} outerRadius={62}
                startAngle={90} endAngle={-270} dataKey="value" stroke="none"
              >
                <Cell fill={gaugeColor} />
                <Cell fill="rgba(34, 197, 94, 0.7)" />
                <Cell fill="var(--line-soft)" />
              </Pie>
            </PieChart>
            <div style={{ position: 'absolute', inset: 0, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
              <span style={{ fontSize: 34, fontWeight: 750, letterSpacing: '-0.03em', lineHeight: 1 }}>
                {session.score_after.toFixed(1)}
              </span>
              {delta > 0 && (
                <span style={{ fontSize: 11, color: 'var(--green)', fontFamily: 'var(--mono)', marginTop: 3, fontWeight: 600 }}>
                  +{delta.toFixed(1)}
                </span>
              )}
            </div>
          </div>

          {/* Row Distribution Cards */}
          <div style={{ flex: 1 }}>
            <span className={`pill ${statusClass}`}>● {statusLabel}</span>
            <p style={{ marginTop: 11, fontSize: 14, color: 'var(--ink-soft)' }}>
              梳理後品質 <b>{session.score_after.toFixed(1)}</b> 分
              {delta > 0 && <span style={{ color: 'var(--green)', fontWeight: 600, marginLeft: 4 }}>(+{delta.toFixed(1)})</span>}
              <span style={{ margin: '0 10px', color: 'var(--ink-faint)' }}>｜</span>
              資料列數 <b>{session.rows_before}</b> 列 → <b>{session.rows_after}</b> 列
            </p>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 14, marginTop: 14 }}>
              <div className="card" style={{ textAlign: 'center' }}>
                <div style={{ fontSize: 12, color: 'var(--ink-faint)', fontWeight: 500, marginBottom: 6, fontFamily: 'var(--mono)' }}>High readiness</div>
                <div style={{ fontSize: 27, fontWeight: 700, color: 'var(--green)' }}>
                  {postDist?.high || 0}
                  {highDelta !== 0 && <span style={{ fontSize: 13, fontWeight: 600, color: highDelta > 0 ? 'var(--green)' : 'var(--rose)', marginLeft: 4 }}>({highDelta > 0 ? '+' : ''}{highDelta})</span>}
                </div>
                <div style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)', marginTop: 4 }}>
                  {totalRows > 0 ? `${Math.round(((postDist?.high || 0) / totalRows) * 100)}%` : '0%'}
                </div>
              </div>
              <div className="card" style={{ textAlign: 'center' }}>
                <div style={{ fontSize: 12, color: 'var(--ink-faint)', fontWeight: 500, marginBottom: 6, fontFamily: 'var(--mono)' }}>Medium</div>
                <div style={{ fontSize: 27, fontWeight: 700, color: 'var(--amber)' }}>
                  {postDist?.medium || 0}
                  {medDelta !== 0 && <span style={{ fontSize: 13, fontWeight: 600, color: medDelta > 0 ? 'var(--amber)' : 'var(--green)', marginLeft: 4 }}>({medDelta > 0 ? '+' : ''}{medDelta})</span>}
                </div>
                <div style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)', marginTop: 4 }}>
                  {totalRows > 0 ? `${Math.round(((postDist?.medium || 0) / totalRows) * 100)}%` : '0%'}
                </div>
              </div>
              <div className="card" style={{ textAlign: 'center' }}>
                <div style={{ fontSize: 12, color: 'var(--ink-faint)', fontWeight: 500, marginBottom: 6, fontFamily: 'var(--mono)' }}>Low</div>
                <div style={{ fontSize: 27, fontWeight: 700, color: 'var(--rose)' }}>
                  {postDist?.low || 0}
                  {lowDelta !== 0 && <span style={{ fontSize: 13, fontWeight: 600, color: lowDelta < 0 ? 'var(--green)' : 'var(--rose)', marginLeft: 4 }}>({lowDelta > 0 ? '+' : ''}{lowDelta})</span>}
                </div>
                <div style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)', marginTop: 4 }}>
                  {totalRows > 0 ? `${Math.round(((postDist?.low || 0) / totalRows) * 100)}%` : '0%'}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* ═══ Six indicators + Radar Chart ═══ */}
        <h3 style={{ fontSize: 17, fontWeight: 700, margin: '28px 0 14px' }}>六項指標改善</h3>
        <div style={{ display: 'flex', gap: 24, alignItems: 'center' }}>
          <div style={{ flex: 1 }}>
            {indicators.map((ind) => {
              const indDelta = ind.after - ind.before
              const baseWidth = Math.max(0, Math.min(100, ind.before))
              const improvementWidth = indDelta > 0 ? Math.min(indDelta, 100 - baseWidth) : 0
              return (
                <div key={ind.nameEn} style={{
                  display: 'flex', alignItems: 'center', gap: 14,
                  padding: '13px 0', borderBottom: '1px solid var(--line-soft)',
                }}>
                  <div style={{ flex: 'none', width: 160 }}>
                    <div style={{ fontSize: 14, fontWeight: 600 }}>{ind.name}</div>
                    <div style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)' }}>{ind.nameEn}</div>
                  </div>
                  <div style={{ flex: 1, height: 7, borderRadius: 5, background: 'var(--line-soft)', overflow: 'hidden', position: 'relative' }}>
                    <div style={{ position: 'absolute', left: 0, top: 0, height: '100%', width: `${baseWidth}%`, background: 'var(--accent)', borderRadius: 5 }} />
                    {improvementWidth > 0 && (
                      <div style={{ position: 'absolute', left: `${baseWidth}%`, top: 0, height: '100%', width: `${improvementWidth}%`, background: 'rgba(34, 197, 94, 0.6)', borderRadius: '0 5px 5px 0' }} />
                    )}
                  </div>
                  <div style={{ flex: 'none', width: 120, textAlign: 'right', fontSize: 12.5, color: 'var(--ink-soft)', fontFamily: 'var(--mono)' }}>
                    {ind.after.toFixed(1)} / 100
                    {indDelta > 0 && (
                      <span style={{ color: 'var(--green)', marginLeft: 4, fontSize: 11 }}>(+{indDelta.toFixed(1)})</span>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
          {/* Radar chart */}
          <div style={{ width: 400, flexShrink: 0, padding: '20px 16px' }}>
            <ResponsiveContainer width="100%" height={340}>
              <RadarChart cx="50%" cy="50%" outerRadius="70%" data={radarData}>
                <PolarGrid />
                <PolarAngleAxis dataKey="subject" style={{ fontSize: 11 }} />
                <PolarRadiusAxis domain={[0, 100]} tick={false} />
                <Radar name="梳理前" dataKey="before" stroke="#94a3b8" fill="#94a3b8" fillOpacity={0.15} />
                <Radar name="梳理後" dataKey="after" stroke="var(--green)" fill="var(--green)" fillOpacity={0.3} />
                <Legend />
              </RadarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* ═══ Issues — resolved vs remaining (expandable cards) ═══ */}
        <div style={{ marginTop: 28 }}>
          <h3 style={{ fontSize: 17, fontWeight: 700, marginBottom: 14 }}>問題解決狀態</h3>

          {/* ── 已修正的問題 (Resolved) ── */}
          <div style={{ marginBottom: 20 }}>
            <h4 style={{ fontSize: 15, fontWeight: 650, marginBottom: 10, color: 'var(--green)' }}>
              已修正的問題
              <span style={{ marginLeft: 8, fontSize: 12, fontFamily: 'var(--mono)', color: 'var(--ink-faint)', fontWeight: 500 }}>
                ({resolvedIssues.length})
              </span>
            </h4>
            {resolvedIssues.length === 0 ? (
              <p style={{ fontSize: 13, color: 'var(--ink-faint)', fontStyle: 'italic' }}>本次梳理未解決任何問題</p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                {resolvedIssues.map((issue, i) => {
                  const key = `resolved-${i}`
                  const isExpanded = expandedIssues.has(key)
                  const postExamples = getPostCleanExamples(issue.title)
                  const hasExamples = postExamples && postExamples.length > 0
                  return (
                    <div key={key} style={{
                      border: '1px solid var(--line)', borderRadius: 10,
                      overflow: 'hidden', background: 'var(--panel)',
                      display: 'flex', alignItems: 'stretch',
                    }}>
                      {/* Green left bar */}
                      <div style={{ width: 5, flexShrink: 0, background: 'var(--green, #22c55e)' }} />
                      <div style={{ flex: 1, minWidth: 0 }}>
                        {/* Header */}
                        <div
                          onClick={() => hasExamples && toggleIssue(key)}
                          style={{
                            padding: '14px 18px',
                            display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between',
                            gap: 16, cursor: hasExamples ? 'pointer' : 'default', userSelect: 'none',
                          }}
                        >
                          <div style={{ flex: 1, minWidth: 0 }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 9, marginBottom: 6 }}>
                              {hasExamples && (
                                <span style={{
                                  display: 'inline-block', fontSize: 11, color: 'var(--ink-faint)',
                                  transition: 'transform 0.2s ease',
                                  transform: isExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                                }}>▶</span>
                              )}
                              <span style={{ fontSize: 15, fontWeight: 650 }}>{issue.title}</span>
                              <span style={{
                                fontFamily: 'var(--mono)', fontSize: 10, padding: '2px 7px',
                                borderRadius: 4, fontWeight: 700, letterSpacing: '0.04em',
                                background: 'rgba(34, 197, 94, 0.1)', color: 'var(--green, #16a34a)',
                              }}>✓ 已修正</span>
                            </div>
                            {issue.description && (
                              <div style={{ fontSize: 13, color: 'var(--ink-soft)', lineHeight: 1.5, paddingLeft: hasExamples ? 21 : 0 }}>
                                {issue.description}
                              </div>
                            )}
                          </div>
                          {issue.affected_rows > 0 && (
                            <div style={{ flexShrink: 0, textAlign: 'right', fontFamily: 'var(--mono)', whiteSpace: 'nowrap' }}>
                              <div style={{ fontSize: 18, fontWeight: 700, color: 'var(--green, #16a34a)', lineHeight: 1.2 }}>
                                {formatCount(issue.affected_rows)}
                              </div>
                              <div style={{ fontSize: 11, color: 'var(--ink-faint)', marginTop: 2 }}>
                                {issue.unit || '列受影響'}
                              </div>
                            </div>
                          )}
                        </div>
                        {/* Expanded: post-cleaning Excel snippet (green highlights) */}
                        {hasExamples && (
                          <div style={{
                            maxHeight: isExpanded ? 600 : 0,
                            overflowY: isExpanded ? 'scroll' : 'hidden',
                            overflowX: 'hidden',
                            transition: isExpanded ? 'max-height 0.3s ease' : 'max-height 0.2s ease',
                          }}>
                            <div style={{ padding: '0 18px 16px 18px', borderTop: '1px solid var(--line-soft)' }}>
                              <ExcelTable
                                examples={postExamples}
                                highlightColor="var(--green, #16a34a)"
                                highlightBg="rgba(34, 197, 94, 0.08)"
                              />
                            </div>
                          </div>
                        )}
                        {!hasExamples && isExpanded && (
                          <div style={{ padding: '0 18px 16px 18px', borderTop: '1px solid var(--line-soft)' }}>
                            <p style={{ fontSize: 13, color: 'var(--green)', fontStyle: 'italic', marginTop: 12 }}>✓ 已修正</p>
                          </div>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>

          {/* ── 尚待解決的問題 (Remaining) ── */}
          <div style={{ marginBottom: 20 }}>
            <h4 style={{ fontSize: 15, fontWeight: 650, marginBottom: 10, color: 'var(--amber)' }}>
              尚待解決的問題
              <span style={{ marginLeft: 8, fontSize: 12, fontFamily: 'var(--mono)', color: 'var(--ink-faint)', fontWeight: 500 }}>
                ({remainingIssues.length})
              </span>
            </h4>
            {remainingIssues.length === 0 ? (
              <p style={{ fontSize: 13, color: 'var(--ink-faint)', fontStyle: 'italic' }}>所有問題已全部修正 🎉</p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                {remainingIssues.map((issue, i) => {
                  const key = `remaining-${i}`
                  const isExpanded = expandedIssues.has(key)
                  const hasExamples = issue.examples && issue.examples.length > 0
                  const sevColor = issue.severity === 'High' ? 'var(--rose)'
                    : issue.severity === 'Medium' ? 'var(--amber)' : 'var(--accent)'
                  const sevBg = issue.severity === 'High' ? 'var(--rose-soft)'
                    : issue.severity === 'Medium' ? 'var(--amber-soft)' : 'var(--accent-soft)'
                  const sevLabel = issue.severity === 'High' ? 'HIGH'
                    : issue.severity === 'Medium' ? 'MED' : 'LOW'
                  return (
                    <div key={key} style={{
                      border: '1px solid var(--line)', borderRadius: 10,
                      overflow: 'hidden', background: 'var(--panel)',
                      display: 'flex', alignItems: 'stretch',
                    }}>
                      {/* Severity colored left bar */}
                      <div style={{ width: 5, flexShrink: 0, background: sevColor }} />
                      <div style={{ flex: 1, minWidth: 0 }}>
                        {/* Header — always visible */}
                        <div
                          onClick={() => hasExamples && toggleIssue(key)}
                          style={{
                            padding: '14px 18px',
                            display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between',
                            gap: 16, cursor: hasExamples ? 'pointer' : 'default', userSelect: 'none',
                          }}
                        >
                          <div style={{ flex: 1, minWidth: 0 }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 9, marginBottom: 6 }}>
                              {hasExamples && (
                                <span style={{
                                  display: 'inline-block', fontSize: 11, color: 'var(--ink-faint)',
                                  transition: 'transform 0.2s ease',
                                  transform: isExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                                }}>▶</span>
                              )}
                              <span style={{ fontSize: 15, fontWeight: 650 }}>
                                {issue.title || issue.indicator}
                              </span>
                              <span style={{
                                fontFamily: 'var(--mono)', fontSize: 10, padding: '2px 7px',
                                borderRadius: 4, fontWeight: 700, letterSpacing: '0.04em',
                                background: sevBg, color: sevColor,
                              }}>{sevLabel}</span>
                            </div>
                            {/* Description */}
                            {issue.description && (
                              issue.description.includes('\n') ? (
                                <div style={{ paddingLeft: hasExamples ? 21 : 0 }}>
                                  <div style={{ fontSize: 13, color: 'var(--ink-soft)', lineHeight: 1.5, marginBottom: 4 }}>
                                    {issue.description.split('\n')[0]}
                                  </div>
                                  <ul style={{ margin: 0, paddingLeft: 18, listStyleType: 'disc', paddingTop: 0 }}>
                                    {issue.description.split('\n').slice(1).filter(Boolean).map((line, li) => (
                                      <li key={li} style={{ marginBottom: 2, fontSize: 13, color: 'var(--ink-soft)', lineHeight: 1.5 }}>
                                        {line}
                                      </li>
                                    ))}
                                  </ul>
                                </div>
                              ) : (
                                <div style={{ fontSize: 13, color: 'var(--ink-soft)', lineHeight: 1.5, paddingLeft: hasExamples ? 21 : 0 }}>
                                  {issue.description}
                                </div>
                              )
                            )}
                          </div>
                          {/* Affected count */}
                          {issue.affected_rows > 0 && (
                            <div style={{ flexShrink: 0, textAlign: 'right', fontFamily: 'var(--mono)', whiteSpace: 'nowrap' }}>
                              <div style={{ fontSize: 18, fontWeight: 700, color: sevColor, lineHeight: 1.2 }}>
                                {formatCount(issue.affected_rows)}
                              </div>
                              <div style={{ fontSize: 11, color: 'var(--ink-faint)', marginTop: 2 }}>
                                {issue.unit || '列受影響'}
                              </div>
                            </div>
                          )}
                        </div>
                        {/* Expanded: Excel spreadsheet snippet */}
                        {hasExamples && (
                          <div style={{
                            maxHeight: isExpanded ? 600 : 0,
                            overflowY: isExpanded ? 'scroll' : 'hidden',
                            overflowX: 'hidden',
                            transition: isExpanded ? 'max-height 0.3s ease' : 'max-height 0.2s ease',
                          }}>
                            <div style={{ padding: '0 18px 16px 18px', borderTop: '1px solid var(--line-soft)' }}>
                              <ExcelTable examples={issue.examples!} strikethrough={issue.indicator === 'strikethrough_formatting'} />
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>

        {/* ═══ Download Section ═══ */}
        <div style={{ marginTop: 28 }}>
          <h3 style={{ fontSize: 17, fontWeight: 700, marginBottom: 14 }}>產出下載</h3>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 14 }}>
            {downloads.map((dl) => (
              <button key={dl.type} onClick={() => handleDownload(dl.type)} disabled={downloading === dl.type}
                style={{ border: '1.5px solid var(--line)', borderRadius: 12, padding: 20, textAlign: 'center', cursor: 'pointer', transition: 'all 0.15s', background: 'var(--panel)', opacity: downloading === dl.type ? 0.5 : 1 }}>
                <div style={{ fontSize: 28, marginBottom: 8 }}>{dl.icon}</div>
                <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 4 }}>{dl.label}</div>
                <div style={{ fontSize: 12, fontFamily: 'var(--mono)', color: 'var(--ink-faint)' }}>{downloading === dl.type ? '下載中...' : dl.filename}</div>
                <div style={{ fontSize: 11.5, color: 'var(--ink-soft)', marginTop: 6 }}>{dl.desc}</div>
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Footer */}
      <div style={{ padding: '16px 28px', borderTop: '1px solid var(--line-soft)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: 'var(--panel)' }}>
        <span style={{ fontSize: 12.5, color: 'var(--ink-faint)' }}>下載完成後可進行存證作業</span>
        <button className="btn btn-primary" onClick={() => navigate('/evidence')}>下一步 →</button>
      </div>
    </div>
  )
}
