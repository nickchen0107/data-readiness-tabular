import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { PieChart, Pie, Cell, RadarChart, Radar, PolarGrid, PolarAngleAxis, PolarRadiusAxis, ResponsiveContainer } from 'recharts'
import apiClient from '../api/client'

interface Indicator {
  name: string
  nameEn: string
  score: number
  color: string
}

interface CellMerge {
  start_col: number
  span: number
}

interface IssueExample {
  label?: string
  headers: string[]
  row_number: number
  cells: string[]
  highlights: number[]
  merges?: CellMerge[]
  format_labels?: string[]
}

interface Issue {
  title: string
  severity: string
  description: string
  affected_rows: number
  unit: string
  indicator: string
  examples?: IssueExample[]
}

interface Assessment {
  id: string
  total_score: number
  status: string
  filename?: string
  total_rows?: number
  row_completeness: number
  column_completeness: number
  format_consistency: number
  duplicate_similar: number
  table_structure: number
  ai_query_readiness: number
  issues: Issue[]
  row_distribution: {
    high: number
    medium: number
    low: number
  }
}

export default function AssessmentPage() {
  const [assessment, setAssessment] = useState<Assessment | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [expandedIssues, setExpandedIssues] = useState<Set<number>>(new Set())
  const [hoveredIndicator, setHoveredIndicator] = useState<string | null>(null)
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { t } = useTranslation()

  useEffect(() => {
    loadAssessment()
  }, [])

  const loadAssessment = async () => {
    try {
      const id = searchParams.get('id')
      if (id) {
        const res = await apiClient.get(`/assess/${id}`)
        setAssessment(res.data)
      } else {
        const res = await apiClient.get('/assess/latest')
        setAssessment(res.data)
      }
    } catch (err: unknown) {
      const axiosErr = err as { response?: { status?: number; data?: { error?: { message?: string } } } }
      if (axiosErr.response?.status === 404) {
        setError(t('error.assessment_not_found'))
      } else {
        setError(axiosErr.response?.data?.error?.message || t('error.load_assessment_failed'))
      }
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 60, color: 'var(--ink-faint)' }}>
        {t('common.loading_assessment')}
      </div>
    )
  }

  if (error || !assessment) {
    return (
      <div style={{ textAlign: 'center', padding: 60 }}>
        <p style={{ color: 'var(--rose)', marginBottom: 16 }}>{error}</p>
        <button className="btn btn-ghost" onClick={() => navigate('/upload')}>
          {t('btn.back_upload')}
        </button>
      </div>
    )
  }

  const indicatorInfo: Record<string, { desc: string; calc: string; source?: string }> = {
    [t('indicator.row_completeness')]: { desc: t('indicator.row_completeness.desc'), calc: t('indicator.row_completeness.calc'), source: t('indicator.row_completeness.source') },
    [t('indicator.column_completeness')]: { desc: t('indicator.column_completeness.desc'), calc: t('indicator.column_completeness.calc'), source: t('indicator.column_completeness.source') },
    [t('indicator.format_consistency')]: { desc: t('indicator.format_consistency.desc'), calc: t('indicator.format_consistency.calc'), source: t('indicator.format_consistency.source') },
    [t('indicator.data_uniqueness')]: { desc: t('indicator.data_uniqueness.desc'), calc: t('indicator.data_uniqueness.calc'), source: t('indicator.data_uniqueness.source') },
    [t('indicator.table_structure')]: { desc: t('indicator.table_structure.desc'), calc: t('indicator.table_structure.calc'), source: t('indicator.table_structure.source') },
    [t('indicator.ai_query_readiness')]: { desc: t('indicator.ai_query_readiness.desc'), calc: t('indicator.ai_query_readiness.calc'), source: t('indicator.ai_query_readiness.source') },
  }

  const indicators: Indicator[] = [
    { name: t('indicator.row_completeness'), nameEn: 'Row Completeness', score: assessment.row_completeness, color: 'var(--accent)' },
    { name: t('indicator.column_completeness'), nameEn: 'Column Completeness', score: assessment.column_completeness, color: 'var(--accent)' },
    { name: t('indicator.format_consistency'), nameEn: 'Format Consistency', score: assessment.format_consistency, color: 'var(--green)' },
    { name: t('indicator.data_uniqueness'), nameEn: 'Data Uniqueness', score: assessment.duplicate_similar, color: 'var(--amber)' },
    { name: t('indicator.table_structure'), nameEn: 'Table Structure', score: assessment.table_structure, color: 'var(--accent)' },
    { name: t('indicator.ai_query_readiness'), nameEn: 'AI Query Readiness', score: assessment.ai_query_readiness, color: 'var(--green)' },
  ]

  const radarData = indicators.map(ind => ({ subject: ind.name, score: ind.score }))

  const statusLabel = assessment.status === 'ready' ? 'Ready'
    : assessment.status === 'conditional' ? 'Conditional' : 'Not Ready'
  const statusClass = assessment.status === 'ready' ? 'ready'
    : assessment.status === 'conditional' ? 'cond' : 'not'

  const totalRows = (assessment.row_distribution?.high || 0) + (assessment.row_distribution?.medium || 0) + (assessment.row_distribution?.low || 0)

  const displayRowCount = assessment.total_rows || totalRows

  const gaugeData = [
    { value: assessment.total_score },
    { value: 100 - assessment.total_score },
  ]
  const gaugeColor = assessment.status === 'ready' ? '#15803d'
    : assessment.status === 'conditional' ? '#b45309' : '#b42318'

  const formatCount = (count: number): string => {
    if (count >= 10000) {
      return (count / 10000).toFixed(1).replace(/\.0$/, '') + t('misc.ten_thousand')
    }
    return count.toLocaleString()
  }

  const toggleIssue = (index: number) => {
    setExpandedIssues(prev => {
      const next = new Set(prev)
      if (next.has(index)) {
        next.delete(index)
      } else {
        next.add(index)
      }
      return next
    })
  }

  return (
    <div style={{
      background: 'var(--paper)', border: '1px solid var(--line)',
      borderRadius: 14, overflow: 'hidden',
    }}>
      {/* Header */}
      <div style={{
        padding: '20px 28px', borderBottom: '1px solid var(--line-soft)',
        display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between',
      }}>
        <div>
          <div style={{
            fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--accent)',
            letterSpacing: '0.08em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 5,
          }}>STEP 2</div>
          <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>{t('page.assessment.title')}</h2>
        </div>
        <span className={`pill ${statusClass}`}>● {statusLabel}</span>
      </div>

      {/* Body */}
      <div style={{ padding: 28 }}>
        {/* Score + indicators */}
        <div style={{ display: 'flex', gap: 26, alignItems: 'center', marginBottom: 28 }}>
          {/* Ring chart */}
          <div style={{ position: 'relative', width: 140, height: 140, flexShrink: 0 }}>
            <PieChart width={140} height={140}>
              <Pie
                data={gaugeData}
                cx={65}
                cy={65}
                innerRadius={48}
                outerRadius={62}
                startAngle={90}
                endAngle={-270}
                dataKey="value"
                stroke="none"
              >
                <Cell fill={gaugeColor} />
                <Cell fill="var(--line-soft)" />
              </Pie>
            </PieChart>
            <div style={{
              position: 'absolute', inset: 0, display: 'flex', flexDirection: 'column',
              alignItems: 'center', justifyContent: 'center',
            }}>
              <span style={{ fontSize: 34, fontWeight: 750, letterSpacing: '-0.03em', lineHeight: 1 }}>
                {assessment.total_score.toFixed(1)}
              </span>
              <span style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)', marginTop: 3 }}>
                / 100
              </span>
            </div>
          </div>

          {/* Row Distribution Cards */}
          <div style={{ flex: 1 }}>
            <span className={`pill ${statusClass}`}>● {statusLabel}</span>
            <p style={{ marginTop: 11, fontSize: 14, color: 'var(--ink-soft)' }}>
              {t('assessment.file_summary', { filename: assessment.filename || 'Excel', rows: displayRowCount })} {assessment.status === 'not_ready' ? t('assessment.not_ready_hint') : assessment.status === 'conditional' ? t('assessment.conditional_hint') : t('assessment.ai_ready_hint')}
            </p>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 14, marginTop: 14 }}>
              <div className="card" style={{ textAlign: 'center' }}>
                <div style={{ fontSize: 12, color: 'var(--ink-faint)', fontWeight: 500, marginBottom: 6, fontFamily: 'var(--mono)' }}>{t('distribution.high_readiness')}</div>
                <div style={{ fontSize: 27, fontWeight: 700, color: 'var(--green)' }}>
                  {assessment.row_distribution?.high || 0}
                </div>
                <div style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)', marginTop: 4 }}>
                  {totalRows > 0 ? `${Math.round(((assessment.row_distribution?.high || 0) / totalRows) * 100)}%` : '0%'}
                </div>
              </div>
              <div className="card" style={{ textAlign: 'center' }}>
                <div style={{ fontSize: 12, color: 'var(--ink-faint)', fontWeight: 500, marginBottom: 6, fontFamily: 'var(--mono)' }}>{t('distribution.medium')}</div>
                <div style={{ fontSize: 27, fontWeight: 700, color: 'var(--amber)' }}>
                  {assessment.row_distribution?.medium || 0}
                </div>
                <div style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)', marginTop: 4 }}>
                  {totalRows > 0 ? `${Math.round(((assessment.row_distribution?.medium || 0) / totalRows) * 100)}%` : '0%'}
                </div>
              </div>
              <div className="card" style={{ textAlign: 'center' }}>
                <div style={{ fontSize: 12, color: 'var(--ink-faint)', fontWeight: 500, marginBottom: 6, fontFamily: 'var(--mono)' }}>{t('distribution.low')}</div>
                <div style={{ fontSize: 27, fontWeight: 700, color: 'var(--rose)' }}>
                  {assessment.row_distribution?.low || 0}
                </div>
                <div style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)', marginTop: 4 }}>
                  {totalRows > 0 ? `${Math.round(((assessment.row_distribution?.low || 0) / totalRows) * 100)}%` : '0%'}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Six indicators section */}
        <h3 style={{ fontSize: 17, fontWeight: 700, margin: '28px 0 14px' }}>{t('assessment.six_indicators')}</h3>
        <div style={{ display: 'flex', gap: 24, alignItems: 'center' }}>
          {/* Indicator bars */}
          <div style={{ flex: 1 }}>
            {indicators.map((ind) => (
              <div key={ind.nameEn} style={{
                display: 'flex', alignItems: 'center', gap: 14,
                padding: '13px 0', borderBottom: '1px solid var(--line-soft)',
                position: 'relative',
              }}>
                <div style={{ flex: 'none', width: 160 }}>
                  <div style={{ fontSize: 14, fontWeight: 600 }}>
                    {ind.name}
                    <span
                      onMouseEnter={() => setHoveredIndicator(ind.name)}
                      onMouseLeave={() => setHoveredIndicator(null)}
                      style={{ cursor: 'pointer', marginLeft: 4, color: '#3f57a5', fontSize: 12 }}
                    >ⓘ</span>
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)' }}>
                    {ind.nameEn}
                  </div>
                </div>
                <div style={{
                  flex: 1, height: 7, borderRadius: 5, background: 'var(--line-soft)', overflow: 'hidden',
                }}>
                  <div style={{
                    height: '100%', borderRadius: 5, width: `${ind.score}%`,
                    background: ind.color,
                  }} />
                </div>
                <div style={{
                  flex: 'none', width: 80, textAlign: 'right',
                  fontSize: 12.5, color: 'var(--ink-soft)', fontFamily: 'var(--mono)',
                }}>
                  {ind.score.toFixed(1)} / 100
                </div>
                {hoveredIndicator === ind.name && indicatorInfo[ind.name] && (
                  <div style={{
                    position: 'absolute', top: '100%', left: 0, marginTop: 4,
                    background: 'var(--panel)', border: '1px solid var(--line)',
                    borderRadius: 8, padding: '10px 14px', fontSize: 12,
                    boxShadow: '0 4px 12px rgba(0,0,0,0.08)', zIndex: 10,
                    minWidth: 220, maxWidth: 320, whiteSpace: 'normal',
                  }}>
                    <div style={{ fontWeight: 600, marginBottom: 4 }}>{indicatorInfo[ind.name].desc}</div>
                    <div style={{ color: 'var(--ink-faint)' }}>{t('misc.calculation')}：{indicatorInfo[ind.name].calc}</div>
                    {indicatorInfo[ind.name].source && (
                      <div style={{ color: 'var(--accent)', fontSize: 11, marginTop: 6, fontFamily: 'var(--mono)' }}>
                        {t('misc.reference')}：{indicatorInfo[ind.name].source}
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
          {/* Radar chart */}
          <div style={{ width: 400, flexShrink: 0, padding: '20px 16px' }}>
            <ResponsiveContainer width="100%" height={340}>
              <RadarChart cx="50%" cy="50%" outerRadius="70%" data={radarData}>
                <PolarGrid />
                <PolarAngleAxis dataKey="subject" style={{ fontSize: 11 }} />
                <PolarRadiusAxis domain={[0, 100]} tick={false} />
                <Radar dataKey="score" fill="var(--accent)" fillOpacity={0.35} stroke="var(--accent)" strokeWidth={2} />
              </RadarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Issues */}
        {assessment.issues && assessment.issues.length > 0 && (
          <div style={{ marginTop: 28 }}>
            <h3 style={{ fontSize: 17, fontWeight: 700, marginBottom: 14 }}>{t('assessment.issue_list')}</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {assessment.issues.map((issue, i) => {
                const sevColor = issue.severity === 'High' ? 'var(--rose)'
                  : issue.severity === 'Medium' ? 'var(--amber)' : 'var(--accent)'
                const sevBg = issue.severity === 'High' ? 'var(--rose-soft)'
                  : issue.severity === 'Medium' ? 'var(--amber-soft)' : 'var(--accent-soft)'
                const sevLabel = issue.severity === 'High' ? t('severity.high')
                  : issue.severity === 'Medium' ? t('severity.medium') : t('severity.low')
                const isExpanded = expandedIssues.has(i)
                const hasExamples = issue.examples && issue.examples.length > 0
                return (
                  <div key={i} style={{
                    border: '1px solid var(--line)', borderRadius: 10,
                    overflow: 'hidden', background: 'var(--panel)',
                    display: 'flex', alignItems: 'stretch',
                  }}>
                    {/* Left severity bar */}
                    <div style={{
                      width: 5, flexShrink: 0, background: sevColor,
                    }} />
                    {/* Main content */}
                    <div style={{ flex: 1, minWidth: 0 }}>
                      {/* Header — always visible */}
                      <div
                        onClick={() => hasExamples && toggleIssue(i)}
                        style={{
                          padding: '14px 18px',
                          display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between',
                          gap: 16, cursor: hasExamples ? 'pointer' : 'default', userSelect: 'none',
                        }}
                      >
                        <div style={{ flex: 1, minWidth: 0 }}>
                          {/* Title row with badge */}
                          <div style={{ display: 'flex', alignItems: 'center', gap: 9, marginBottom: 6 }}>
                            {hasExamples && (
                              <span style={{
                                display: 'inline-block',
                                fontSize: 11, color: 'var(--ink-faint)',
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
                            }}>
                              {sevLabel}
                            </span>
                          </div>
                          {/* Description — always visible */}
                          {issue.description.includes('\n') ? (
                            <div style={{ paddingLeft: hasExamples ? 21 : 0 }}>
                              <div style={{ fontSize: 13, color: 'var(--ink-soft)', lineHeight: 1.5, marginBottom: 4 }}>
                                {issue.description.split('\n')[0]}
                              </div>
                              <ul style={{
                                margin: 0, paddingLeft: 18, listStyleType: 'disc',
                                paddingTop: 0,
                              }}>
                                {issue.description.split('\n').slice(1).filter(Boolean).map((line, li) => (
                                  <li key={li} style={{
                                    marginBottom: 2, fontSize: 13,
                                    color: 'var(--ink-soft)', lineHeight: 1.5,
                                  }}>
                                    {line}
                                  </li>
                                ))}
                              </ul>
                            </div>
                          ) : (
                            <div style={{
                              fontSize: 13, color: 'var(--ink-soft)', lineHeight: 1.5,
                              paddingLeft: hasExamples ? 21 : 0,
                            }}>
                              {issue.description.replace(/\n/g, ' ')}
                            </div>
                          )}
                        </div>
                        {/* Affected count */}
                        {issue.affected_rows > 0 && (
                          <div style={{
                            flexShrink: 0, textAlign: 'right',
                            fontFamily: 'var(--mono)', whiteSpace: 'nowrap',
                          }}>
                            <div style={{ fontSize: 18, fontWeight: 700, color: sevColor, lineHeight: 1.2 }}>
                              {formatCount(issue.affected_rows)}
                            </div>
                            <div style={{ fontSize: 11, color: 'var(--ink-faint)', marginTop: 2 }}>
                              {issue.unit || t('common.rows_affected')}
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
                          <div style={{
                            padding: '0 18px 16px 18px',
                            borderTop: '1px solid var(--line-soft)',
                          }}>
                            {(() => {
                              // Group examples by label
                              const groups: Array<{label: string | undefined, items: IssueExample[]}> = []
                              let currentLabel: string | undefined = undefined
                              let currentItems: IssueExample[] = []
                              for (const ex of issue.examples!) {
                                if (ex.label !== currentLabel) {
                                  if (currentItems.length > 0) {
                                    groups.push({label: currentLabel, items: currentItems})
                                  }
                                  currentLabel = ex.label
                                  currentItems = [ex]
                                } else {
                                  currentItems.push(ex)
                                }
                              }
                              if (currentItems.length > 0) {
                                groups.push({label: currentLabel, items: currentItems})
                              }

                              return groups.slice(0, 3).map((group, gIdx) => (
                                <div key={gIdx}>
                                  {group.label && (
                                    <div style={{fontSize: 12, fontWeight: 600, color: 'var(--ink-soft)', margin: '12px 0 6px', paddingLeft: 4}}>● {group.label}</div>
                                  )}
                                  <div style={{
                                    marginTop: group.label ? 0 : 12, borderRadius: 8, overflow: 'auto',
                                    border: '1px solid var(--line-soft)',
                                  }}>
                                    <table style={{
                                      width: '100%', borderCollapse: 'collapse',
                                      fontSize: 12, fontFamily: 'var(--mono)',
                                      tableLayout: 'auto', minWidth: 'max-content',
                                    }}>
                                      {/* Header row */}
                                      <thead>
                                        <tr>
                                          {/* Row number column header */}
                                          <th style={{
                                            background: '#f3f4f6', padding: '6px 10px',
                                            border: '1px solid #e5e7eb', fontSize: 11,
                                            color: 'var(--ink-faint)', fontWeight: 600,
                                            textAlign: 'center', width: 40, minWidth: 40,
                                          }}>#</th>
                                          {(() => {
                                            const isFirstRowHeader = group.items[0].row_number === 1
                                            const headerRow = isFirstRowHeader ? group.items[0] : null
                                            const headerContent = isFirstRowHeader ? group.items[0].cells : group.items[0].headers

                                            // Build merge info for header row
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
                                                  background: isHighlighted ? 'rgba(220, 38, 38, 0.08)' : '#f3f4f6',
                                                  padding: '6px 10px',
                                                  border: isHighlighted ? '1.5px solid var(--rose, #dc2626)' : '1px solid #e5e7eb',
                                                  fontSize: 11,
                                                  color: isHighlighted ? 'var(--rose, #dc2626)' : 'var(--ink-soft)',
                                                  fontWeight: isHighlighted ? 600 : 600,
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
                                            {/* Row number cell (grey like Excel) */}
                                            <td style={{
                                              background: '#f3f4f6', padding: '5px 8px',
                                              border: '1px solid #e5e7eb',
                                              color: 'var(--ink-faint)', fontWeight: 600,
                                              textAlign: 'center', fontSize: 11,
                                            }}>
                                              {ex.row_number > 0 ? ex.row_number : ''}
                                            </td>
                                            {/* Data cells — handle merges */}
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
                                                    border: isHighlighted
                                                      ? '1.5px solid var(--rose, #dc2626)'
                                                      : '1px solid #e5e7eb',
                                                    background: isMerged
                                                      ? 'rgba(59, 130, 246, 0.06)'
                                                      : isHighlighted
                                                        ? 'rgba(220, 38, 38, 0.06)'
                                                        : 'var(--paper)',
                                                    color: isMerged ? 'var(--accent)' : isHighlighted ? 'var(--rose, #dc2626)' : 'var(--ink-soft)',
                                                    whiteSpace: 'pre-line',
                                                    fontWeight: isMerged ? 600 : isHighlighted ? 600 : 400,
                                                    textAlign: isMerged ? 'center' : 'left',
                                                    textDecoration: isHighlighted && issue.indicator === 'strikethrough_formatting' ? 'line-through' : 'none',
                                                  }}>
                                                    {isMerged ? `⬌ ${cell || `(${t('clean.merged_cell_label')})`}` : isEmpty ? '—' : cell}
                                                    {ex.format_labels?.[k] && (
                                                      <div style={{ marginTop: 2 }}>
                                                        <span style={{
                                                          fontSize: 10, fontFamily: 'var(--mono)',
                                                          background: 'rgba(99, 102, 241, 0.1)',
                                                          color: '#4338ca',
                                                          borderRadius: 3, padding: '1px 5px',
                                                          fontWeight: 500,
                                                        }}>
                                                          {ex.format_labels[k]}
                                                        </span>
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
                              ))
                            })()}
                            {hasExamples && issue.examples!.length >= 5 && (
                              <div style={{
                                marginTop: 8, fontSize: 11, color: 'var(--ink-faint)',
                                textAlign: 'center', fontFamily: 'var(--mono)',
                              }}>
                                {t('common.show_first_n')}
                              </div>
                            )}
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>

            {/* Best Row Benchmark note */}
            <div style={{
              marginTop: 16, padding: '12px 16px',
              background: 'var(--accent-soft, rgba(59,130,246,0.06))',
              borderRadius: 8, fontSize: 12.5, color: 'var(--ink-soft)',
              lineHeight: 1.6, display: 'flex', gap: 8, alignItems: 'flex-start',
            }}>
              <span style={{ flexShrink: 0, fontSize: 14 }}>ⓘ</span>
              <span>
                {t('assessment.best_row_note')}
              </span>
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
          {t('assessment.complete_hint')}
        </span>
        <button className="btn btn-primary" onClick={() => navigate('/routing')}>
          {t('btn.next_step')} →
        </button>
      </div>
    </div>
  )
}
