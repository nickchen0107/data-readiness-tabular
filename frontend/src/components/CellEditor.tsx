import { useState } from 'react'
import { FlaggedCell, CellEditAction, InteractiveFixResponse, submitInteractiveEdits } from '../api/client'

interface CellEditorProps {
  assessmentId: string
  flaggedCells: FlaggedCell[]
  onComplete: (result: InteractiveFixResponse) => void
}

interface CellDecision {
  action: CellEditAction['action']
  value: string
}

const ISSUE_TYPE_LABELS: Record<string, string> = {
  cell_reference_placeholder: '儲存格引用佔位符',
  column_type_mismatch: '欄位型別不一致',
  inline_remark: '行內備註',
  empty_header: '空白標題欄',
}

export default function CellEditor({ assessmentId, flaggedCells, onComplete }: CellEditorProps) {
  // Group flagged cells by issue_type
  const grouped = flaggedCells.reduce<Record<string, FlaggedCell[]>>((acc, cell) => {
    if (!acc[cell.issue_type]) acc[cell.issue_type] = []
    acc[cell.issue_type].push(cell)
    return acc
  }, {})

  // Track decisions per cell (keyed by "row_index:col_index")
  const [decisions, setDecisions] = useState<Record<string, CellDecision>>(() => {
    const init: Record<string, CellDecision> = {}
    for (const cell of flaggedCells) {
      const key = `${cell.row_index}:${cell.col_index}`
      init[key] = { action: 'keep', value: '' }
    }
    return init
  })

  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<InteractiveFixResponse | null>(null)

  const updateDecision = (key: string, action: CellEditAction['action'], value?: string) => {
    setDecisions(prev => ({
      ...prev,
      [key]: { action, value: value !== undefined ? value : prev[key]?.value || '' },
    }))
  }

  const handleSubmit = async () => {
    setError('')
    setSubmitting(true)

    const edits: CellEditAction[] = flaggedCells
      .map(cell => {
        const key = `${cell.row_index}:${cell.col_index}`
        const decision = decisions[key]
        if (!decision || decision.action === 'keep') return null
        const edit: CellEditAction = {
          row_index: cell.row_index,
          col_index: cell.col_index,
          action: decision.action,
        }
        if (decision.action === 'replace' || decision.action === 'header_rename') {
          edit.value = decision.value
        }
        return edit
      })
      .filter((e): e is CellEditAction => e !== null)

    if (edits.length === 0) {
      setError('請至少修改一筆資料（目前所有欄位皆為「保留原值」）')
      setSubmitting(false)
      return
    }

    try {
      const res = await submitInteractiveEdits({
        assessment_id: assessmentId,
        edits,
      })
      setResult(res)
      onComplete(res)
    } catch (err: unknown) {
      const axiosErr = err as { response?: { status?: number; data?: { error?: { message?: string } } } }
      if (axiosErr.response?.status === 404) {
        setError('評估記錄不存在，請重新執行評估')
      } else if (axiosErr.response?.status === 400) {
        setError(axiosErr.response?.data?.error?.message || '請求格式錯誤')
      } else {
        setError('網路錯誤，請重試')
      }
    } finally {
      setSubmitting(false)
    }
  }

  if (result) {
    return (
      <div style={{
        background: 'var(--green-soft)', border: '1px solid #cfe8d8',
        borderRadius: 'var(--radius, 10px)', padding: '18px 20px', marginTop: 20,
      }}>
        <h4 style={{ fontSize: 14, fontWeight: 600, color: 'var(--green)', marginBottom: 8 }}>
          ✓ 互動式修正完成
        </h4>
        <p style={{ fontSize: 13, color: 'var(--ink-soft)', marginBottom: 8 }}>
          影響 {result.rows_affected} 列資料
        </p>
        {result.warnings && result.warnings.length > 0 && (
          <div style={{ marginTop: 10 }}>
            <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--amber)', marginBottom: 4 }}>
              ⚠ 部分修正已跳過：
            </div>
            <ul style={{ margin: 0, paddingLeft: 18, fontSize: 12, color: 'var(--ink-soft)' }}>
              {result.warnings.map((w, i) => (
                <li key={i}>{w}</li>
              ))}
            </ul>
          </div>
        )}
      </div>
    )
  }

  return (
    <div style={{ marginTop: 24 }}>
      <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 14 }}>互動式儲存格修正</h3>
      <p style={{ fontSize: 13, color: 'var(--ink-soft)', marginBottom: 18 }}>
        以下儲存格需要人工判斷修正方式，請逐筆選擇操作後點擊「確認修正」
      </p>

      {error && (
        <div style={{
          background: 'var(--rose-soft)', color: 'var(--rose)',
          padding: '10px 14px', borderRadius: 'var(--radius-sm, 6px)',
          fontSize: 13, fontWeight: 500, marginBottom: 16,
        }}>
          {error}
        </div>
      )}

      {Object.entries(grouped).map(([issueType, cells]) => (
        <div key={issueType} style={{ marginBottom: 20 }}>
          <div style={{
            fontSize: 13, fontWeight: 650, color: 'var(--ink)',
            marginBottom: 10, display: 'flex', alignItems: 'center', gap: 8,
          }}>
            <span style={{
              display: 'inline-block', width: 8, height: 8, borderRadius: '50%',
              background: issueType === 'cell_reference_placeholder' || issueType === 'column_type_mismatch'
                ? 'var(--rose)' : issueType === 'inline_remark' ? 'var(--amber)' : 'var(--accent)',
            }} />
            {ISSUE_TYPE_LABELS[issueType] || issueType}
            <span style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)' }}>
              ({cells.length} 筆)
            </span>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {cells.map(cell => {
              const key = `${cell.row_index}:${cell.col_index}`
              const decision = decisions[key]
              return (
                <CellRow
                  key={key}
                  cell={cell}
                  decision={decision}
                  onUpdate={(action, value) => updateDecision(key, action, value)}
                />
              )
            })}
          </div>
        </div>
      ))}

      <div style={{ marginTop: 20, display: 'flex', alignItems: 'center', gap: 12 }}>
        <button
          className="btn btn-primary"
          onClick={handleSubmit}
          disabled={submitting}
          style={{ minWidth: 120 }}
        >
          {submitting ? '提交中...' : '確認修正'}
        </button>
        {submitting && (
          <span style={{ fontSize: 12, color: 'var(--ink-faint)', fontFamily: 'var(--mono)' }}>
            正在處理修正...
          </span>
        )}
      </div>
    </div>
  )
}

// --- Individual Cell Row ---

function CellRow({
  cell,
  decision,
  onUpdate,
}: {
  cell: FlaggedCell
  decision: CellDecision
  onUpdate: (action: CellEditAction['action'], value?: string) => void
}) {
  const isReplace = decision.action === 'replace'
  const isHeaderRename = decision.action === 'header_rename'
  const showInput = isReplace || isHeaderRename

  return (
    <div style={{
      border: '1px solid var(--line)', borderRadius: 8,
      padding: '12px 14px', background: 'var(--panel)',
    }}>
      {/* Cell info */}
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 14, marginBottom: 10 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
            <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--ink)' }}>
              {cell.column_name}
            </span>
            <span style={{ fontSize: 11, color: 'var(--ink-faint)', fontFamily: 'var(--mono)' }}>
              第 {cell.row_number} 列
            </span>
          </div>
          <div style={{ fontSize: 13, color: 'var(--ink-soft)', fontFamily: 'var(--mono)' }}>
            {cell.current_value || <span style={{ color: 'var(--ink-faint)' }}>(空值)</span>}
          </div>
          <div style={{ fontSize: 11, color: 'var(--ink-faint)', marginTop: 4 }}>
            {cell.issue_description}
          </div>
        </div>
      </div>

      {/* Action buttons */}
      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', alignItems: 'center' }}>
        {/* 輸入新值 */}
        {cell.issue_type !== 'empty_header' && (
          <ActionButton
            label="輸入新值"
            active={isReplace}
            onClick={() => onUpdate('replace')}
          />
        )}

        {/* 保留原值 */}
        <ActionButton
          label="保留原值"
          active={decision.action === 'keep'}
          onClick={() => onUpdate('keep')}
        />

        {/* 刪除該列 */}
        {cell.issue_type !== 'empty_header' && (
          <ActionButton
            label="刪除該列"
            active={decision.action === 'delete_row'}
            onClick={() => onUpdate('delete_row')}
            danger
          />
        )}

        {/* 分離備註 — only for inline_remark */}
        {cell.issue_type === 'inline_remark' && (
          <ActionButton
            label="分離備註"
            active={decision.action === 'remark_split'}
            onClick={() => onUpdate('remark_split')}
          />
        )}

        {/* 重新命名 — only for empty_header */}
        {cell.issue_type === 'empty_header' && (
          <ActionButton
            label="重新命名"
            active={isHeaderRename}
            onClick={() => onUpdate('header_rename')}
          />
        )}
      </div>

      {/* Text input for replace or header_rename */}
      {showInput && (
        <div style={{ marginTop: 8 }}>
          <input
            type="text"
            placeholder={isHeaderRename ? '輸入新欄位名稱...' : '輸入新值...'}
            value={decision.value}
            onChange={e => onUpdate(decision.action, e.target.value)}
            style={{
              width: '100%', padding: '7px 10px',
              border: '1px solid var(--line)', borderRadius: 6,
              fontSize: 13, fontFamily: 'var(--mono)',
              background: 'var(--paper)', color: 'var(--ink)',
              outline: 'none',
            }}
          />
        </div>
      )}
    </div>
  )
}

// --- Action Button ---

function ActionButton({
  label,
  active,
  onClick,
  danger,
}: {
  label: string
  active: boolean
  onClick: () => void
  danger?: boolean
}) {
  const activeColor = danger ? 'var(--rose)' : 'var(--accent)'
  const activeBg = danger ? 'var(--rose-soft)' : 'var(--accent-soft)'

  return (
    <button
      onClick={onClick}
      style={{
        padding: '5px 10px', borderRadius: 14,
        border: `1.5px solid ${active ? activeColor : 'var(--line)'}`,
        background: active ? activeBg : 'var(--paper)',
        color: active ? activeColor : 'var(--ink-soft)',
        fontSize: 12, fontWeight: 550,
        cursor: 'pointer', transition: 'all 0.15s',
      }}
    >
      {label}
    </button>
  )
}
