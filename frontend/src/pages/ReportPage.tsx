import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import apiClient from '../api/client'

interface Issue {
  title: string
  title_en?: string
  severity: string
  affected_rows: number
  unit?: string
  description: string
  description_en?: string
  indicator?: string
  examples?: IssueExample[]
}

interface IssueExample {
  label?: string
  headers: string[]
  row_number: number
  cells: string[]
  highlights?: number[]
  merges?: { start: number; span: number }[]
  format_labels?: string[]
}

interface AssessmentSummary {
  total_score: number
  status: string
  row_completeness: number
  column_completeness: number
  format_consistency: number
  duplicate_similar: number
  table_structure: number
  ai_query_readiness: number
  issues: Issue[]
}

interface ComparisonData {
  session: {
    id: string
    rows_before: number
    rows_after: number
    score_before: number
    score_after: number
    rules_applied: string[]
    original_filename?: string
  }
  original_assessment: AssessmentSummary
  post_clean_assessment: AssessmentSummary
}

export default function ReportPage() {
  const [searchParams] = useSearchParams()
  const { i18n } = useTranslation()
  const [data, setData] = useState<ComparisonData | null>(null)
  const [loading, setLoading] = useState(true)

  const sessionId = searchParams.get('session_id')
  const locale = searchParams.get('locale') || 'en'

  useEffect(() => {
    if (locale) i18n.changeLanguage(locale === 'zh-TW' ? 'zh-TW' : 'en')
  }, [locale, i18n])

  useEffect(() => {
    if (!sessionId) {
      // Check if data is embedded in window (for server-side rendering via chromedp)
      const embedded = (window as unknown as { __REPORT_DATA__?: ComparisonData }).__REPORT_DATA__
      if (embedded) {
        setData(embedded)
        setLoading(false)
        setTimeout(() => { document.title = 'REPORT_READY' }, 300)
        return
      }
      setLoading(false)
      return
    }
    apiClient.get(`/compare/${sessionId}`).then(res => {
      setData(res.data)
      setLoading(false)
      // Signal to chromedp that page is ready
      setTimeout(() => { document.title = 'REPORT_READY' }, 300)
    }).catch(() => setLoading(false))
  }, [sessionId])

  if (loading) return <div style={{ padding: 40, textAlign: 'center' }}>Loading report...</div>
  if (!data) return <div style={{ padding: 40 }}>Error loading report data</div>

  const isEn = locale === 'en'
  const { original_assessment, post_clean_assessment, session } = data

  const getRuleLabel = (rule: string) => {
    if (isEn) {
      const map: Record<string, string> = { date_normalize: 'Date Normalization', dedup: 'Remove Duplicates', name_normalize: 'Name Normalization', subtotal_remove: 'Remove Subtotal Rows', fill_na: 'Fill Empty Values', empty_col_remove: 'Remove Empty Columns', keep_block: 'Keep Data Block' }
      return map[rule] || rule
    }
    const map: Record<string, string> = { date_normalize: '統一日期格式', dedup: '移除重複列', name_normalize: '客戶名正規化', subtotal_remove: '移除小計列', fill_na: '填補空值', empty_col_remove: '移除空白欄位', keep_block: '保留資料區塊' }
    return map[rule] || rule
  }

  const getGradeLabel = (grade: string) => {
    if (isEn) return grade === 'ready' ? 'Ready' : grade === 'conditional' ? 'Conditional' : 'Not Ready'
    return grade === 'ready' ? '已就緒' : grade === 'conditional' ? '有條件就緒' : '未就緒'
  }

  const getGradeColor = (grade: string) => grade === 'ready' ? '#15803d' : grade === 'conditional' ? '#b45309' : '#b42318'

  return (
    <div style={{ fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif', maxWidth: 800, margin: '0 auto', padding: '30px 40px', color: '#1a1f2e', fontSize: 13, lineHeight: 1.6 }}>

      {/* ─── Cover ─── */}
      <div style={{ textAlign: 'center', marginBottom: 50, paddingTop: 60, pageBreakAfter: 'always' }}>
        <h1 style={{ fontSize: 36, fontWeight: 700, marginBottom: 10 }}>S.A.F.E.-AI</h1>
        <h2 style={{ fontSize: 22, fontWeight: 500, color: '#555' }}>{isEn ? 'Data Quality Report' : '資料梳理報告'}</h2>
        {session.original_filename && <p style={{ marginTop: 20, fontSize: 14, color: '#666' }}>{isEn ? 'Source File' : '來源檔案'}: {session.original_filename}</p>}
        <p style={{ marginTop: 10, fontSize: 12, color: '#999' }}>Generated: {new Date().toLocaleString()}</p>
      </div>

      {/* ─── Section 1: Original Assessment ─── */}
      <h2 style={{ fontSize: 18, borderBottom: '2px solid #2b6cb0', paddingBottom: 6, marginBottom: 16 }}>
        {isEn ? '1. Original Assessment' : '1. 原始評估'}
      </h2>
      <ScoreSection assess={original_assessment} getGradeLabel={getGradeLabel} getGradeColor={getGradeColor} isEn={isEn} />
      <IndicatorTable assess={original_assessment} isEn={isEn} />
      <IssueSection issues={original_assessment.issues} isEn={isEn} title={isEn ? 'Issues Detected' : '偵測到的問題'} />

      <div style={{ pageBreakBefore: 'always' }} />

      {/* ─── Section 2: Cleaning Operations ─── */}
      <h2 style={{ fontSize: 18, borderBottom: '2px solid #2b6cb0', paddingBottom: 6, marginBottom: 16 }}>
        {isEn ? '2. Cleaning Operations' : '2. 梳理操作'}
      </h2>
      <h3 style={{ fontSize: 14, marginBottom: 8 }}>{isEn ? 'Rules Applied' : '已套用規則'}</h3>
      {session.rules_applied.length > 0 ? (
        <ol style={{ paddingLeft: 20, marginBottom: 20 }}>
          {session.rules_applied.map((r, i) => <li key={i} style={{ marginBottom: 4 }}>{getRuleLabel(r)}</li>)}
        </ol>
      ) : <p style={{ color: '#666' }}>{isEn ? 'No rules applied' : '未套用任何規則'}</p>}

      <h3 style={{ fontSize: 14, marginBottom: 8 }}>{isEn ? 'Before / After' : '前後對比'}</h3>
      <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: 20 }}>
        <thead><tr style={{ background: '#2b6cb0', color: '#fff' }}>
          <th style={thStyle}>{isEn ? 'Metric' : '指標'}</th>
          <th style={thStyle}>{isEn ? 'Before' : '梳理前'}</th>
          <th style={thStyle}>{isEn ? 'After' : '梳理後'}</th>
        </tr></thead>
        <tbody>
          <tr><td style={tdStyle}>{isEn ? 'Data Rows' : '資料列數'}</td><td style={tdStyle}>{session.rows_before}</td><td style={tdStyle}>{session.rows_after}</td></tr>
          <tr style={{ background: '#f9fafb' }}><td style={tdStyle}>{isEn ? 'Score' : '評分'}</td><td style={tdStyle}>{session.score_before.toFixed(1)}</td><td style={{ ...tdStyle, color: '#15803d', fontWeight: 600 }}>{session.score_after.toFixed(1)}</td></tr>
        </tbody>
      </table>

      <div style={{ pageBreakBefore: 'always' }} />

      {/* ─── Section 3: Post-Cleaning Assessment ─── */}
      <h2 style={{ fontSize: 18, borderBottom: '2px solid #2b6cb0', paddingBottom: 6, marginBottom: 16 }}>
        {isEn ? '3. Post-Cleaning Assessment' : '3. 梳理後評估'}
      </h2>
      <ScoreSection assess={post_clean_assessment} getGradeLabel={getGradeLabel} getGradeColor={getGradeColor} isEn={isEn} />
      <IndicatorTable assess={post_clean_assessment} isEn={isEn} />

      <div style={{ pageBreakBefore: 'always' }} />

      {/* ─── Section 4: Remaining Issues ─── */}
      <h2 style={{ fontSize: 18, borderBottom: '2px solid #2b6cb0', paddingBottom: 6, marginBottom: 16 }}>
        {isEn ? '4. Remaining Issues (Manual Processing Required)' : '4. 待手動處理的問題'}
      </h2>
      <p style={{ color: '#666', fontSize: 12, marginBottom: 12 }}>
        {isEn ? 'These issues could not be resolved automatically and require manual intervention.' : '以下問題無法透過自動化規則處理，需要人工介入。'}
      </p>
      <IssueSection issues={post_clean_assessment.issues} isEn={isEn} title="" />
    </div>
  )
}

const thStyle: React.CSSProperties = { padding: '8px 12px', textAlign: 'center', fontSize: 12, fontWeight: 600 }
const tdStyle: React.CSSProperties = { padding: '6px 12px', textAlign: 'center', fontSize: 12, border: '1px solid #e5e7eb' }

function ScoreSection({ assess, getGradeLabel, getGradeColor }: { assess: AssessmentSummary; getGradeLabel: (g: string) => string; getGradeColor: (g: string) => string; isEn?: boolean }) {
  return (
    <div style={{ textAlign: 'center', marginBottom: 20 }}>
      <div style={{ fontSize: 48, fontWeight: 700, color: getGradeColor(assess.status) }}>{assess.total_score.toFixed(1)}</div>
      <div style={{ fontSize: 14, fontWeight: 600, color: getGradeColor(assess.status) }}>{getGradeLabel(assess.status)}</div>
    </div>
  )
}

function IndicatorTable({ assess, isEn }: { assess: AssessmentSummary; isEn: boolean }) {
  const indicators = [
    { name: isEn ? 'Row Completeness' : '列完整度', score: assess.row_completeness },
    { name: isEn ? 'Column Completeness' : '欄完整度', score: assess.column_completeness },
    { name: isEn ? 'Format Consistency' : '格式一致性', score: assess.format_consistency },
    { name: isEn ? 'Duplicate/Similar' : '重複/相似', score: assess.duplicate_similar },
    { name: isEn ? 'Table Structure' : '表格結構', score: assess.table_structure },
    { name: isEn ? 'AI Query Readiness' : 'AI查詢準備度', score: assess.ai_query_readiness },
  ]
  return (
    <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: 20 }}>
      <thead><tr style={{ background: '#2b6cb0', color: '#fff' }}>
        <th style={thStyle}>{isEn ? 'Indicator' : '指標'}</th>
        <th style={thStyle}>{isEn ? 'Score' : '分數'}</th>
      </tr></thead>
      <tbody>
        {indicators.map((ind, i) => (
          <tr key={i} style={{ background: i % 2 === 0 ? '#f9fafb' : '#fff' }}>
            <td style={{ ...tdStyle, textAlign: 'left' }}>{ind.name}</td>
            <td style={tdStyle}>{ind.score.toFixed(1)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function IssueSection({ issues, isEn, title }: { issues: Issue[]; isEn: boolean; title: string }) {
  if (!issues || issues.length === 0) return <p style={{ color: '#666' }}>{isEn ? 'No issues found.' : '無問題。'}</p>
  return (
    <div style={{ marginBottom: 20 }}>
      {title && <h3 style={{ fontSize: 14, marginBottom: 10 }}>{title}</h3>}
      {issues.map((issue, idx) => (
        <div key={idx} style={{ border: '1px solid #e5e7eb', borderRadius: 8, marginBottom: 12, overflow: 'hidden' }}>
          {/* Issue header */}
          <div style={{ padding: '10px 14px', background: '#f9fafb', borderBottom: '1px solid #e5e7eb', display: 'flex', alignItems: 'center', gap: 10 }}>
            <span style={{ fontSize: 10, fontWeight: 600, padding: '2px 7px', borderRadius: 4, background: issue.severity === 'High' ? '#fef2f2' : issue.severity === 'Medium' ? '#fffbeb' : '#f0fdf4', color: issue.severity === 'High' ? '#dc2626' : issue.severity === 'Medium' ? '#d97706' : '#16a34a' }}>
              {issue.severity}
            </span>
            <span style={{ fontSize: 13, fontWeight: 600 }}>{isEn ? (issue.title_en || issue.title) : issue.title}</span>
            <span style={{ marginLeft: 'auto', fontSize: 11, color: '#666' }}>{issue.affected_rows} {isEn ? 'rows' : '列'}</span>
          </div>
          {/* Description */}
          <div style={{ padding: '8px 14px', fontSize: 12, color: '#555' }}>
            {isEn && issue.description_en ? issue.description_en : issue.description}
          </div>
          {/* Examples (Excel-style table) */}
          {issue.examples && issue.examples.length > 0 && (
            <div style={{ padding: '0 14px 12px' }}>
              <ExcelTable examples={issue.examples} />
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

function ExcelTable({ examples }: { examples: IssueExample[] }) {
  // Group by label
  const groups: Record<string, IssueExample[]> = {}
  for (const ex of examples) {
    const key = ex.label || '__default'
    if (!groups[key]) groups[key] = []
    groups[key].push(ex)
  }

  return (
    <>
      {Object.entries(groups).slice(0, 3).map(([label, items]) => (
        <div key={label} style={{ marginBottom: 8 }}>
          {label !== '__default' && <div style={{ fontSize: 11, fontWeight: 600, color: '#2b6cb0', marginBottom: 4 }}>{label}</div>}
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11 }}>
            <thead>
              <tr>
                <th style={{ ...cellStyle, background: '#f3f4f6', width: 30, fontWeight: 600 }}>#</th>
                {(items[0]?.headers || []).map((h, i) => (
                  <th key={i} style={{ ...cellStyle, background: '#f3f4f6', fontWeight: 600 }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {items.slice(0, 5).map((ex, rowIdx) => (
                <tr key={rowIdx}>
                  <td style={{ ...cellStyle, color: '#999', fontSize: 10 }}>{ex.row_number}</td>
                  {ex.cells.map((cell, colIdx) => {
                    const isHighlighted = ex.highlights?.includes(colIdx)
                    return (
                      <td key={colIdx} style={{
                        ...cellStyle,
                        background: isHighlighted ? 'rgba(220,38,38,0.06)' : undefined,
                        border: isHighlighted ? '1.5px solid #dc2626' : '1px solid #e5e7eb',
                        color: isHighlighted ? '#dc2626' : undefined,
                      }}>
                        {cell || '—'}
                      </td>
                    )
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ))}
    </>
  )
}

const cellStyle: React.CSSProperties = { padding: '4px 8px', border: '1px solid #e5e7eb', fontSize: 11, maxWidth: 150, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }
