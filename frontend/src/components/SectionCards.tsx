import { useEffect, useState } from 'react'
import api from '../api/client'

interface Host { id: string }
interface Deviation { id: string; is_active: boolean }
interface ScanJob { id: string; status: string }

export function SectionCards() {
  const [counts, setCounts] = useState({
    hosts: 0,
    deviations: 0,
    allowed: 0,
    scans: 0,
  })

  useEffect(() => {
    Promise.all([
      api.get('/hosts'),
      api.get('/deviations'),
      api.get('/scans'),
    ])
      .then(([hostsRes, deviationsRes, scansRes]) => {
        const hosts: Host[] = hostsRes.data || []
        const deviations: Deviation[] = deviationsRes.data || []
        const scans: ScanJob[] = scansRes.data || []
        setCounts({
          hosts: hosts.length,
          deviations: deviations.filter((d) => d.is_active !== false).length,
          allowed: deviations.filter((d) => d.is_active !== false).length,
          scans: scans.length,
        })
      })
      .catch((err) => console.error('Failed to load section cards', err))
  }, [])

  const cards = [
    { label: 'Inventory Hosts', value: counts.hosts.toString(), trend: '', trendColor: 'text-green-500', sub: 'registered hosts', accent: 'rgba(250,204,21,0.06)' },
    { label: 'Deviations Found', value: counts.deviations.toString(), trend: '', trendColor: 'text-red-500', sub: 'active exceptions', accent: 'rgba(239,68,68,0.06)' },
    { label: 'Completed Scans', value: counts.scans.toString(), trend: '', trendColor: 'text-green-500', sub: 'total jobs', accent: 'rgba(34,197,94,0.06)' },
  ]

  return (
    <div className="grid grid-cols-3 gap-4">
      {cards.map((card) => (
        <div
          key={card.label}
          className="bg-card border border-border rounded-xl p-5 flex flex-col gap-2 relative overflow-hidden cursor-pointer transition-all duration-200 hover:border-[#3f3f46] hover:-translate-y-0.5"
        >
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: `radial-gradient(circle at top right, ${card.accent}, transparent 70%)` }} />
          <div className="text-xs text-muted-foreground font-medium">{card.label}</div>
          <div className="text-[28px] font-bold text-foreground tracking-tight">{card.value}</div>
          <div className="text-xs text-muted-foreground flex items-center gap-1">
            {card.trend && <span className={`font-semibold ${card.trendColor}`}>{card.trend}</span>} {card.sub}
          </div>
        </div>
      ))}
    </div>
  )
}
