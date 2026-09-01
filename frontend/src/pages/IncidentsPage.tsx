import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import api from '../api/client'

interface Incident {
  id: string
  incident_number: string
  entry_key: string
  expected_value?: string
  actual_value: string
  severity: string
  status: string
  service_now_ticket_opened: boolean
}

interface HostResult {
  id: string
  hostname: string
  status: string
  deviations_found: number
  incidents: Incident[]
}

interface ScanJob {
  id: string
  ansible_job_id?: string
  job_template_id: number
  limit?: string
  status: string
  callbacks_received: number
  successful_hosts: number
  failed_hosts: number
  created_at: string
}

interface PaginatedScanJobs {
  items: ScanJob[]
  total: number
  page: number
  limit: number
}

interface ScanDetailData {
  job: ScanJob
  results: HostResult[]
}

function StatusBadge({ status }: { status: string }) {
  const styles: Record<string, string> = {
    completed: 'bg-green-500/10 text-green-500 border-green-500/15',
    failed: 'bg-red-500/10 text-red-500 border-red-500/15',
    running: 'bg-primary/10 text-primary border-primary/15',
    initiating: 'bg-primary/10 text-primary border-primary/15',
    success: 'bg-green-500/10 text-green-500 border-green-500/15',
    deviation_found: 'bg-red-500/10 text-red-500 border-red-500/15',
    allowed_deviation: 'bg-primary/10 text-primary border-primary/15',
    pending: 'bg-primary/10 text-primary border-primary/15',
  }
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border ${styles[status] || styles.pending}`}>
      {status}
    </span>
  )
}

export default function IncidentsPage() {
  const [scans, setScans] = useState<ScanJob[]>([])
  const [selectedScanId, setSelectedScanId] = useState<string | null>(null)
  const [detail, setDetail] = useState<ScanDetailData | null>(null)
  const [error, setError] = useState('')
  const [ticketError, setTicketError] = useState('')
  const [ticketSuccess, setTicketSuccess] = useState('')

  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [totalScans, setTotalScans] = useState(0)

  const fetchScans = (nextPage = page, nextSize = pageSize, nextSearch = search) => {
    api
      .get(`/scans?page=${nextPage}&limit=${nextSize}&search=${encodeURIComponent(nextSearch)}`)
      .then((res) => {
        const data: PaginatedScanJobs | ScanJob[] = res.data
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

  useEffect(() => {
    fetchScans()
  }, [page, pageSize, search])

  const loadScanDetail = (scanId: string) => {
    setSelectedScanId(scanId)
    setDetail(null)
    setTicketError('')
    setTicketSuccess('')
    api
      .get(`/scans/${scanId}?include_incidents=true`)
      .then((res) => {
        const data: ScanDetailData = res.data
        setDetail({ ...data, results: data.results || [] })
      })
      .catch((err) => setError(err.message))
  }

  const openHostTickets = (hostResult: HostResult) => {
    setTicketError('')
    setTicketSuccess('')
    const incidentIds = hostResult.incidents.filter((i) => !i.service_now_ticket_opened).map((i) => i.id)
    if (incidentIds.length === 0) return
    api
      .post('/incidents/bulk-servicenow', { incident_ids: incidentIds })
      .then(() => {
        setTicketSuccess(`Opened ${incidentIds.length} ServiceNow ticket(s) for ${hostResult.hostname}`)
        if (selectedScanId) loadScanDetail(selectedScanId)
      })
      .catch((err) => setTicketError(err.response?.data?.error || err.message))
  }

  const failingHosts = detail?.results.filter(
    (r) => r.status === 'deviation_found' || r.deviations_found > 0 || r.incidents.length > 0
  ) || []

  const totalPages = Math.max(1, Math.ceil(totalScans / pageSize))

  return (
    <div className="flex flex-col gap-6">
      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
            <div>
              <div className="font-semibold text-sm text-foreground">Scans with Incidents</div>
              <div className="text-xs text-muted-foreground mt-0.5">{totalScans} scan(s) — select one to view hosts that failed compliance</div>
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
                className="bg-background border border-border rounded-lg px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 w-full sm:w-56"
              />
              <select
                value={pageSize}
                onChange={(e) => {
                  setPageSize(Number(e.target.value))
                  setPage(1)
                }}
                className="bg-background border border-border rounded-lg px-2 py-1.5 text-xs text-foreground outline-none"
              >
                <option value={5}>5 / page</option>
                <option value={10}>10 / page</option>
                <option value={25}>25 / page</option>
                <option value={50}>50 / page</option>
              </select>
            </div>
          </div>
        </div>
        {error && <div className="px-5 py-3 text-xs text-red-500">{error}</div>}
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Job ID</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">AAP Job ID</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Limit</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Hosts</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Created</th>
              </tr>
            </thead>
            <tbody>
              {scans.map((scan) => (
                <tr
                  key={scan.id}
                  onClick={() => loadScanDetail(scan.id)}
                  className={`border-b border-border/50 transition-colors cursor-pointer ${
                    selectedScanId === scan.id ? 'bg-primary/10' : 'hover:bg-primary/[0.03]'
                  }`}
                >
                  <td className="px-5 py-3 text-foreground font-medium font-mono text-xs">
                    <Link to={`/scans/${scan.id}`} className="text-primary hover:underline" onClick={(e) => e.stopPropagation()}>
                      {scan.id.slice(0, 8)}
                    </Link>
                  </td>
                  <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{scan.ansible_job_id || '-'}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{scan.limit || '-'}</td>
                  <td className="px-5 py-3"><StatusBadge status={scan.status} /></td>
                  <td className="px-5 py-3 text-right text-muted-foreground text-xs">{scan.callbacks_received}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{new Date(scan.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="px-5 py-3 border-t border-border flex justify-end items-center gap-2">
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
            className="px-3 py-1.5 border border-border rounded-lg text-xs disabled:opacity-50 hover:bg-secondary transition-all"
          >
            Previous
          </button>
          <span className="text-xs text-muted-foreground">
            Page {page} of {totalPages}
          </span>
          <button
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
            className="px-3 py-1.5 border border-border rounded-lg text-xs disabled:opacity-50 hover:bg-secondary transition-all"
          >
            Next
          </button>
        </div>
      </div>

      {selectedScanId && detail && (
        <div className="bg-card border border-border rounded-xl overflow-hidden">
          <div className="px-5 py-4 border-b border-border flex justify-between items-center">
            <div>
              <div className="font-semibold text-sm text-foreground">Non-Compliant Hosts</div>
              <div className="text-xs text-muted-foreground mt-0.5">
                {failingHosts.length} host(s) with deviations in scan {selectedScanId.slice(0, 8)}
              </div>
            </div>
            <div className="flex items-center gap-2">
              {ticketError && <span className="text-xs text-red-500">{ticketError}</span>}
              {ticketSuccess && <span className="text-xs text-green-500">{ticketSuccess}</span>}
            </div>
          </div>
          {failingHosts.length === 0 ? (
            <div className="p-5 text-sm text-muted-foreground">No hosts with deviations for this scan.</div>
          ) : (
            <div className="overflow-auto max-h-[60vh]">
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-card z-10">
                  <tr className="border-b border-border">
                    <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Hostname</th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                    <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Deviations</th>
                    <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Open Tickets</th>
                  </tr>
                </thead>
                <tbody>
                  {failingHosts.map((result) => {
                    const openIncidents = result.incidents.filter((i) => !i.service_now_ticket_opened)
                    return (
                      <tr key={result.id} className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]">
                        <td className="px-5 py-3 text-foreground font-medium font-mono text-xs">{result.hostname}</td>
                        <td className="px-5 py-3"><StatusBadge status={result.status} /></td>
                        <td className="px-5 py-3 text-right text-red-500 text-xs font-medium">{result.deviations_found}</td>
                        <td className="px-5 py-3">
                          {openIncidents.length > 0 ? (
                            <button
                              onClick={() => openHostTickets(result)}
                              className="px-3 py-1 bg-primary text-primary-foreground rounded-md text-xs font-semibold hover:shadow-lg hover:shadow-primary/20 transition-all"
                            >
                              Open Host Ticket
                            </button>
                          ) : (
                            <span className="text-xs text-muted-foreground">All tickets opened</span>
                          )}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {selectedScanId && detail && failingHosts.length > 0 && (
        <div className="flex flex-col gap-4 max-h-[70vh] overflow-y-auto pr-1">
          {failingHosts.map((result) => (
            <div key={result.id} className="bg-card border border-border rounded-xl overflow-hidden">
              <div className="px-5 py-4 border-b border-border flex justify-between items-center">
                <div>
                  <div className="font-semibold text-sm text-foreground">Incidents for {result.hostname}</div>
                  <div className="text-xs text-muted-foreground mt-0.5">{result.incidents.length} incident(s)</div>
                </div>
                {result.incidents.some((i) => !i.service_now_ticket_opened) && (
                  <button
                    onClick={() => openHostTickets(result)}
                    className="px-3 py-1.5 bg-primary text-primary-foreground rounded-md text-xs font-semibold hover:shadow-lg hover:shadow-primary/20 transition-all"
                  >
                    Open Host Ticket
                  </button>
                )}
              </div>
              <div className="overflow-auto max-h-[50vh]">
                <table className="w-full text-sm">
                  <thead className="sticky top-0 bg-card z-10">
                    <tr className="border-b border-border">
                      <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Number</th>
                      <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Entry</th>
                      <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Expected</th>
                      <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Actual</th>
                      <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Severity</th>
                      <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                      <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground"></th>
                    </tr>
                  </thead>
                  <tbody>
                    {result.incidents.map((incident) => (
                      <tr key={incident.id} className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]">
                        <td className="px-5 py-3 text-foreground font-mono text-xs">{incident.incident_number}</td>
                        <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{incident.entry_key}</td>
                        <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{incident.expected_value || '-'}</td>
                        <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{incident.actual_value}</td>
                        <td className="px-5 py-3 text-muted-foreground text-xs">{incident.severity}</td>
                        <td className="px-5 py-3 text-muted-foreground text-xs">{incident.status}</td>
                        <td className="px-5 py-3 text-right">

                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
