import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../contexts/AuthContext'
import apiClient from '../api/client'

export default function LoginPage() {
  const { t } = useTranslation()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { login } = useAuth()
  const navigate = useNavigate()

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const res = await apiClient.post('/auth/login', { username, password })
      setError('')
      await login(res.data.token)
      navigate('/landing')
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: { message?: string } } } }
      setError(axiosErr.response?.data?.error?.message || t('error.login_failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: 'var(--canvas)', padding: 24,
    }}>
      <div style={{
        background: 'var(--paper)', border: '1px solid var(--line)',
        borderRadius: 14, padding: '40px 36px', width: '100%', maxWidth: 400,
      }}>
        <div style={{ textAlign: 'center', marginBottom: 28 }}>
          <div style={{
            width: 44, height: 44, borderRadius: 10, background: 'var(--ink)', color: '#fff',
            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
            fontWeight: 700, fontSize: 19, fontFamily: 'var(--mono)', marginBottom: 14,
          }}>S</div>
          <h1 style={{ fontSize: 20, fontWeight: 650, marginBottom: 4 }}>{t('page.login.title')}</h1>
          <p style={{ fontSize: 13.5, color: 'var(--ink-soft)' }}>
            {t('page.login.desc')}
          </p>
        </div>

        {error && (
          <div style={{
            background: 'var(--rose-soft)', color: 'var(--rose)',
            padding: '10px 14px', borderRadius: 'var(--radius-sm)',
            fontSize: 13, fontWeight: 500, marginBottom: 16,
          }}>
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', fontSize: 13, fontWeight: 550, marginBottom: 6 }}>
              {t('form.email')}
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={t('form.email_placeholder')}
              required
            />
          </div>
          <div style={{ marginBottom: 24 }}>
            <label style={{ display: 'block', fontSize: 13, fontWeight: 550, marginBottom: 6 }}>
              {t('form.password')}
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t('form.password_placeholder')}
              required
            />
          </div>
          <button
            type="submit"
            className="btn btn-primary"
            disabled={loading}
            style={{ width: '100%', justifyContent: 'center', padding: '12px 18px' }}
          >
            {loading ? t('common.login_progress') : t('btn.login')}
          </button>
        </form>

        <p style={{ textAlign: 'center', marginTop: 20, fontSize: 13.5, color: 'var(--ink-soft)' }}>
          {t('form.no_account')}<Link to="/register">{t('btn.go_register')}</Link>
        </p>
      </div>
    </div>
  )
}
