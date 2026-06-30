import { Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import ProtectedRoute from './components/ProtectedRoute'
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

function App() {
  return (
    <AuthProvider>
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
      </Routes>
    </AuthProvider>
  )
}

export default App
