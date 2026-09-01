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
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [jobIdSearch, setJobIdSearch] = useState('')
  const [fromDate, setFromDate] = useState('')
  const [toDate, setToDate] = useState('')

  const toRFC3339 = (date: string, endOfDay = false) => {
    if (!date) return ''
    if (endOfDay) {
      return new Date(`${date}T23:59:59.999Z`).toISOString()
    }
    return new Date(`${date}T00:00:00.000Z`).toISOString()
  }

  const fetchScans = (
    nextPage = page,
    nextSize = pageSize,
    nextJobId = jobIdSearch,
    nextFrom = fromDate,
    nextTo = toDate
  ) => {
    const params = new URLSearchParams()
    params.set('page', String(nextPage))
    params.set('limit', String(nextSize))
    if (nextJobId.trim()) {
      params.set('search', nextJobId.trim())
    }
    if (nextFrom) {
      params.set('from_date', toRFC3339(nextFrom))
    }
    if (nextTo) {
      params.set('to_date', toRFC3339(nextTo, true))
    }
    api
      .get(`/scans?${params.toString()}`)
      .then((res) => {
        const data = res.data
        if (Array.isArray(data)) {
          setScans(data)
          setTotal(data.length)
        } else {
          setScans(data?.items || [])
          setTotal(data?.total || 0)
        }
      })
      .catch((err) => setError(err.message))
  }

  useEffect(() => {
    fetchScans()
  }, [])

  useEffect(() => {
    fetchScans(page, pageSize)
  }, [page, pageSize])

  useEffect(() => {
    setPage(1)
    fetchScans(1, pageSize, jobIdSearch, fromDate, toDate)
  }, [jobIdSearch, fromDate, toDate])

  const completed = scans.filter((s) => s.status === 'completed').length
  const failed = scans.filter((s) => s.status === 'failed').length

  return (
    <div className="flex flex-col gap-6">
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-card border border-border rounded-xl p-5 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(250,204,21,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Total Scans</div>
          <div className="text-[28px] font-bold text-foreground tracking-tight">{total}</div>
          <div className="text-xs text-muted-foreground">Since first scan</div>
        </div>
        <div className="bg-card border border-border rounded-xl p-5 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(34,197,94,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Successful</div>
          <div className="text-[28px] font-bold text-green-500 tracking-tight">{completed}</div>
          <div className="text-xs text-muted-foreground">On this page</div>
        </div>
        <div className="bg-card border border-border rounded-xl p-5 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(239,68,68,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Failed</div>
          <div className="text-[28px] font-bold text-red-500 tracking-tight">{failed}</div>
          <div className="text-xs text-muted-foreground">On this page</div>
        </div>
      </div>

      {error && <div className="text-xs text-red-500">{error}</div>}

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <div className="font-semibold text-sm text-foreground">Scan History</div>
            <div className="text-xs text-muted-foreground mt-0.5">All scan jobs</div>
          </div>
          <div className="flex flex-col sm:flex-row gap-2">
            <input
              type="text"
              value={jobIdSearch}
              onChange={(e) => setJobIdSearch(e.target.value)}
              placeholder="Search job ID..."
              className="bg-background border border-border rounded-lg px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 w-full sm:w-48"
            />
            <input
              type="date"
              value={fromDate}
              onChange={(e) => setFromDate(e.target.value)}
              className="bg-background border border-border rounded-lg px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
            />
            <input
              type="date"
              value={toDate}
              onChange={(e) => setToDate(e.target.value)}
              className="bg-background border border-border rounded-lg px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
            />
          </div>
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
        <div className="px-5 py-3 border-t border-border flex justify-between items-center">
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">Rows per page</span>
            <select
              value={pageSize}
              onChange={(e) => {
                setPageSize(Number(e.target.value))
                setPage(1)
              }}
              className="bg-background border border-border rounded-lg px-2 py-1 text-xs text-foreground outline-none"
            >
              <option value={5}>5</option>
              <option value={10}>10</option>
              <option value={25}>25</option>
              <option value={50}>50</option>
            </select>
          </div>
          <div className="flex items-center gap-2">
            {(() => {
              const totalPages = Math.max(1, Math.ceil(total / pageSize))
              const safePage = Math.min(page, totalPages)
              return (
                <>
                  <span className="text-xs text-muted-foreground">
                    Page {safePage} of {totalPages}
                  </span>
                  <button
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={safePage <= 1}
                    className="px-3 py-1.5 border border-border rounded-lg text-xs disabled:opacity-50 hover:bg-secondary transition-all"
                  >
                    Previous
                  </button>
                  <button
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={safePage >= totalPages}
                    className="px-3 py-1.5 border border-border rounded-lg text-xs disabled:opacity-50 hover:bg-secondary transition-all"
                  >
                    Next
                  </button>
                </>
              )
            })()}
          </div>
        </div>
      </div>
    </div>
  )
}
