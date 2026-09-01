import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

const navItems = [
  { icon: DashboardIcon, label: 'Dashboard', path: '/' },
  { icon: PlaybookIcon, label: 'Inventory', path: '/inventory' },
  { icon: HostIcon, label: 'Hosts', path: '/hosts' },
  { icon: BaselineIcon, label: 'Baselines', path: '/baselines' },
  { icon: DeviationIcon, label: 'Deviations', path: '/deviations' },
  { icon: IncidentIcon, label: 'Incidents', path: '/incidents' },
  { icon: HistoryIcon, label: 'History', path: '/history' },
  { icon: ScheduleIcon, label: 'Schedules', path: '/schedules' },
]

export function AppSidebar() {
  const location = useLocation()
  const { user, logout } = useAuth()

  const displayName = user?.info?.name || user?.username || 'User'
  const displayRole = user?.info?.role || ''
  const initials = displayName.slice(0, 2).toUpperCase()

  return (
    <div className="w-52 bg-background border-r border-border flex flex-col py-4 flex-shrink-0">
      <Link
        to="/"
        className="h-10 rounded-xl flex items-center gap-3 mx-4 mb-6 cursor-pointer shadow-lg px-3"
        style={{
          background: 'linear-gradient(135deg, #facc15, #ca8a04)',
          boxShadow: '0 0 16px rgba(250,204,21,0.25)',
        }}
      >
        <span className="text-lg font-bold text-[#18181b]">U</span>
        <span className="text-sm font-bold text-[#18181b]">Ulas</span>
      </Link>

      <div className="flex flex-col gap-1 px-3">
        {navItems.map((item) => {
          const isActive = location.pathname === item.path
          return (
            <Link
              key={item.label}
              to={item.path}
              className={`flex items-center gap-3 px-3 py-2.5 rounded-xl cursor-pointer transition-all duration-200 ${
                isActive
                  ? 'bg-primary text-[#18181b]'
                  : 'text-muted-foreground hover:bg-secondary hover:text-primary'
              }`}
            >
              <item.icon />
              <span className="text-sm font-medium">{item.label}</span>
            </Link>
          )
        })}
      </div>

      <div className="mt-auto flex flex-col gap-1 px-3">
        <button
          type="button"
          onClick={logout}
          className="flex items-center gap-3 px-3 py-2.5 rounded-xl cursor-pointer text-muted-foreground hover:bg-secondary hover:text-primary transition-all duration-200 w-full text-left"
        >
          <LogoutIcon />
          <span className="text-sm font-medium">Logout</span>
        </button>
        <div
          className="flex items-center gap-3 mx-3 mt-2 px-3 py-2 rounded-lg text-xs font-bold text-[#18181b]"
          style={{ background: 'linear-gradient(135deg, #facc15, #ca8a04)' }}
        >
          <div className="w-6 h-6 rounded-full bg-[#18181b]/10 flex items-center justify-center">{initials}</div>
          <div className="min-w-0">
            <div className="truncate">{displayName}</div>
            {displayRole && <div className="truncate text-[10px] opacity-80 font-normal">{displayRole}</div>}
          </div>
        </div>
      </div>
    </div>
  )
}

function DashboardIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect width="7" height="9" x="3" y="3" rx="1" /><rect width="7" height="5" x="14" y="3" rx="1" />
      <rect width="7" height="9" x="14" y="12" rx="1" /><rect width="7" height="5" x="3" y="16" rx="1" />
    </svg>
  )
}
function PlaybookIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect width="20" height="14" x="2" y="3" rx="2" /><line x1="8" x2="8" y1="21" y2="3" /><line x1="16" x2="16" y1="21" y2="3" />
    </svg>
  )
}
function HostIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect width="20" height="14" x="2" y="3" rx="2" /><line x1="8" x2="16" y1="21" y2="21" /><line x1="12" x2="12" y1="17" y2="21" />
    </svg>
  )
}
function BaselineIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" /><polyline points="14 2 14 8 20 8" /><line x1="12" x2="12" y1="18" y2="12" /><line x1="9" x2="15" y1="15" y2="15" />
    </svg>
  )
}
function DeviationIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z" /><line x1="12" x2="12" y1="9" y2="13" /><line x1="12" x2="12.01" y1="17" y2="17" />
    </svg>
  )
}
function IncidentIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="m10.29 3.86 1 2.82a2 2 0 0 0 1.42 1.39l2.83.95a2 2 0 0 1 1.07 3.29l-2.2 2.2a2 2 0 0 0-.57 1.68l.67 2.85a2 2 0 0 1-2.92 2.23l-2.83-1.41a2 2 0 0 0-1.79 0l-2.83 1.41a2 2 0 0 1-2.92-2.23l.67-2.85a2 2 0 0 0-.57-1.68l-2.2-2.2a2 2 0 0 1 1.07-3.29l2.83-.95a2 2 0 0 0 1.42-1.39l1-2.82a2 2 0 0 1 3.74 0z" />
    </svg>
  )
}
function HistoryIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" /><polyline points="12 6 12 12 16 14" />
    </svg>
  )
}
function ScheduleIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" /><polyline points="12 6 12 12 16 14" /><path d="M16 2v4" /><path d="M8 2v4" />
    </svg>
  )
}
function LogoutIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
      <polyline points="16 17 21 12 16 7" />
      <line x1="21" x2="9" y1="12" y2="12" />
    </svg>
  )
}
