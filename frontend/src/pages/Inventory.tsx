import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import api from '../api/client'

interface Host {
  id: string
  hostname: string
  os_type: string
  os_name?: string
  os_version?: string
  environment?: string
  datacenter?: string
  created_at: string
}

interface PaginatedHosts {
  items: Host[]
  total: number
  page: number
  limit: number
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

export default function Inventory() {
  const [hosts, setHosts] = useState<Host[]>([])
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [totalHosts, setTotalHosts] = useState(0)

  const fetchHosts = useCallback(
    (nextPage = page, nextSize = pageSize, nextSearch = search) => {
      api
        .get(`/hosts?page=${nextPage}&limit=${nextSize}&search=${encodeURIComponent(nextSearch)}`)
        .then((res) => {
          const data: PaginatedHosts | Host[] = res.data
          if (Array.isArray(data)) {
            setHosts(data)
            setTotalHosts(data.length)
          } else {
            setHosts(data?.items || [])
            setTotalHosts(data?.total || 0)
          }
        })
        .catch((err) => setError(err.message))
    },
    [page, pageSize, search]
  )

  useEffect(() => {
    fetchHosts()
  }, [fetchHosts])

  const groups = hosts.reduce<Record<string, Host[]>>((acc, host) => {
    const key = host.environment || 'unknown'
    if (!acc[key]) acc[key] = []
    acc[key].push(host)
    return acc
  }, {})

  const total = totalHosts
  const withDeviation = 0 // would require scan result data
  const online = total // placeholder
  const totalPages = Math.max(1, Math.ceil(totalHosts / pageSize))

  return (
    <div className="flex flex-col gap-6">
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-card border border-border rounded-xl p-5 flex flex-col gap-2 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(250,204,21,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Total Hosts</div>
          <div className="text-[28px] font-bold text-foreground tracking-tight">{total}</div>
          <div className="text-xs text-muted-foreground">Across {Object.keys(groups).length} groups</div>
        </div>
        <div className="bg-card border border-border rounded-xl p-5 flex flex-col gap-2 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(34,197,94,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Online</div>
          <div className="text-[28px] font-bold text-green-500 tracking-tight">{online}</div>
          <div className="text-xs text-muted-foreground">Tracked hosts</div>
        </div>
        <div className="bg-card border border-border rounded-xl p-5 flex flex-col gap-2 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(239,68,68,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">With Deviations</div>
          <div className="text-[28px] font-bold text-red-500 tracking-tight">{withDeviation}</div>
          <div className="text-xs text-muted-foreground">From latest scan</div>
        </div>
        <div className="bg-card border border-border rounded-xl p-5 flex flex-col gap-2 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(250,204,21,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Avg Compliance</div>
          <div className="text-[28px] font-bold text-primary tracking-tight">{total ? '100%' : 'N/A'}</div>
          <div className="text-xs text-muted-foreground">Fleet-wide average</div>
        </div>
      </div>

      {error && <div className="text-xs text-red-500">{error}</div>}

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
          <div>
            <div className="font-semibold text-sm text-foreground">Inventory</div>
            <div className="text-xs text-muted-foreground mt-0.5">{totalHosts} hosts</div>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value)
                setPage(1)
              }}
              placeholder="Search hosts..."
              className="bg-background border border-border rounded-lg px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 w-full sm:w-64"
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

        {Object.entries(groups).map(([groupName, groupHosts]) => (
          <div key={groupName} className="border-b border-border last:border-b-0">
            <div className="px-5 py-3 border-b border-border/50 bg-secondary/30 flex items-center gap-3">
              <div className="w-7 h-7 rounded-lg bg-secondary flex items-center justify-center">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#a1a1aa" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <rect width="20" height="14" x="2" y="3" rx="2"/><line x1="8" x2="8" y1="21" y2="3"/><line x1="16" x2="16" y1="21" y2="3"/>
                </svg>
              </div>
              <div>
                <div className="font-semibold text-sm text-foreground">{groupName}</div>
                <div className="text-xs text-muted-foreground">{groupHosts.length} hosts on this page</div>
              </div>
            </div>
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border/50">
                  <th className="px-5 py-2.5 text-left text-xs font-medium text-muted-foreground">Host</th>
                  <th className="px-5 py-2.5 text-left text-xs font-medium text-muted-foreground">OS Type</th>
                  <th className="px-5 py-2.5 text-left text-xs font-medium text-muted-foreground">OS Name</th>
                  <th className="px-5 py-2.5 text-left text-xs font-medium text-muted-foreground">Environment</th>
                  <th className="px-5 py-2.5 text-left text-xs font-medium text-muted-foreground">Datacenter</th>
                  <th className="px-5 py-2.5 text-left text-xs font-medium text-muted-foreground">Status</th>
                  <th className="px-5 py-2.5 text-right text-xs font-medium text-muted-foreground"></th>
                </tr>
              </thead>
              <tbody>
                {groupHosts.map((host) => (
                  <tr key={host.id} className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]">
                    <td className="px-5 py-3 text-foreground font-medium font-mono text-xs">{host.hostname}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{host.os_type}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{host.os_name || '-'}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{host.environment || '-'}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{host.datacenter || '-'}</td>
                    <td className="px-5 py-3"><StatusBadge status="clean" /></td>
                    <td className="px-5 py-3 text-right">
                      <Link
                        to={`/snapshots/${host.id}/passwd`}
                        className="text-xs text-primary hover:underline mr-2"
                      >
                        passwd
                      </Link>
                      <Link
                        to={`/snapshots/${host.id}/group`}
                        className="text-xs text-primary hover:underline"
                      >
                        group
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}

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
    </div>
  )
}
