import { Navigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import type { ReactNode } from 'react'

interface AdminRouteProps {
  children: ReactNode
}

export default function AdminRoute({ children }: AdminRouteProps) {
  const { user } = useAuth()

  if (!user || user.role !== 'admin') {
    return <Navigate to="/" replace />
  }

  return <>{children}</>
}
