import { useEffect, useState } from 'react'
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
}

interface PaginatedHosts {
  items: Host[]
  total: number
  page: number
  limit: number
}

export default function HostsPage() {
  const [hosts, setHosts] = useState<Host[]>([])
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [totalHosts, setTotalHosts] = useState(0)

  const fetchHosts = (nextPage = page, nextSize = pageSize, nextSearch = search) => {
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
  }

  useEffect(() => {
    fetchHosts()
  }, [page, pageSize, search])

  const totalPages = Math.max(1, Math.ceil(totalHosts / pageSize))

  return (
    <div className="flex flex-col gap-6">
      {error && <div className="text-xs text-red-500">{error}</div>}

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
          <div>
            <div className="font-semibold text-sm text-foreground">Discovered Hosts</div>
            <div className="text-xs text-muted-foreground mt-0.5">{totalHosts} hosts discovered by Ansible scans</div>
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
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Hostname</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">OS Type</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">OS Name</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Environment</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Datacenter</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Snapshots</th>
              </tr>
            </thead>
            <tbody>
              {hosts.map((host) => (
                <tr key={host.id} className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]">
                  <td className="px-5 py-3 text-foreground font-medium font-mono text-xs">{host.hostname}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{host.os_type}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{host.os_name || '-'}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{host.environment || '-'}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{host.datacenter || '-'}</td>
                  <td className="px-5 py-3 text-xs">
                    <Link to={`/snapshots/${host.id}/passwd`} className="text-primary hover:underline mr-2">passwd</Link>
                    <Link to={`/snapshots/${host.id}/group`} className="text-primary hover:underline">group</Link>
                  </td>
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
    </div>
  )
}
