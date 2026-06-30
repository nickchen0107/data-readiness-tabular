import { useNavigate } from 'react-router-dom'

export default function DashboardLandingPage() {
  const navigate = useNavigate()

  return (
    <div style={{
      background: 'var(--paper)', border: '1px solid var(--line)',
      borderRadius: 14, overflow: 'hidden', minHeight: 440,
      display: 'flex', flexDirection: 'column',
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
          }}>LANDING</div>
          <h2 style={{ fontSize: 21, fontWeight: 650, letterSpacing: '-0.015em' }}>
            先了解你的 AI 資料現況
          </h2>
          <p style={{ color: 'var(--ink-soft)', fontSize: 14, marginTop: 5, maxWidth: 560 }}>
            想全面評估 AI 資訊安全，可先填寫 AICM 資安成熟度問卷（獨立平台）。或直接試用下方工具。
          </p>
        </div>
        <div style={{
          fontFamily: 'var(--mono)', fontSize: 10.5, color: 'var(--ink-faint)',
          border: '1px dashed var(--line)', borderRadius: 6, padding: '5px 9px',
          whiteSpace: 'nowrap', background: 'var(--panel)', lineHeight: 1.5, textAlign: 'left',
        }}>
          <b style={{ color: 'var(--ink-soft)', fontWeight: 600 }}>邊界</b><br />
          AICM 屬獨立平台<br />
          本工具僅放一行提示
        </div>
      </div>

      {/* Stage body */}
      <div style={{ padding: 28, flex: 1 }}>
        {/* AICM card */}
        <div style={{
          display: 'flex', gap: 16, alignItems: 'flex-start',
          border: '1px dashed var(--line)', borderRadius: 'var(--radius)', padding: '16px 18px',
        }}>
          <div style={{ fontSize: 22 }}>🛡️</div>
          <div>
            <div style={{ fontWeight: 650, marginBottom: 3 }}>AICM AI 資安成熟度問卷</div>
            <div style={{ color: 'var(--ink-faint)', fontSize: 13 }}>
              獨立平台。可先評估貴公司 AI 資安現況，或略過直接試用梳理工具。
            </div>
          </div>
        </div>

        {/* Excel tool card */}
        <div style={{
          marginTop: 16, border: '1px solid var(--accent)', background: 'var(--accent-soft)',
          borderRadius: 'var(--radius)', padding: '16px 18px',
          display: 'flex', gap: 14, alignItems: 'center', cursor: 'pointer',
        }}
          onClick={() => navigate('/upload')}
        >
          <div style={{
            width: 46, height: 46, borderRadius: 10, background: 'var(--accent)', color: '#fff',
            display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 22,
          }}>📊</div>
          <div style={{ flex: 1 }}>
            <div style={{ fontWeight: 650, fontSize: 16 }}>Excel 梳理小工具</div>
            <div style={{ fontSize: 13, color: '#1c4e7a' }}>
              上傳 Excel → 評估 readiness → 梳理 → evidence 存證 → AI 前後對比問答。
            </div>
          </div>
        </div>

        <p style={{ color: 'var(--ink-faint)', fontSize: 13, marginTop: 18, textAlign: 'center' }}>
          點「下一步」或點擊上方「Excel 梳理小工具」開始體驗工具動線。
        </p>
      </div>

      {/* Stage footer */}
      <div style={{
        padding: '16px 28px', borderTop: '1px solid var(--line-soft)',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        background: 'var(--panel)',
      }}>
        <div />
        <div style={{ fontSize: 12.5, color: 'var(--ink-faint)' }}>1 / 8</div>
        <button className="btn btn-primary" onClick={() => navigate('/upload')}>
          下一步 →
        </button>
      </div>
    </div>
  )
}
