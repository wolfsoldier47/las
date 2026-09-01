import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import api from '../api/client'

interface ScanJob {
  id: string
  ansible_job_id?: string
  job_template_id: number
  limit?: string
  status: string
  initiated_by?: string
  callbacks_received: number
  successful_hosts: number
  failed_hosts: number
  created_at: string
}

function StatusBadge({ status }: { status: 'completed' | 'failed' | 'running' }) {
  const styles = {
    completed: 'bg-green-500/10 text-green-500 border-green-500/15',
    failed: 'bg-red-500/10 text-red-500 border-red-500/15',
    running: 'bg-primary/10 text-primary border-primary/15',
  }
  const icon = status === 'completed' ? '✓' : status === 'failed' ? '✗' : '●'
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border ${styles[status]}`}>
      {icon} {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  )
}

export default function History() {
  const [scans, setScans] = useState<ScanJob[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get('/scans')
      .then((res) => setScans(res.data || []))
      .catch((err) => setError(err.message))
  }, [])

  const completed = scans.filter((s) => s.status === 'completed').length
  const failed = scans.filter((s) => s.status === 'failed').length

  return (
    <div className="flex flex-col gap-6">
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-card border border-border rounded-xl p-5 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(250,204,21,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Total Scans</div>
          <div className="text-[28px] font-bold text-foreground tracking-tight">{scans.length}</div>
          <div className="text-xs text-muted-foreground">Since first scan</div>
        </div>
        <div className="bg-card border border-border rounded-xl p-5 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(34,197,94,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Successful</div>
          <div className="text-[28px] font-bold text-green-500 tracking-tight">{completed}</div>
          <div className="text-xs text-muted-foreground">Completed jobs</div>
        </div>
        <div className="bg-card border border-border rounded-xl p-5 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(239,68,68,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Failed</div>
          <div className="text-[28px] font-bold text-red-500 tracking-tight">{failed}</div>
          <div className="text-xs text-muted-foreground">Failed jobs</div>
        </div>
      </div>

      {error && <div className="text-xs text-red-500">{error}</div>}

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <div className="font-semibold text-sm text-foreground">Scan History</div>
          <div className="text-xs text-muted-foreground mt-0.5">All scan jobs</div>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Job ID</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Template ID</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Limit</th>
                <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Hosts</th>
                <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Successful</th>
                <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Failed</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Created</th>
              </tr>
            </thead>
            <tbody>
              {scans.map((s) => (
                <tr key={s.id} className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]">
                  <td className="px-5 py-3 text-foreground font-medium font-mono text-xs">
                    <Link to={`/scans/${s.id}`} className="text-primary hover:underline">
                      #{s.id.slice(0, 8)}
                    </Link>
                  </td>
                  <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{s.job_template_id}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{s.limit || '-'}</td>
                  <td className="px-5 py-3 text-right text-muted-foreground text-xs">{s.callbacks_received}</td>
                  <td className="px-5 py-3 text-right text-green-500 text-xs font-medium">{s.successful_hosts}</td>
                  <td className="px-5 py-3 text-right text-red-500 text-xs font-medium">{s.failed_hosts}</td>
                  <td className="px-5 py-3">
                    <StatusBadge status={s.status as 'completed' | 'failed' | 'running'} />
                  </td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{new Date(s.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
