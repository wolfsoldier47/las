import { Link, useLocation } from 'react-router-dom'

const pageTitles: Record<string, string> = {
  '/': 'Dashboard',
  '/inventory': 'Inventory',
  '/hosts': 'Hosts',
  '/baselines': 'Baselines',
  '/deviations': 'Deviations',
  '/incidents': 'Incidents',
  '/history': 'History',
  '/scans': 'Scan History',
}

export function SiteHeader() {
  const location = useLocation()
  const title = pageTitles[location.pathname] || 'Dashboard'

  return (
    <div className="h-14 border-b border-border flex items-center justify-between px-6 bg-background flex-shrink-0">
      <div className="flex items-center gap-3">
        <div className="text-sm text-muted-foreground flex items-center gap-2">
          <Link to="/" className="hover:text-primary transition-colors">Dashboard</Link>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#52525b" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m9 18 6-6-6-6"/></svg>
          <span className="text-foreground font-medium">{title}</span>
        </div>
      </div>
      <div className="flex items-center gap-3">
        <div className="w-9 h-9 rounded-lg border border-border flex items-center justify-center cursor-pointer text-muted-foreground hover:border-primary hover:text-primary transition-all duration-200 relative">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>
          </svg>
          <div className="absolute top-2 right-2 w-1.5 h-1.5 bg-primary rounded-full border-2 border-background" />
        </div>
      </div>
    </div>
  )
}
