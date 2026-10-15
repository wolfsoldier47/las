import { useEffect, useState } from 'react'
import api from '../api/client'

interface FileKeyStat {
  key: string
  host_count: number
  occurrence_count: number
}

// KeyRow is one row in the explorer: a key within one file type. The same
// key can appear twice (once for passwd, once for group) with separate host
// listings.
interface KeyRow extends FileKeyStat {
  file_type: 'passwd' | 'group'
}

interface FileKeyStatsData {
  items: KeyRow[]
  total: number
  page: number
  limit: number
}

interface FileKeyHost {
  hostname: string
  value: string
}

interface FileKeyHostsData {
  items: FileKeyHost[]
  total: number
  page: number
  limit: number
}

type FileTypeTab = 'all' | 'passwd' | 'group'

const KEY_PAGE_SIZE = 50
const HOST_PAGE_SIZE = 25

// ScanKeyExplorer lists every unique key found in the selected scan
// (/etc/passwd accounts and /etc/group groups). Clicking a key expands it
// and loads the hosts carrying that key, paginated.
export function ScanKeyExplorer({ scanId }: { scanId: string | null }) {
  const [tab, setTab] = useState<FileTypeTab>('all')
  const [keys, setKeys] = useState<FileKeyStatsData | null>(null)
  const [keysPage, setKeysPage] = useState(1)
  const [expanded, setExpanded] = useState<{ key: string; fileType: 'passwd' | 'group' } | null>(null)
  const [hosts, setHosts] = useState<FileKeyHostsData | null>(null)
  const [hostsPage, setHostsPage] = useState(1)
  const [hostSearch, setHostSearch] = useState('')
  const [hostSearchInput, setHostSearchInput] = useState('')
  const [error, setError] = useState('')

  // Debounce the host search box so typing does not fire a request per keystroke.
  useEffect(() => {
    const timer = setTimeout(() => {
      setHostSearch(hostSearchInput.trim())
      setHostsPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [hostSearchInput])

  // Reset everything when the scan changes.
  useEffect(() => {
    setKeys(null)
    setKeysPage(1)
    setExpanded(null)
    setHosts(null)
    setHostSearch('')
    setHostSearchInput('')
    setError('')
  }, [scanId])

  // Load keys for the current tab. "all" concatenates both file types so a
  // key appearing in both /etc/passwd and /etc/group shows as two rows with
  // their own host listings.
  useEffect(() => {
    if (!scanId) return
    let cancelled = false

    const load = async () => {
      try {
        let items: KeyRow[]
        let total: number
        if (tab === 'all') {
          const [pw, gr] = await Promise.all([
            api.get(`/scans/${scanId}/file-keys?file_type=passwd&page=1&limit=100000`),
            api.get(`/scans/${scanId}/file-keys?file_type=group&page=1&limit=100000`),
          ])
          items = [
            ...(pw.data.items as FileKeyStat[]).map((s) => ({ ...s, file_type: 'passwd' as const })),
            ...(gr.data.items as FileKeyStat[]).map((s) => ({ ...s, file_type: 'group' as const })),
          ].sort((a, b) => b.host_count - a.host_count || a.key.localeCompare(b.key))
          total = items.length
        } else {
          const res = await api.get(`/scans/${scanId}/file-keys?file_type=${tab}&page=1&limit=100000`)
          items = (res.data.items as FileKeyStat[]).map((s) => ({ ...s, file_type: tab }))
          total = res.data.total
        }
        if (cancelled) return
        setKeys({ items, total, page: 1, limit: KEY_PAGE_SIZE })
        setKeysPage(1)
      } catch (err: any) {
        if (!cancelled) setError(err.response?.data?.error || err.message)
      }
    }

    load()
    return () => {
      cancelled = true
    }
  }, [scanId, tab])

  // Load hosts for the expanded key, filtered by the host search box.
  useEffect(() => {
    if (!scanId || !expanded) return
    let cancelled = false
    setHosts(null)
    api
      .get(
        `/scans/${scanId}/file-keys/${encodeURIComponent(expanded.key)}/hosts?file_type=${expanded.fileType}&page=${hostsPage}&limit=${HOST_PAGE_SIZE}&search=${encodeURIComponent(hostSearch)}`
      )
      .then((res) => {
        if (!cancelled) setHosts(res.data)
      })
      .catch((err) => {
        if (!cancelled) setError(err.response?.data?.error || err.message)
      })
    return () => {
      cancelled = true
    }
  }, [scanId, expanded, hostsPage, hostSearch])

  if (!scanId) return null

  const totalKeys = keys?.total ?? 0
  const totalKeyPages = Math.max(1, Math.ceil(totalKeys / KEY_PAGE_SIZE))
  const safeKeysPage = Math.min(keysPage, totalKeyPages)
  const visibleKeys = (keys?.items || []).slice((safeKeysPage - 1) * KEY_PAGE_SIZE, safeKeysPage * KEY_PAGE_SIZE)

  const toggle = (key: string, fileType: 'passwd' | 'group') => {
    if (expanded?.key === key && expanded.fileType === fileType) {
      setExpanded(null)
      setHosts(null)
    } else {
      setExpanded({ key, fileType })
      setHostsPage(1)
      setHostSearch('')
    }
  }

  const hostTotalPages = Math.max(1, Math.ceil((hosts?.total ?? 0) / HOST_PAGE_SIZE))

  return (
    <div className="bg-card border border-border rounded-xl overflow-hidden">
      <div className="px-5 py-4 border-b border-border flex justify-between items-center flex-wrap gap-3">
        <div>
          <div className="font-semibold text-sm text-foreground">Keys Found in This Scan</div>
          <div className="text-xs text-muted-foreground mt-0.5">
            {keys ? `${totalKeys.toLocaleString()} unique keys across all hosts — click a key to see its hosts` : 'Loading...'}
          </div>
        </div>
        <div className="flex gap-1">
          {(['all', 'passwd', 'group'] as FileTypeTab[]).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`px-3 py-1 rounded-lg text-xs transition-colors ${
                tab === t
                  ? 'bg-primary/10 text-primary border border-primary/30'
                  : 'text-muted-foreground border border-border hover:bg-secondary'
              }`}
            >
              {t === 'all' ? 'All' : t === 'passwd' ? '/etc/passwd' : '/etc/group'}
            </button>
          ))}
        </div>
      </div>

      {error && <div className="px-5 py-3 text-xs text-red-500">{error}</div>}

      <div className="overflow-x-auto max-h-[60vh] overflow-y-auto">
        <table className="w-full text-sm">
          <thead className="sticky top-0 z-10">
            <tr className="border-b border-border">
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground bg-background w-10">#</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground bg-background">Key</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground bg-background">File</th>
              <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground bg-background">Hosts</th>
              <th className="px-4 py-3 text-right text-xs font-medium text-muted-foreground bg-background">Occurrences</th>
            </tr>
          </thead>
          <tbody>
            {!keys ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground text-sm">Loading...</td>
              </tr>
            ) : visibleKeys.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground text-sm">No keys found for this scan.</td>
              </tr>
            ) : (
              visibleKeys.map((stat, idx) => {
                const isOpen = expanded?.key === stat.key && expanded?.fileType === stat.file_type
                return (
                  <ExpandableKeyRow
                    key={`${stat.file_type}-${stat.key}`}
                    stat={stat}
                    rank={(safeKeysPage - 1) * KEY_PAGE_SIZE + idx + 1}
                    isOpen={isOpen}
                    onToggle={() => toggle(stat.key, stat.file_type)}
                    hosts={isOpen ? hosts : null}
                    hostsPage={hostsPage}
                    hostTotalPages={hostTotalPages}
                    onHostsPage={setHostsPage}
                    hostSearchInput={hostSearchInput}
                    onHostSearch={setHostSearchInput}
                  />
                )
              })
            )}
          </tbody>
        </table>
      </div>

      <div className="px-5 py-3 border-t border-border flex justify-end items-center gap-2">
        <button
          onClick={() => setKeysPage((p) => Math.max(1, p - 1))}
          disabled={safeKeysPage <= 1}
          className="px-3 py-1.5 border border-border rounded-lg text-xs disabled:opacity-50 hover:bg-secondary transition-all"
        >
          Previous
        </button>
        <span className="text-xs text-muted-foreground">
          Page {safeKeysPage} of {totalKeyPages}
        </span>
        <button
          onClick={() => setKeysPage((p) => Math.min(totalKeyPages, p + 1))}
          disabled={safeKeysPage >= totalKeyPages}
          className="px-3 py-1.5 border border-border rounded-lg text-xs disabled:opacity-50 hover:bg-secondary transition-all"
        >
          Next
        </button>
      </div>
    </div>
  )
}

function ExpandableKeyRow({
  stat,
  rank,
  isOpen,
  onToggle,
  hosts,
  hostsPage,
  hostTotalPages,
  onHostsPage,
  hostSearchInput,
  onHostSearch,
}: {
  stat: KeyRow
  rank: number
  isOpen: boolean
  onToggle: () => void
  hosts: FileKeyHostsData | null
  hostsPage: number
  hostTotalPages: number
  onHostsPage: (page: number) => void
  hostSearchInput: string
  onHostSearch: (value: string) => void
}) {
  return (
    <>
      <tr onClick={onToggle} className={`border-b border-border/50 cursor-pointer transition-colors ${isOpen ? 'bg-primary/10' : 'hover:bg-primary/[0.03]'}`}>
        <td className="px-4 py-2.5 text-muted-foreground text-xs">
          {isOpen ? '▲' : '▼'} {rank}
        </td>
        <td className="px-4 py-2.5 text-foreground font-mono text-xs font-medium">{stat.key}</td>
        <td className="px-4 py-2.5">
          <span className={`text-[10px] px-1.5 py-0.5 rounded border ${stat.file_type === 'passwd' ? 'bg-primary/10 text-primary border-primary/15' : 'bg-green-500/10 text-green-500 border-green-500/15'}`}>
            {stat.file_type}
          </span>
        </td>
        <td className="px-4 py-2.5 text-right text-foreground text-xs font-medium">{stat.host_count.toLocaleString()}</td>
        <td className="px-4 py-2.5 text-right text-muted-foreground text-xs">{stat.occurrence_count.toLocaleString()}</td>
      </tr>
      {isOpen && (
        <tr className="border-b border-border/50">
          <td colSpan={5} className="px-6 py-3 bg-primary/[0.02]">
            {!hosts ? (
              <div className="text-xs text-muted-foreground">Loading hosts...</div>
            ) : hosts.items.length === 0 ? (
              <div className="text-xs text-muted-foreground">No hosts found for this key.</div>
            ) : (
              <div className="flex flex-col gap-2">
                <div className="relative">
                  <svg className="absolute left-3 top-2.5 w-3.5 h-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
                  </svg>
                  <input
                    type="text"
                    value={hostSearchInput}
                    onChange={(e) => onHostSearch(e.target.value)}
                    onClick={(e) => e.stopPropagation()}
                    placeholder="Search hosts..."
                    className="w-full pl-8 pr-3 py-1.5 bg-background border border-border rounded-lg text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
                  />
                </div>
                <div className="text-[11px] text-muted-foreground">
                  {hosts.total.toLocaleString()} host(s) {hostSearchInput.trim() ? 'matching' : 'carry this key'} — page {hosts.page} of {hostTotalPages}
                </div>
                <table className="w-full text-sm border border-border rounded-lg overflow-hidden">
                  <thead>
                    <tr className="bg-secondary/50">
                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">Hostname</th>
                      <th className="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground">Value</th>
                    </tr>
                  </thead>
                  <tbody>
                    {hosts.items.map((h) => (
                      <tr key={h.hostname} className="border-t border-border/50">
                        <td className="px-3 py-1.5 text-foreground font-mono text-[11px]">{h.hostname}</td>
                        <td className="px-3 py-1.5 text-muted-foreground font-mono text-[11px] break-all">{h.value}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {hostTotalPages > 1 && (
                  <div className="flex justify-end items-center gap-2">
                    <button
                      onClick={(e) => { e.stopPropagation(); onHostsPage(Math.max(1, hostsPage - 1)) }}
                      disabled={hostsPage <= 1}
                      className="px-2 py-1 border border-border rounded text-[11px] disabled:opacity-50 hover:bg-secondary"
                    >
                      Prev
                    </button>
                    <span className="text-[11px] text-muted-foreground">{hostsPage} / {hostTotalPages}</span>
                    <button
                      onClick={(e) => { e.stopPropagation(); onHostsPage(Math.min(hostTotalPages, hostsPage + 1)) }}
                      disabled={hostsPage >= hostTotalPages}
                      className="px-2 py-1 border border-border rounded text-[11px] disabled:opacity-50 hover:bg-secondary"
                    >
                      Next
                    </button>
                  </div>
                )}
              </div>
            )}
          </td>
        </tr>
      )}
    </>
  )
}
