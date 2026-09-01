import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import api from '../api/client'

interface Incident {
  id: string
  file_type: 'passwd' | 'group'
  entry_key: string
  actual_value: string
  expected_value?: string
}

interface AllowedDeviation {
  file_type: string
  entry_key: string
  expected_value?: string
  actual_value: string
}

interface HostResult {
  id: string
  scan_job_id: string
  host_id: string
  hostname: string
  status: string
  deviations_found: number
  allowed_deviations: AllowedDeviation[]
  incidents: Incident[]
}

interface ScanJob {
  id: string
  ansible_job_id?: string
  status: string
  callbacks_received: number
  successful_hosts: number
  failed_hosts: number
  created_at: string
}

interface ScanDetailData {
  job: ScanJob
  results: HostResult[]
  total: number
  page: number
  limit: number
}

interface DataTableProps {
  selectedHost?: string
  onSelectHost?: (host: string) => void
}

const DEFAULT_LIMIT = 20

export function DataTable({ selectedHost, onSelectHost }: DataTableProps) {
  const [detail, setDetail] = useState<ScanDetailData | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [page, setPage] = useState(1)
  const [limit, setLimit] = useState(DEFAULT_LIMIT)
  const [error, setError] = useState('')

  useEffect(() => {
    setPage(1)
  }, [searchTerm, limit])

  useEffect(() => {
    let cancelled = false

    api.get('/scans')
      .then((res) => {
        const scans: ScanJob[] = res.data || []
        if (scans.length === 0) return
        const latest = scans[0]
        return api.get(`/scans/${latest.id}?page=${page}&limit=${limit}&include_incidents=true`)
      })
      .then((res) => {
        if (cancelled || !res) return
        const data: ScanDetailData = res.data
        const results = (data.results || []).map((r) => ({
          ...r,
          incidents: r.incidents || [],
          allowed_deviations: r.allowed_deviations || [],
        }))
        setDetail({ ...data, results })

        if (results.length > 0) {
          const stillVisible = selectedHost && results.some((r) => r.hostname === selectedHost)
          if (!stillVisible) {
            onSelectHost?.(results[0].hostname)
          }
        }
      })
      .catch((err) => setError(err.message))

    return () => {
      cancelled = true
    }
  }, [page, limit])

  const rows = (detail?.results || []).map((r) => {
    const passwdIncidents = r.incidents.filter((i) => i.file_type === 'passwd').length
    const groupIncidents = r.incidents.filter((i) => i.file_type === 'group').length
    const passwdAllowed = r.allowed_deviations.filter((d) => d.file_type === 'passwd').length
    const groupAllowed = r.allowed_deviations.filter((d) => d.file_type === 'group').length
    let status: 'deviation' | 'allowed' | 'clean' = 'clean'
    if (r.status === 'deviation_found') status = 'deviation'
    else if (r.status === 'allowed_deviation') status = 'allowed'
    else if (r.status === 'failed') status = 'deviation'
    const deviation = r.incidents.length
      ? r.incidents.map((i) => `${i.file_type}:${i.entry_key}`).join(', ')
      : '—'
    return {
      id: r.id,
      hostId: r.host_id,
      host: r.hostname,
      passwd: passwdIncidents > 0 ? 'mismatch' : passwdAllowed > 0 ? 'allowed' : 'match',
      group: groupIncidents > 0 ? 'mismatch' : groupAllowed > 0 ? 'allowed' : 'match',
      status,
      deviation,
      allowedCount: r.allowed_deviations.length,
    }
  })

  const filteredRows = rows.filter(
    (row) =>
      row.host.toLowerCase().includes(searchTerm.toLowerCase()) ||
      row.deviation.toLowerCase().includes(searchTerm.toLowerCase()) ||
      row.status.toLowerCase().includes(searchTerm.toLowerCase())
  )

  const job = detail?.job
  const total = detail?.total ?? 0
  const currentPage = detail?.page ?? page
  const currentLimit = detail?.limit ?? limit
  const start = total === 0 ? 0 : (currentPage - 1) * currentLimit + 1
  const end = Math.min(currentPage * currentLimit, total)
  const hasNext = currentPage * currentLimit < total
  const hasPrev = currentPage > 1

  return (
    <div className="bg-card border border-border rounded-xl overflow-hidden">
      <div className="px-5 py-4 border-b border-border">
        <div className="flex justify-between items-start mb-4">
          <div>
            <div className="font-semibold text-sm text-foreground">Scan Results</div>
            <div className="text-xs text-muted-foreground mt-0.5">
              {job
                ? `Job ${job.id.slice(0, 8)} • ${job.callbacks_received} callbacks • ${job.status}`
                : 'No scans available'}
            </div>
          </div>
          <div className="flex gap-2">
            <Link
              to="/scans"
              className="px-3 py-1.5 bg-secondary border border-[#3f3f46] rounded-md text-xs text-foreground hover:bg-[#3f3f46] transition-colors"
            >
              All Scans
            </Link>
            <Link
              to="/scans"
              className="px-3 py-1.5 bg-primary text-primary-foreground rounded-md text-xs font-semibold hover:shadow-lg hover:shadow-primary/20 transition-all"
            >
              ▶ Run Scan
            </Link>
          </div>
        </div>
        <div className="relative">
          <svg className="absolute left-3 top-3 w-4 h-4 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
          </svg>
          <input
            type="text"
            placeholder="Search by hostname, status, or deviation..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 bg-background border border-border rounded-lg text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-all"
          />
        </div>
      </div>

      {error && <div className="px-5 py-3 text-xs text-red-500">{error}</div>}

      <div className="overflow-x-auto max-h-[60vh]">
        <table className="w-full text-sm">
          <thead className="sticky top-0 z-10">
            <tr>
              {['Host', '/etc/passwd', '/etc/group', 'Status', 'Deviation', 'Allowed', ''].map((h) => (
                <th key={h} className="px-4 py-3 text-left text-xs font-medium text-muted-foreground border-b border-border bg-background">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {filteredRows.length > 0 ? (
              filteredRows.map((row) => (
                <tr
                  key={row.host}
                  onClick={() => onSelectHost?.(row.host)}
                  className={`transition-colors duration-150 cursor-pointer border-b border-border/50 ${
                    selectedHost === row.host ? 'bg-primary/10 hover:bg-primary/15' : 'hover:bg-primary/[0.03]'
                  }`}
                >
                  <td className="px-4 py-3 text-foreground font-medium font-mono text-xs">
                    {row.host}
                    <div className="text-[10px] text-muted-foreground mt-0.5">
                      <Link
                        to={`/snapshots/${row.hostId}/passwd`}
                        className="hover:text-primary hover:underline mr-2"
                        onClick={(e) => e.stopPropagation()}
                      >
                        passwd
                      </Link>
                      <Link
                        to={`/snapshots/${row.hostId}/group`}
                        className="hover:text-primary hover:underline"
                        onClick={(e) => e.stopPropagation()}
                      >
                        group
                      </Link>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-center">
                    <FileBadge type={row.passwd as 'match' | 'mismatch'} />
                  </td>
                  <td className="px-4 py-3 text-center">
                    <FileBadge type={row.group as 'match' | 'mismatch'} />
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={row.status} />
                  </td>
                  <td className={`px-4 py-3 text-xs font-mono ${row.status === 'deviation' ? 'text-red-500' : row.status === 'allowed' ? 'text-primary' : 'text-muted-foreground'}`}>
                    {row.deviation}
                  </td>
                  <td className="px-4 py-3 text-xs text-muted-foreground">
                    {row.allowedCount > 0 ? row.allowedCount : '—'}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button className="text-muted-foreground hover:text-primary hover:bg-primary/10 p-1 rounded transition-all">
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/></svg>
                    </button>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-muted-foreground text-sm">
                  {searchTerm ? `No hosts found matching "${searchTerm}"` : 'No scan results available'}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <div className="px-5 py-3 border-t border-border flex justify-between items-center">
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <span>
            Showing {start}-{end} of {total} results
          </span>
          <label className="flex items-center gap-2">
            Page size
            <select
              value={limit}
              onChange={(e) => setLimit(Number(e.target.value))}
              className="bg-background border border-border rounded px-2 py-1 text-xs text-foreground focus:outline-none focus:border-primary"
            >
              {[10, 20, 50, 100].map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </label>
        </div>
        <div className="flex gap-1.5 items-center">
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={!hasPrev}
            className="px-3 py-1.5 bg-secondary border border-[#3f3f46] rounded-md text-xs text-muted-foreground hover:bg-[#3f3f46] hover:text-foreground transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Previous
          </button>
          <span className="px-3 py-1.5 bg-primary text-primary-foreground rounded-md text-xs font-semibold">
            {currentPage}
          </span>
          <button
            onClick={() => setPage((p) => p + 1)}
            disabled={!hasNext}
            className="px-3 py-1.5 bg-secondary border border-[#3f3f46] rounded-md text-xs text-muted-foreground hover:bg-[#3f3f46] hover:text-foreground transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  )
}

function FileBadge({ type }: { type: 'match' | 'mismatch' | 'allowed' }) {
  const config = {
    match: { bg: 'bg-green-500/10', text: 'text-green-500', border: 'border-green-500/15', dot: '#22c55e', label: 'Match' },
    mismatch: { bg: 'bg-red-500/10', text: 'text-red-500', border: 'border-red-500/15', dot: '#ef4444', label: 'Mismatch' },
    allowed: { bg: 'bg-primary/10', text: 'text-primary', border: 'border-primary/15', dot: '#facc15', label: 'Allowed' },
  }
  const c = config[type]
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border ${c.bg} ${c.text} ${c.border}`}>
      <span className="w-1.5 h-1.5 rounded-full" style={{ background: c.dot }} />
      {c.label}
    </span>
  )
}

function StatusBadge({ status }: { status: 'deviation' | 'allowed' | 'clean' }) {
  const styles = {
    deviation: 'bg-red-500/10 text-red-500 border-red-500/15',
    allowed: 'bg-primary/10 text-primary border-primary/15',
    clean: 'bg-green-500/10 text-green-500 border-green-500/15',
  }
  const labels = { deviation: '⚠️ Deviation', allowed: '✓ Allowed', clean: '✓ Clean' }
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border ${styles[status]}`}>
      {labels[status]}
    </span>
  )
}
