import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import apiClient from '../api/client'

export default function RoutingPage() {
  const navigate = useNavigate()
  const [status, setStatus] = useState<string>('')
  const [score, setScore] = useState<number>(0)

  useEffect(() => {
    apiClient.get('/assess/latest').then((res) => {
      setStatus(res.data.status)
      setScore(res.data.total_score)
    }).catch(() => {})
  }, [])

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
        }}>STEP 3</div>
        <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>分流決策</h2>
        <p style={{ color: 'var(--ink-soft)', fontSize: 14, marginTop: 5 }}>
          根據評估結果，請選擇後續操作方式
        </p>
      </div>

      {/* Body */}
      <div style={{ padding: 28 }}>
        {/* Risk warning */}
        {status === 'not_ready' && (
          <div style={{
            display: 'flex', gap: 11, background: 'var(--rose-soft)',
            border: '1px solid #f5c6c2', borderRadius: 'var(--radius)',
            padding: '13px 16px', fontSize: 13, color: '#7a1a12', marginBottom: 20,
          }}>
            <span style={{ fontWeight: 700, fontFamily: 'var(--mono)' }}>⚠</span>
            <span>
              資料品質評分為 <strong>{score.toFixed(1)}</strong>，狀態為「Not Ready」。
              建議補齊資料後重新上傳以獲得更佳梳理效果。
            </span>
          </div>
        )}

        {/* Fork cards */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          {/* Option 1: Clean with current data */}
          <div
            onClick={() => navigate('/cleaning')}
            style={{
              border: '1.5px solid var(--line)', borderRadius: 12,
              padding: 22, cursor: 'pointer', transition: 'all 0.18s',
              background: 'var(--panel)',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--accent)'
              e.currentTarget.style.transform = 'translateY(-2px)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--line)'
              e.currentTarget.style.transform = 'translateY(0)'
            }}
          >
            <div style={{
              width: 40, height: 40, borderRadius: 9,
              background: 'var(--accent-soft)', color: 'var(--accent)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: 20, marginBottom: 13,
            }}>🔧</div>
            <h3 style={{ fontSize: 16, fontWeight: 650, marginBottom: 6 }}>以現況梳理</h3>
            <p style={{ fontSize: 13, color: 'var(--ink-soft)', lineHeight: 1.55 }}>
              使用目前資料直接進行批次梳理，適合品質尚可的資料集。
            </p>
            {(
              <div style={{
                fontFamily: 'var(--mono)', fontSize: 10.5, color: 'var(--green)',
                fontWeight: 600, marginTop: 10,
              }}>
                ✓ 推薦選項
              </div>
            )}
          </div>

          {/* Option 2: Re-upload */}
          <div
            onClick={() => navigate('/upload')}
            style={{
              border: '1.5px solid var(--line)', borderRadius: 12,
              padding: 22, cursor: 'pointer', transition: 'all 0.18s',
              background: 'var(--panel)',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--accent)'
              e.currentTarget.style.transform = 'translateY(-2px)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--line)'
              e.currentTarget.style.transform = 'translateY(0)'
            }}
          >
            <div style={{
              width: 40, height: 40, borderRadius: 9,
              background: 'var(--amber-soft)', color: 'var(--amber)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: 20, marginBottom: 13,
            }}>📤</div>
            <h3 style={{ fontSize: 16, fontWeight: 650, marginBottom: 6 }}>補齊後重新上傳</h3>
            <p style={{ fontSize: 13, color: 'var(--ink-soft)', lineHeight: 1.55 }}>
              修正原始資料中的問題後重新上傳，以獲得更高的品質分數。
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
