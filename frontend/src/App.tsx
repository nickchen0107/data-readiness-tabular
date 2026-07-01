import { Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import { StepperProvider } from './contexts/StepperContext'
import ProtectedRoute from './components/ProtectedRoute'
import AdminRoute from './components/AdminRoute'
import Layout from './components/Layout'
import LandingPage from './pages/LandingPage'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import DashboardLandingPage from './pages/DashboardLandingPage'
import UploadPage from './pages/UploadPage'
import AssessmentPage from './pages/AssessmentPage'
import RoutingPage from './pages/RoutingPage'
import CleaningPage from './pages/CleaningPage'
import ExportPage from './pages/ExportPage'
import EvidencePage from './pages/EvidencePage'
import QAPage from './pages/QAPage'
import SettingsPage from './pages/SettingsPage'
import AdminLayout from './pages/admin/AdminLayout'
import UsersPage from './pages/admin/UsersPage'
import QuotaSettingsPage from './pages/admin/QuotaSettingsPage'
import TranslationsPage from './pages/admin/TranslationsPage'
import AssessmentRecordsPage from './pages/admin/AssessmentRecordsPage'

function App() {
  return (
    <AuthProvider>
      <StepperProvider>
        <Routes>
          <Route path="/" element={<LandingPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route
            element={
              <ProtectedRoute>
                <Layout />
              </ProtectedRoute>
            }
          >
            <Route path="/dashboard" element={<Navigate to="/landing" replace />} />
            <Route path="/landing" element={<DashboardLandingPage />} />
            <Route path="/upload" element={<UploadPage />} />
            <Route path="/assessment" element={<AssessmentPage />} />
            <Route path="/routing" element={<RoutingPage />} />
            <Route path="/cleaning" element={<CleaningPage />} />
            <Route path="/export" element={<ExportPage />} />
            <Route path="/evidence" element={<EvidencePage />} />
            <Route path="/qa" element={<QAPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Route>
          <Route
            path="/admin"
            element={
              <ProtectedRoute>
                <AdminRoute>
                  <AdminLayout />
                </AdminRoute>
              </ProtectedRoute>
            }
          >
            <Route index element={<Navigate to="/admin/users" replace />} />
            <Route path="users" element={<UsersPage />} />
            <Route path="quota" element={<QuotaSettingsPage />} />
            <Route path="translations" element={<TranslationsPage />} />
            <Route path="records" element={<AssessmentRecordsPage />} />
          </Route>
        </Routes>
      </StepperProvider>
    </AuthProvider>
  )
}

export default App
