import React, { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import api, { downloadScanReport } from '../api/client'

interface Incident {
  id: string
  incident_number: string
  file_type: 'passwd' | 'group'
  entry_key: string
  expected_value?: string
  actual_value: string
  status: string
}

interface AllowedDeviation {
  file_type: string
  entry_key: string
  expected_value?: string
  actual_value: string
}

interface HostResult {
  id: string
  host_id: string
  hostname: string
  status: string
  os_type?: string
  os_version?: string
  os_name?: string
  environment?: string
  datacenter?: string
  baseline_version_at_scan?: number
  no_baseline?: boolean
  deviations_found: number
  allowed_deviations: AllowedDeviation[]
  incidents: Incident[]
}

interface BaselineSnapshot {
  host_id?: string
  hostname?: string
  os_type: string
  file_type: 'passwd' | 'group'
  version: number
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
  callbacks_received: number
  successful_hosts: number
  failed_hosts: number
  failed_host_names?: string[]
  baseline_snapshot?: BaselineSnapshot[]
  created_at: string
}

interface ScanDetailData {
  job: ScanJob
  results: HostResult[]
  total: number
  page: number
  limit: number
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
    failed_host: 'bg-red-500/10 text-red-500 border-red-500/15',
    no_baseline: 'bg-orange-500/10 text-orange-500 border-orange-500/15',
    pending: 'bg-primary/10 text-primary border-primary/15',
  }
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border ${styles[status] || styles.pending}`}>
      {status}
    </span>
  )
}

export default function ScanDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [detail, setDetail] = useState<ScanDetailData | null>(null)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(50)

  const [expandedHost, setExpandedHost] = useState<string | null>(null)
  const [hostDetails, setHostDetails] = useState<Record<string, HostResult>>({})
  const [loadingHost, setLoadingHost] = useState<string | null>(null)
  const [hostError, setHostError] = useState<Record<string, string>>({})

  const fetchDetail = (nextPage = page, nextSize = pageSize) => {
    if (!id) return
    api
      .get(`/scans/${id}?page=${nextPage}&limit=${nextSize}`)
      .then((res) => {
        const data: ScanDetailData = res.data
        setDetail({
          job: data.job,
          results: (data.results || []).map((r) => ({
            ...r,
            incidents: [],
            allowed_deviations: r.allowed_deviations || [],
          })),
          total: data.total || 0,
          page: data.page || nextPage,
          limit: data.limit || nextSize,
        })
      })
      .catch((err) => setError(err.message))
  }

  useEffect(() => {
    setPage(1)
  }, [id])

  useEffect(() => {
    fetchDetail(page, pageSize)
  }, [id, page, pageSize])

  const toggleHost = (hostResult: HostResult) => {
    const hostId = hostResult.host_id
    if (expandedHost === hostId) {
      setExpandedHost(null)
      return
    }
    setExpandedHost(hostId)

    if (hostDetails[hostId] || hostResult.status === 'failed_host') {
      return
    }

    setLoadingHost(hostId)
    setHostError((prev) => ({ ...prev, [hostId]: '' }))
    api
      .get(`/scans/${id}/hosts/${hostId}`)
      .then((res) => {
        const data: HostResult = res.data
        setHostDetails((prev) => ({
          ...prev,
          [hostId]: {
            ...data,
            incidents: data.incidents || [],
            allowed_deviations: data.allowed_deviations || [],
          },
        }))
      })
      .catch((err) => {
        setHostError((prev) => ({ ...prev, [hostId]: err.response?.data?.error || err.message }))
      })
      .finally(() => setLoadingHost(null))
  }

  if (error) return <p className="text-red-500 text-sm">{error}</p>
  if (!detail) return <p className="text-muted-foreground text-sm">Loading...</p>

  const failedRows: HostResult[] = (detail.job.failed_host_names || []).map((hostname) => ({
    id: `failed-${hostname}`,
    host_id: `failed-${hostname}`,
    hostname,
    status: 'failed_host',
    deviations_found: 0,
    allowed_deviations: [],
    incidents: [],
  }))

  const displayHosts: HostResult[] = page === 1 ? [...failedRows, ...detail.results] : detail.results
  const totalHosts = detail.total + failedRows.length
  const totalPages = Math.max(1, Math.ceil(totalHosts / pageSize))
  const safePage = Math.min(page, totalPages)

  const activeHost = expandedHost ? hostDetails[expandedHost] || detail.results.find((r) => r.host_id === expandedHost) || failedRows.find((r) => r.host_id === expandedHost) : null

  return (
    <div className="flex flex-col gap-6">
      <div className="bg-card border border-border rounded-xl p-5">
        <div className="flex justify-between items-start mb-4">
          <div className="font-semibold text-sm text-foreground">Scan Job Details</div>
          <button
            onClick={() => {
              if (id) {
                downloadScanReport(id).catch((err) => setError(err.response?.data?.error || err.message))
              }
            }}
            className="text-xs px-3 py-1.5 border border-border rounded-lg hover:bg-secondary transition-all"
          >
            Download PDF Report
          </button>
        </div>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div><span className="text-muted-foreground">Job ID:</span> <span className="font-mono text-xs">{detail.job.id}</span></div>
          <div><span className="text-muted-foreground">AAP Job ID:</span> <span className="font-mono text-xs">{detail.job.ansible_job_id || '-'}</span></div>
          <div><span className="text-muted-foreground">Template ID:</span> {detail.job.job_template_id}</div>
          <div><span className="text-muted-foreground">Limit:</span> {detail.job.limit || '-'}</div>
          <div><span className="text-muted-foreground">Status:</span> <StatusBadge status={detail.job.status} /></div>
          <div><span className="text-muted-foreground">Callbacks:</span> {detail.job.callbacks_received}</div>
          <div><span className="text-muted-foreground">Successful:</span> {detail.job.successful_hosts}</div>
          <div><span className="text-muted-foreground">Failed:</span> {detail.job.failed_hosts}</div>
          <div><span className="text-muted-foreground">Created:</span> {new Date(detail.job.created_at).toLocaleString()}</div>
        </div>
      </div>

      {detail.job.baseline_snapshot && detail.job.baseline_snapshot.length > 0 && (
        <div className="bg-card border border-border rounded-xl overflow-hidden">
          <div className="px-5 py-4 border-b border-border">
            <div className="font-semibold text-sm text-foreground">Active Baselines</div>
            <div className="text-xs text-muted-foreground mt-0.5">
              Master files used for the initial deviation check
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border">
                  <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Scope</th>
                  <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">OS Type</th>
                  <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">File</th>
                  <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Version</th>
                  <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Entries</th>
                  <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Description</th>
                </tr>
              </thead>
              <tbody>
                {detail.job.baseline_snapshot.map((baseline, idx) => (
                  <tr key={idx} className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]">
                    <td className="px-5 py-3 text-foreground font-mono text-xs">
                      {baseline.host_id ? baseline.hostname || baseline.host_id : 'Global'}
                    </td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{baseline.os_type}</td>
                    <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{baseline.file_type}</td>
                    <td className="px-5 py-3 text-right text-muted-foreground text-xs">{baseline.version}</td>
                    <td className="px-5 py-3 text-right text-muted-foreground text-xs">{baseline.entry_count}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{baseline.description || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex justify-between items-center">
          <div>
            <div className="font-semibold text-sm text-foreground">Host Results</div>
            <div className="text-xs text-muted-foreground mt-0.5">{totalHosts} host(s) — click a row to view deviations</div>
          </div>
          <div className="flex items-center gap-2">
            <select
              value={pageSize}
              onChange={(e) => {
                setPageSize(Number(e.target.value))
                setPage(1)
              }}
              className="bg-background border border-border rounded-lg px-2 py-1 text-xs text-foreground outline-none"
            >
              <option value={25}>25 / page</option>
              <option value={50}>50 / page</option>
              <option value={100}>100 / page</option>
              <option value={250}>250 / page</option>
              <option value={500}>500 / page</option>
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
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Hostname</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">OS Type</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">OS Version</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Environment</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Datacenter</th>
                <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Deviations</th>
                <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Allowed</th>
              </tr>
            </thead>
            <tbody>
              {displayHosts.map((result) => (
                <React.Fragment key={result.id}>
                  <tr
                    onClick={() => toggleHost(result)}
                    className={`border-b border-border/50 transition-colors cursor-pointer ${
                      expandedHost === result.host_id ? 'bg-primary/10' : 'hover:bg-primary/[0.03]'
                    }`}
                  >
                    <td className="px-5 py-3 text-foreground font-medium font-mono text-xs">
                      {result.hostname}
                      {result.status !== 'failed_host' && (
                        <span className="ml-2 text-[10px] text-muted-foreground">
                          {expandedHost === result.host_id ? '▲' : '▼'}
                        </span>
                      )}
                    </td>
                    <td className="px-5 py-3"><StatusBadge status={result.status} /></td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{result.os_type || '-'}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{result.os_version || '-'}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{result.environment || '-'}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{result.datacenter || '-'}</td>
                    <td className="px-5 py-3 text-right text-muted-foreground text-xs">
                      {result.status === 'failed_host' || result.no_baseline ? '-' : result.deviations_found}
                    </td>
                    <td className="px-5 py-3 text-right text-muted-foreground text-xs">
                      {result.status === 'failed_host' || result.no_baseline ? '-' : result.allowed_deviations.length}
                    </td>
                  </tr>
                  {expandedHost === result.host_id && (
                    <tr key={`${result.id}-details`}>
                      <td colSpan={8} className="px-5 py-4 bg-primary/[0.02] border-b border-border">
                        {loadingHost === result.host_id ? (
                          <p className="text-xs text-muted-foreground">Loading deviations...</p>
                        ) : hostError[result.host_id] ? (
                          <p className="text-xs text-red-500">{hostError[result.host_id]}</p>
                        ) : activeHost ? (
                          <div className="flex flex-col gap-4">
                            {activeHost.no_baseline && (
                              <div className="rounded-lg bg-orange-500/10 border border-orange-500/15 px-3 py-2 text-xs text-orange-500">
                                No baseline available for {activeHost.os_type} version {activeHost.baseline_version_at_scan ?? activeHost.os_version?.split('.')[0]}
                              </div>
                            )}
                            {(activeHost.incidents || []).length > 0 && (
                              <div>
                                <div className="font-semibold text-xs text-foreground mb-2">Incidents</div>
                                <table className="w-full text-sm border border-border rounded-lg overflow-hidden">
                                  <thead>
                                    <tr className="bg-secondary/50">
                                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">Number</th>
                                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">File</th>
                                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">Entry</th>
                                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">Expected</th>
                                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">Actual</th>
                                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">Status</th>
                                    </tr>
                                  </thead>
                                  <tbody>
                                    {activeHost.incidents.map((incident) => (
                                      <tr key={incident.id} className="border-t border-border/50">
                                        <td className="px-3 py-2 text-foreground font-mono text-[11px]">{incident.incident_number}</td>
                                        <td className="px-3 py-2 text-muted-foreground font-mono text-[11px]">/{incident.file_type}</td>
                                        <td className="px-3 py-2 text-muted-foreground font-mono text-[11px]">{incident.entry_key}</td>
                                        <td className="px-3 py-2 text-muted-foreground font-mono text-[11px]">{incident.expected_value || '-'}</td>
                                        <td className="px-3 py-2 text-muted-foreground font-mono text-[11px]">{incident.actual_value}</td>
                                        <td className="px-3 py-2 text-muted-foreground text-[11px]">{incident.status}</td>
                                      </tr>
                                    ))}
                                  </tbody>
                                </table>
                              </div>
                            )}
                            {(activeHost.allowed_deviations || []).length > 0 && (
                              <div>
                                <div className="font-semibold text-xs text-foreground mb-2">Allowed Deviations</div>
                                <table className="w-full text-sm border border-border rounded-lg overflow-hidden">
                                  <thead>
                                    <tr className="bg-secondary/50">
                                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">File</th>
                                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">Entry</th>
                                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">Expected</th>
                                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">Actual</th>
                                    </tr>
                                  </thead>
                                  <tbody>
                                    {activeHost.allowed_deviations.map((d, idx) => (
                                      <tr key={idx} className="border-t border-border/50">
                                        <td className="px-3 py-2 text-muted-foreground text-[11px]">{d.file_type}</td>
                                        <td className="px-3 py-2 text-muted-foreground font-mono text-[11px]">{d.entry_key}</td>
                                        <td className="px-3 py-2 text-muted-foreground font-mono text-[11px]">{d.expected_value || '-'}</td>
                                        <td className="px-3 py-2 text-muted-foreground font-mono text-[11px]">{d.actual_value}</td>
                                      </tr>
                                    ))}
                                  </tbody>
                                </table>
                              </div>
                            )}
                            {(activeHost.incidents || []).length === 0 && (activeHost.allowed_deviations || []).length === 0 && (
                              <p className="text-xs text-muted-foreground">No deviations recorded for this host.</p>
                            )}
                          </div>
                        ) : null}
                      </td>
                    </tr>
                  )}
                </React.Fragment>
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
    </div>
  )
}
