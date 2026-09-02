import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import api, { downloadScanReport } from '../api/client'

interface BaselineVersion {
  host_id?: string
  hostname?: string
  os_type: string
  file_type: 'passwd' | 'group'
  version: number
  is_active: boolean
  entry_count: number
  description?: string
  created_by?: string
  created_at: string
}

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
  failed_host_names?: string[]
  total_deviations: number
  total_allowed_deviations: number
  created_at: string
}

function StatusBadge({ status }: { status: 'completed' | 'failed' | 'running' | string }) {
  const styles: Record<string, string> = {
    completed: 'bg-green-500/10 text-green-500 border-green-500/15',
    failed: 'bg-red-500/10 text-red-500 border-red-500/15',
    running: 'bg-primary/10 text-primary border-primary/15',
    initiating: 'bg-primary/10 text-primary border-primary/15',
  }
  const icon = status === 'completed' ? '✓' : status === 'failed' ? '✗' : '●'
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border ${styles[status] || styles.running}`}>
      {icon} {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  )
}

interface AAPHealth {
  status: string
  aap_status: string
}

export default function ScanHistoryPage() {
  const [scans, setScans] = useState<ScanJob[]>([])
  const [baselines, setBaselines] = useState<BaselineVersion[]>([])
  const [aapHealth, setAapHealth] = useState<AAPHealth | null>(null)
  const [aapLoading, setAapLoading] = useState(true)
  const [error, setError] = useState('')
  const [form, setForm] = useState({
    limit: '',
    os_type: 'linux',
  })
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [totalScans, setTotalScans] = useState(0)
  const [onlyDeviations, setOnlyDeviations] = useState(false)
  const [search, setSearch] = useState('')

  const fetchScans = (nextPage = page, nextSize = pageSize, nextOnlyDeviations = onlyDeviations, nextSearch = search) => {
    api
      .get(`/scans?page=${nextPage}&limit=${nextSize}&has_deviations=${nextOnlyDeviations}&search=${encodeURIComponent(nextSearch)}`)
      .then((res) => {
        const data = res.data
        if (Array.isArray(data)) {
          setScans(data)
          setTotalScans(data.length)
        } else {
          setScans(data?.items || [])
          setTotalScans(data?.total || 0)
        }
      })
      .catch((err) => setError(err.message))
  }

  const fetchBaselines = () => {
    api
      .get('/baselines/versions')
      .then((res) => setBaselines(res.data || []))
      .catch((err) => setError(err.message))
  }

  const fetchAAPHealth = (osType = form.os_type) => {
    setAapLoading(true)
    api
      .get(`/health/aap?os_type=${osType}`)
      .then((res) => setAapHealth(res.data))
      .catch(() => setAapHealth({ status: 'degraded', aap_status: 'unreachable' }))
      .finally(() => setAapLoading(false))
  }

  useEffect(() => {
    fetchScans()
    fetchBaselines()
    fetchAAPHealth()
    const interval = setInterval(() => fetchAAPHealth(form.os_type), 30000)
    return () => clearInterval(interval)
  }, [form.os_type])

  useEffect(() => {
    fetchScans(page, pageSize)
  }, [page, pageSize, onlyDeviations, search])

  const aapLive = aapHealth?.aap_status === 'ok'

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!aapLive) {
      setError('AAP is not reachable. Cannot start scan.')
      return
    }
    api
      .post('/scans', form)
      .then(() => {
        setForm((prev) => ({ ...prev, limit: '' }))
        setPage(1)
        setOnlyDeviations(false)
        setSearch('')
        fetchScans(1, pageSize, false, '')
      })
      .catch((err) => {
        if (err.response?.status === 409) {
          setError('The scan is already in place')
        } else {
          setError(err.response?.data?.error || err.message)
        }
      })
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <div className="font-semibold text-sm text-foreground">Trigger Scan</div>
          <div className="text-xs text-muted-foreground mt-0.5">Launch an Ansible scan via AAP using the configured job template</div>
        </div>
        <form onSubmit={handleSubmit} className="p-5 grid grid-cols-2 gap-4">
          {error && <div className="col-span-2 text-xs text-red-500">{error}</div>}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">Limit (host pattern)</label>
            <input
              type="text"
              className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={form.limit}
              onChange={(e) => setForm({ ...form, limit: e.target.value })}
              placeholder="e.g., host1,host2 or leave empty for all"
              disabled={!aapLive}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">Target OS</label>
            <select
              className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={form.os_type}
              onChange={(e) => {
                const osType = e.target.value
                setForm({ ...form, os_type: osType })
                fetchAAPHealth(osType)
              }}
            >
              <option value="linux">Linux (RHEL)</option>
              <option value="solaris">Solaris</option>
            </select>
          </div>
          <div className="col-span-2 flex items-center gap-3">
            <button
              type="submit"
              disabled={!aapLive}
              className={`px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
                aapLive
                  ? 'bg-primary text-primary-foreground hover:shadow-lg hover:shadow-primary/20'
                  : 'bg-muted text-muted-foreground cursor-not-allowed'
              }`}
            >
              Start Scan
            </button>
            <AAPStatusBadge health={aapHealth} loading={aapLoading} />
          </div>
        </form>

        <div className="px-5 pb-5">
          <div className="text-xs font-medium text-muted-foreground mb-2">Selected Master Files</div>
          {baselines.filter((b) => b.is_active).length === 0 ? (
            <div className="text-xs text-muted-foreground">No active master files configured.</div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              {baselines
                .filter((b) => b.is_active)
                .map((baseline, idx) => (
                  <div
                    key={idx}
                    className="bg-background border border-border rounded-lg p-3 flex flex-col gap-1"
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-medium text-foreground">
                        {baseline.file_type === 'passwd' ? '/etc/passwd' : '/etc/group'}
                      </span>
                      <span className="text-[10px] px-1.5 py-0.5 bg-green-500/10 text-green-500 border border-green-500/15 rounded">
                        v{baseline.version}
                      </span>
                    </div>
                    <div className="text-[11px] text-muted-foreground">
                      {baseline.os_type} • {baseline.host_id ? baseline.hostname || baseline.host_id : 'Global'}
                    </div>
                    <div className="text-[11px] text-muted-foreground">{baseline.entry_count} entries</div>
                    {baseline.description && (
                      <div className="text-[11px] text-muted-foreground truncate" title={baseline.description}>
                        {baseline.description}
                      </div>
                    )}
                  </div>
                ))}
            </div>
          )}
        </div>
      </div>

      {(() => {
        const totalPages = Math.max(1, Math.ceil(totalScans / pageSize))
        const safePage = Math.min(page, totalPages)

        return (
          <div className="bg-card border border-border rounded-xl overflow-hidden">
            <div className="px-5 py-4 border-b border-border flex justify-between items-center">
              <div>
                <div className="font-semibold text-sm text-foreground">Scan Jobs</div>
                <div className="text-xs text-muted-foreground mt-0.5">{totalScans} jobs</div>
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={search}
                  onChange={(e) => {
                    setSearch(e.target.value)
                    setPage(1)
                  }}
                  placeholder="Search job ID..."
                  className="bg-background border border-border rounded-lg px-3 py-1 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 w-40 sm:w-56"
                />
                <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer">
                  <input
                    type="checkbox"
                    checked={onlyDeviations}
                    onChange={(e) => {
                      setOnlyDeviations(e.target.checked)
                      setPage(1)
                    }}
                    className="rounded border-border bg-background text-primary focus:ring-primary"
                  />
                  Only with deviations
                </label>
                <select
                  value={pageSize}
                  onChange={(e) => {
                    setPageSize(Number(e.target.value))
                    setPage(1)
                  }}
                  className="bg-background border border-border rounded-lg px-2 py-1 text-xs text-foreground outline-none"
                >
                  <option value={5}>5 / page</option>
                  <option value={10}>10 / page</option>
                  <option value={25}>25 / page</option>
                  <option value={50}>50 / page</option>
                </select>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={safePage <= 1}
                    className="px-2 py-1 border border-border rounded-lg text-xs disabled:opacity-50 hover:bg-secondary transition-all"
                  >
                    Prev
                  </button>
                  <span className="text-xs text-muted-foreground px-2">
                    {safePage} / {totalPages}
                  </span>
                  <button
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={safePage >= totalPages}
                    className="px-2 py-1 border border-border rounded-lg text-xs disabled:opacity-50 hover:bg-secondary transition-all"
                  >
                    Next
                  </button>
                </div>
              </div>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Job ID</th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">AAP Job ID</th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Template ID</th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Limit</th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                    <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Success</th>
                    <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Failed</th>
                    <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Deviations</th>
                    <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Allowed</th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Failed Hosts</th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Created</th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Report</th>
                  </tr>
                </thead>
                <tbody>
                  {scans.map((scan) => (
                    <tr key={scan.id} className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]">
                      <td className="px-5 py-3 text-foreground font-medium font-mono text-xs">
                        <Link to={`/scans/${scan.id}`} className="text-primary hover:underline">
                          {scan.id.slice(0, 8)}
                        </Link>
                      </td>
                      <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{scan.ansible_job_id || '-'}</td>
                      <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{scan.job_template_id}</td>
                      <td className="px-5 py-3 text-muted-foreground text-xs">{scan.limit || '-'}</td>
                      <td className="px-5 py-3"><StatusBadge status={scan.status} /></td>
                      <td className="px-5 py-3 text-right text-green-500 text-xs font-medium">{scan.successful_hosts}</td>
                      <td className="px-5 py-3 text-right text-red-500 text-xs font-medium">{scan.failed_hosts}</td>
                      <td className="px-5 py-3 text-right text-red-500 text-xs font-medium">{scan.total_deviations || 0}</td>
                      <td className="px-5 py-3 text-right text-amber-500 text-xs font-medium">{scan.total_allowed_deviations || 0}</td>
                      <td className="px-5 py-3 text-muted-foreground text-xs">
                        {scan.failed_host_names && scan.failed_host_names.length > 0
                          ? scan.failed_host_names.join(', ')
                          : '-'}
                      </td>
                      <td className="px-5 py-3 text-muted-foreground text-xs">{new Date(scan.created_at).toLocaleString()}</td>
                      <td className="px-5 py-3">
                        <button
                          onClick={() =>
                            downloadScanReport(scan.id).catch((err) =>
                              setError(err.response?.data?.error || err.message)
                            )
                          }
                          className="text-xs px-2 py-1 border border-border rounded-lg hover:bg-secondary transition-all"
                        >
                          PDF
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="px-5 py-3 border-t border-border flex justify-end items-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={safePage <= 1}
                className="px-3 py-1.5 border border-border rounded-lg text-xs disabled:opacity-50 hover:bg-secondary transition-all"
              >
                Previous
              </button>
              <span className="text-xs text-muted-foreground">
                Page {safePage} of {totalPages}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={safePage >= totalPages}
                className="px-3 py-1.5 border border-border rounded-lg text-xs disabled:opacity-50 hover:bg-secondary transition-all"
              >
                Next
              </button>
            </div>
          </div>
        )
      })()}
    </div>
  )
}

function AAPStatusBadge({ health, loading }: { health: AAPHealth | null; loading: boolean }) {
  if (loading && !health) {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border bg-muted text-muted-foreground border-border">
        <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-pulse" />
        Checking AAP…
      </span>
    )
  }

  if (health?.aap_status === 'ok') {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border bg-green-500/10 text-green-500 border-green-500/15">
        <span className="w-1.5 h-1.5 rounded-full bg-green-500" />
        AAP Live
      </span>
    )
  }

  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border bg-red-500/10 text-red-500 border-red-500/15">
      <span className="w-1.5 h-1.5 rounded-full bg-red-500" />
      AAP Unreachable
    </span>
  )
}
