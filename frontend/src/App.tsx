import { Routes, Route } from 'react-router-dom'
import { AuthProvider, useAuth } from './context/AuthContext'
import { AppSidebar } from './components/AppSidebar'
import { SiteHeader } from './components/SiteHeader'
import LoginPage from './pages/LoginPage'
import Dashboard from './pages/Dashboard'
import Inventory from './pages/Inventory'
import Deviations from './pages/Deviations'
import History from './pages/History'
import HostsPage from './pages/HostsPage'
import BaselinesPage from './pages/BaselinesPage'
import IncidentsPage from './pages/IncidentsPage'
import ScanHistoryPage from './pages/ScanHistoryPage'
import ScanDetailPage from './pages/ScanDetailPage'
import SnapshotHistoryPage from './pages/SnapshotHistoryPage'
import SchedulesPage from './pages/SchedulesPage'

function AppContent() {
  const { user, loading } = useAuth()

  if (loading) {
    return (
      <div className="flex h-screen w-full items-center justify-center bg-background text-foreground">
        <div className="flex items-center gap-3">
          <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          <span className="text-sm text-muted-foreground">Loading...</span>
        </div>
      </div>
    )
  }

  if (!user) {
    return <LoginPage />
  }

  return (
    <div className="flex h-screen min-h-[700px] bg-background text-foreground overflow-hidden">
      <AppSidebar />
      <div className="flex-1 flex flex-col overflow-hidden">
        <SiteHeader />
        <div className="flex-1 overflow-y-auto p-6">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/inventory" element={<Inventory />} />
            <Route path="/deviations" element={<Deviations />} />
            <Route path="/history" element={<History />} />
            <Route path="/hosts" element={<HostsPage />} />
            <Route path="/baselines" element={<BaselinesPage />} />
            <Route path="/incidents" element={<IncidentsPage />} />
            <Route path="/scans" element={<ScanHistoryPage />} />
            <Route path="/scans/:id" element={<ScanDetailPage />} />
            <Route path="/snapshots/:hostId/:fileType" element={<SnapshotHistoryPage />} />
            <Route path="/schedules" element={<SchedulesPage />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}

function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  )
}

export default App
