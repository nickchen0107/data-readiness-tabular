import { useState, useEffect } from 'react'
import apiClient from '../api/client'

interface QAResult {
  original_answer: string
  cleaned_answer: string
  guardrail_triggered?: boolean
}

export default function QAPage() {
  const [consent, setConsent] = useState(false)
  const [showConsentModal, setShowConsentModal] = useState(true)
  const [sessionId, setSessionId] = useState('')
  const [assessmentId, setAssessmentId] = useState('')
  const [suggestions, setSuggestions] = useState<string[]>([])
  const [question, setQuestion] = useState('')
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<QAResult | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    apiClient.get('/clean/latest').then((res) => {
      setSessionId(res.data.session_id || res.data.id)
      setAssessmentId(res.data.assessment_id || '')
    }).catch(() => {})
  }, [])

  useEffect(() => {
    if (assessmentId && consent) {
      apiClient.get(`/qa/suggestions/${assessmentId}`).then((res) => {
        setSuggestions(res.data.suggestions || [])
      }).catch(() => {})
    }
  }, [assessmentId, consent])

  const handleConsent = () => {
    setConsent(true)
    setShowConsentModal(false)
  }

  const handleAsk = async (q?: string) => {
    const questionText = q || question
    if (!questionText.trim() || !sessionId) return
    setError('')
    setLoading(true)
    setResult(null)
    try {
      const res = await apiClient.post('/qa/ask', {
        session_id: sessionId,
        question: questionText,
        consent: true,
      })
      setResult(res.data)
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: { message?: string } } } }
      setError(axiosErr.response?.data?.error?.message || '問答請求失敗')
    } finally {
      setLoading(false)
    }
  }

  // Consent modal
  if (showConsentModal && !consent) {
    return (
      <div style={{
        background: 'var(--paper)', border: '1px solid var(--line)',
        borderRadius: 14, overflow: 'hidden',
      }}>
        <div style={{ padding: 28 }}>
          <div style={{ maxWidth: 480, margin: '0 auto', textAlign: 'center', padding: '40px 0' }}>
            <div style={{ fontSize: 40, marginBottom: 16 }}>🤖</div>
            <h2 style={{ fontSize: 20, fontWeight: 650, marginBottom: 12 }}>問答對比功能</h2>
            <p style={{ fontSize: 14, color: 'var(--ink-soft)', lineHeight: 1.7, marginBottom: 24 }}>
              此功能會將您的資料片段傳送至 Google Gemini API 進行分析。
              系統僅傳送必要的欄位資料，不會傳送您的個人資訊。
            </p>
            <div style={{
              background: 'var(--amber-soft)', border: '1px solid #f0dbb8',
              borderRadius: 'var(--radius)', padding: '14px 18px',
              fontSize: 13, color: '#7a4506', textAlign: 'left', marginBottom: 24,
            }}>
              <strong>注意事項：</strong>
              <ul style={{ margin: '8px 0 0 16px', lineHeight: 1.8 }}>
                <li>資料將傳送至外部 AI 服務進行處理</li>
                <li>請確認資料不含機密或敏感資訊</li>
                <li>此同意僅適用於本次工作階段</li>
              </ul>
            </div>
            <div style={{ display: 'flex', gap: 12, justifyContent: 'center' }}>
              <button className="btn btn-ghost" onClick={() => setShowConsentModal(false)}>
                暫時跳過
              </button>
              <button className="btn btn-primary" onClick={handleConsent}>
                我同意，開始使用
              </button>
            </div>
          </div>
        </div>
      </div>
    )
  }

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
        }}>STEP 7</div>
        <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>問答對比</h2>
        <p style={{ color: 'var(--ink-soft)', fontSize: 14, marginTop: 5 }}>
          對比原始資料與梳理後資料的 AI 回答差異
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

        {/* Suggested questions */}
        {suggestions.length > 0 && (
          <div style={{ display: 'flex', gap: 9, flexWrap: 'wrap', marginBottom: 18 }}>
            {suggestions.map((s, i) => (
              <button
                key={i}
                onClick={() => { setQuestion(s); handleAsk(s) }}
                style={{
                  fontSize: 13, border: '1px solid var(--line)', borderRadius: 20,
                  padding: '7px 14px', color: 'var(--ink-soft)', background: 'var(--panel)',
                  transition: 'all 0.15s', cursor: 'pointer',
                }}
              >
                {s}
              </button>
            ))}
          </div>
        )}

        {/* Input */}
        <div style={{ display: 'flex', gap: 10, marginBottom: 20 }}>
          <input
            type="text"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') handleAsk() }}
            placeholder="輸入您的問題..."
            style={{ flex: 1 }}
          />
          <button
            className="btn btn-primary"
            onClick={() => handleAsk()}
            disabled={loading || !question.trim()}
          >
            {loading ? '處理中' : '提問'}
          </button>
        </div>

        {/* Question display */}
        {question && result && (
          <div style={{
            background: 'var(--ink)', color: '#fff', borderRadius: 'var(--radius)',
            padding: '13px 17px', fontSize: 14.5, fontWeight: 500, marginBottom: 16,
            display: 'flex', alignItems: 'center', gap: 9,
          }}>
            <span style={{ fontFamily: 'var(--mono)', fontSize: 11, color: '#9fb8d4', flexShrink: 0 }}>
              Q
            </span>
            {question}
          </div>
        )}

        {/* Loading */}
        {loading && (
          <div style={{ textAlign: 'center', padding: 40, color: 'var(--ink-faint)' }}>
            <div style={{ fontSize: 24, marginBottom: 8 }}>⏳</div>
            AI 分析中，請稍候...
          </div>
        )}

        {/* Comparison panes */}
        {result && (
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
            {/* Original pane */}
            <div style={{ border: '1px solid var(--line)', borderRadius: 12, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
              <div style={{
                padding: '11px 16px', fontSize: 12.5, fontWeight: 650,
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                fontFamily: 'var(--mono)', background: 'var(--rose-soft)', color: 'var(--rose)',
              }}>
                <span>原始資料回答</span>
                <span>RAW</span>
              </div>
              <div style={{ padding: 16, fontSize: 13.5, lineHeight: 1.6, flex: 1 }}>
                {result.guardrail_triggered ? (
                  <div style={{
                    background: 'var(--rose-soft)', borderRadius: 7,
                    padding: '9px 12px', fontSize: 12.5, color: 'var(--rose)', fontWeight: 500,
                  }}>
                    ⚠ 資料不足 — 該欄位遺漏率超過 50%，無法產生有效回答
                  </div>
                ) : (
                  <p>{result.original_answer}</p>
                )}
              </div>
            </div>

            {/* Cleaned pane */}
            <div style={{ border: '1px solid var(--line)', borderRadius: 12, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
              <div style={{
                padding: '11px 16px', fontSize: 12.5, fontWeight: 650,
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                fontFamily: 'var(--mono)', background: 'var(--green-soft)', color: 'var(--green)',
              }}>
                <span>梳理後資料回答</span>
                <span>REFINED</span>
              </div>
              <div style={{ padding: 16, fontSize: 13.5, lineHeight: 1.6, flex: 1 }}>
                {result.guardrail_triggered ? (
                  <div style={{
                    background: 'var(--rose-soft)', borderRadius: 7,
                    padding: '9px 12px', fontSize: 12.5, color: 'var(--rose)', fontWeight: 500,
                  }}>
                    ⚠ 資料不足 — 該欄位遺漏率超過 50%，無法產生有效回答
                  </div>
                ) : (
                  <p>{result.cleaned_answer}</p>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
