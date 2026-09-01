import { useEffect, useState } from 'react'
import api from '../api/client'

interface BaselineVersion {
  os_type: string
  file_type: string
  version: number
  is_active: boolean
  entry_count: number
  description?: string
  created_by?: string
  created_at: string
}

interface PaginatedBaselineVersions {
  items: BaselineVersion[]
  total: number
  page: number
  limit: number
}

interface BaselineEntry {
  id: string
  entry_key: string
  entry_value: string
  is_active: boolean
  version: number
}

export default function BaselinesPage() {
  const [versions, setVersions] = useState<BaselineVersion[]>([])
  const [osVersions, setOsVersions] = useState<Record<string, number[]>>({})
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [form, setForm] = useState({
    os_type: 'linux',
    file_type: 'passwd',
    version: '',
    content: '',
    description: '',
  })

  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [totalVersions, setTotalVersions] = useState(0)

  const [viewing, setViewing] = useState<{
    version: BaselineVersion
    entries: BaselineEntry[]
    loading: boolean
  } | null>(null)

  const [confirmUpload, setConfirmUpload] = useState(false)

  const fetchVersions = (nextPage = page, nextSize = pageSize) => {
    api
      .get(`/baselines/versions?page=${nextPage}&limit=${nextSize}`)
      .then((res) => {
        const data: PaginatedBaselineVersions | BaselineVersion[] = res.data
        if (Array.isArray(data)) {
          setVersions(data)
          setTotalVersions(data.length)
        } else {
          setVersions(data?.items || [])
          setTotalVersions(data?.total || 0)
        }
      })
      .catch((err) => setError(err.message))
  }

  const fetchOSVersions = () => {
    api
      .get('/os-versions')
      .then((res) => setOsVersions(res.data || {}))
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const availableVersions = osVersions[form.os_type] || []

  useEffect(() => {
    fetchVersions()
    fetchOSVersions()
  }, [page, pageSize])

  useEffect(() => {
    if (availableVersions.length > 0 && !availableVersions.includes(Number(form.version))) {
      setForm((prev) => ({ ...prev, version: String(availableVersions[0]) }))
    }
  }, [form.os_type, osVersions])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSuccess('')
    setConfirmUpload(true)
  }

  const doUpload = () => {
    setConfirmUpload(false)

    const payload = {
      os_type: form.os_type,
      file_type: form.file_type,
      version: Number(form.version),
      content: form.content,
      description: form.description,
    }

    api
      .post('/baselines/upload', payload)
      .then((res) => {
        setSuccess(`Created baseline for ${form.os_type} version ${res.data.version}`)
        setForm({ os_type: 'linux', file_type: 'passwd', version: String(availableVersions[0] || ''), content: '', description: '' })
        setPage(1)
        fetchVersions(1, pageSize)
      })
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const activateVersion = (v: BaselineVersion) => {
    setError('')
    setSuccess('')
    api
      .post('/baselines/versions/activate', {
        os_type: v.os_type,
        file_type: v.file_type,
        version: v.version,
      })
      .then(() => {
        setSuccess(`Activated version ${v.version}`)
        fetchVersions()
      })
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const deactivateScope = (v: BaselineVersion) => {
    setError('')
    setSuccess('')
    api
      .post('/baselines/versions/deactivate', {
        os_type: v.os_type,
        file_type: v.file_type,
        version: v.version,
      })
      .then(() => {
        setSuccess(`Deactivated ${v.os_type} version ${v.version}`)
        fetchVersions()
      })
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const viewVersionContent = (v: BaselineVersion) => {
    setViewing({ version: v, entries: [], loading: true })
    const params = new URLSearchParams({
      os_type: v.os_type,
      file_type: v.file_type,
      version: String(v.version),
    })
    api
      .get(`/baselines?${params.toString()}`)
      .then((res) => {
        const entries: BaselineEntry[] = (res.data || []).sort((a: BaselineEntry, b: BaselineEntry) =>
          a.entry_key.localeCompare(b.entry_key)
        )
        setViewing({ version: v, entries, loading: false })
      })
      .catch((err) => {
        setError(err.response?.data?.error || err.message)
        setViewing(null)
      })
  }

  const scopeLabel = (v: BaselineVersion) => `${v.os_type} / ${v.file_type}`

  const renderFileContent = (v: BaselineVersion, entries: BaselineEntry[]) => {
    if (v.file_type === 'group') {
      return entries.map((e) => `${e.entry_key}:${e.entry_value}`).join('\n')
    }
    return entries.map((e) => `${e.entry_key}:${e.entry_value}`).join('\n')
  }

  const totalPages = Math.max(1, Math.ceil(totalVersions / pageSize))

  return (
    <div className="flex flex-col gap-6">
      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <div className="font-semibold text-sm text-foreground">Upload Master File</div>
          <div className="text-xs text-muted-foreground mt-0.5">
            Paste the full contents of /etc/passwd or /etc/group. The selected OS major version is used as the baseline version; only one baseline per OS major version can be active at a time.
          </div>
        </div>
        <form onSubmit={handleSubmit} className="p-5 grid grid-cols-2 gap-4">
          {error && <div className="col-span-2 text-xs text-red-500">{error}</div>}
          {success && <div className="col-span-2 text-xs text-green-500">{success}</div>}
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">OS Type</label>
            <select
              className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={form.os_type}
              onChange={(e) => setForm({ ...form, os_type: e.target.value })}
            >
              <option value="linux">RHEL</option>
              <option value="solaris">Solaris</option>
              <option value="aix">AIX</option>
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">OS Major Version</label>
            <select
              className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={form.version}
              onChange={(e) => setForm({ ...form, version: e.target.value })}
              required
            >
              {availableVersions.length === 0 && <option value="">No versions configured</option>}
              {availableVersions.map((v) => (
                <option key={v} value={v}>
                  {v}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">File Type</label>
            <select
              className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={form.file_type}
              onChange={(e) => setForm({ ...form, file_type: e.target.value })}
            >
              <option value="passwd">passwd</option>
              <option value="group">group</option>
            </select>
          </div>
          <div className="col-span-2">
            <label className="block text-xs font-medium text-muted-foreground mb-1">File Contents</label>
            <textarea
              rows={12}
              className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 font-mono"
              value={form.content}
              onChange={(e) => setForm({ ...form, content: e.target.value })}
              placeholder={`root:x:0:0:root:/root:/bin/bash\nadmin:x:1000:1000:admin:/home/admin:/bin/bash`}
              required
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">Description</label>
            <input
              type="text"
              className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
            />
          </div>
          <div className="col-span-2">
            <button
              type="submit"
              className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-semibold hover:shadow-lg hover:shadow-primary/20 transition-all"
            >
              Upload & Activate
            </button>
          </div>
        </form>
      </div>

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
          <div>
            <div className="font-semibold text-sm text-foreground">Master File Versions</div>
            <div className="text-xs text-muted-foreground mt-0.5">{totalVersions} versions</div>
          </div>
          <div className="flex items-center gap-2">
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
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Scope</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Version</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Entries</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Active</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Description</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Created By</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Created</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground"></th>
              </tr>
            </thead>
            <tbody>
              {versions.map((v) => (
                <tr
                  key={`${v.os_type}-${v.file_type}-${v.version}`}
                  className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]"
                >
                  <td className="px-5 py-3 text-foreground text-xs">
                    <button
                      onClick={() => viewVersionContent(v)}
                      className="text-left hover:text-primary hover:underline"
                      title="Click to view content"
                    >
                      {scopeLabel(v)}
                    </button>
                  </td>
                  <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{v.version}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{v.entry_count}</td>
                  <td className="px-5 py-3 text-xs">
                    {v.is_active ? (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border bg-green-500/10 text-green-500 border-green-500/15">
                        Active
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border bg-muted text-muted-foreground border-border">
                        Inactive
                      </span>
                    )}
                  </td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{v.description || '—'}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{v.created_by || '—'}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{new Date(v.created_at).toLocaleString()}</td>
                  <td className="px-5 py-3 text-right">
                    <div className="flex gap-2 justify-end">
                      <button
                        onClick={() => viewVersionContent(v)}
                        className="px-2 py-1 bg-background border border-border text-foreground rounded text-xs hover:bg-primary/[0.05] transition-colors"
                      >
                        View
                      </button>
                      {!v.is_active && (
                        <button
                          onClick={() => activateVersion(v)}
                          className="px-2 py-1 bg-primary text-primary-foreground rounded text-xs font-semibold hover:shadow-lg hover:shadow-primary/20 transition-all"
                        >
                          Activate
                        </button>
                      )}
                      {v.is_active && (
                        <button
                          onClick={() => deactivateScope(v)}
                          className="px-2 py-1 bg-secondary border border-[#3f3f46] text-foreground rounded text-xs hover:bg-[#3f3f46] transition-colors"
                        >
                          Disable
                        </button>
                      )}
                    </div>
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

      {viewing && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          onClick={() => setViewing(null)}
        >
          <div
            className="bg-card border border-border rounded-xl max-w-3xl w-full max-h-[80vh] flex flex-col shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="px-5 py-4 border-b border-border flex justify-between items-center">
              <div>
                <div className="font-semibold text-sm text-foreground">
                  {viewing.version.file_type === 'passwd' ? '/etc/passwd' : '/etc/group'} — Version {viewing.version.version}
                </div>
                <div className="text-xs text-muted-foreground">{scopeLabel(viewing.version)}</div>
              </div>
              <button
                onClick={() => setViewing(null)}
                className="text-muted-foreground hover:text-foreground text-lg leading-none"
              >
                ×
              </button>
            </div>
            <div className="p-5 overflow-auto flex-1">
              {viewing.loading ? (
                <div className="text-xs text-muted-foreground">Loading…</div>
              ) : viewing.entries.length === 0 ? (
                <div className="text-xs text-muted-foreground">No entries found.</div>
              ) : (
                <pre className="bg-background border border-border rounded-lg p-3 text-xs font-mono text-foreground whitespace-pre-wrap">
                  {renderFileContent(viewing.version, viewing.entries)}
                </pre>
              )}
            </div>
            <div className="px-5 py-4 border-t border-border flex justify-end">
              <button
                onClick={() => setViewing(null)}
                className="px-3 py-1.5 bg-secondary border border-border text-foreground rounded-lg text-xs hover:bg-primary/[0.05] transition-colors"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {confirmUpload && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          onClick={() => setConfirmUpload(false)}
        >
          <div
            className="bg-card border border-border rounded-xl max-w-md w-full shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="px-5 py-4 border-b border-border">
              <div className="font-semibold text-sm text-foreground">Confirm Upload</div>
            </div>
            <div className="p-5 text-sm text-foreground">
              Uploading will create a new version and activate it automatically. Master file versions cannot be edited after creation.
            </div>
            <div className="px-5 py-4 border-t border-border flex justify-end gap-2">
              <button
                onClick={() => setConfirmUpload(false)}
                className="px-3 py-1.5 bg-secondary border border-border text-foreground rounded-lg text-xs hover:bg-primary/[0.05] transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={doUpload}
                className="px-3 py-1.5 bg-primary text-primary-foreground rounded-lg text-xs font-semibold hover:shadow-lg hover:shadow-primary/20 transition-all"
              >
                Upload & Activate
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
