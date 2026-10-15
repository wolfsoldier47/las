import { useEffect, useState } from 'react'
import api from '../api/client'
import { isAdmin } from '../auth/permission'

interface BaselineVersion {
  os_type: string
  file_type: string
  version: number
  is_active: boolean
  entry_count: number
  description?: string
  created_by?: string
  created_at: string
  approval_status?: 'pending' | 'approved' | 'rejected'
  approved_by?: string
  approved_at?: string
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
  check_ids?: boolean
  mandatory?: boolean
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
    privilege_list: '',
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
  // Parsed entries of the pasted content, shown in the confirm dialog so the
  // user can mark keys as privileged with checkboxes instead of typing a list.
  const [parsedEntries, setParsedEntries] = useState<{ key: string; value: string }[]>([])
  const [privSelection, setPrivSelection] = useState<Record<string, boolean>>({})
  const [mandSelection, setMandSelection] = useState<Record<string, boolean>>({})
  const [privFilter, setPrivFilter] = useState('')

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

  // parseContent splits pasted master file text into key/value entries. The
  // key is everything before the first colon; lines without a colon use the
  // whole line as the key.
  const parseContent = (content: string): { key: string; value: string }[] => {
    const entries: { key: string; value: string }[] = []
    for (const line of content.split('\n')) {
      const trimmed = line.trim()
      if (!trimmed) continue
      const idx = trimmed.indexOf(':')
      if (idx === -1) {
        entries.push({ key: trimmed, value: '' })
      } else {
        entries.push({ key: trimmed.slice(0, idx), value: trimmed.slice(idx + 1) })
      }
    }
    return entries
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSuccess('')

    const entries = parseContent(form.content)
    if (entries.length === 0) {
      setError('File contents are empty — paste the master file first.')
      return
    }

    // Pre-check keys the user may already have listed manually.
    const preChecked: Record<string, boolean> = {}
    for (const k of form.privilege_list.split(/[\s,]+/)) {
      if (k) preChecked[k] = true
    }
    // Every key starts mandatory by default, preserving the original
    // all-entries-required behavior unless the user opts out.
    const preMandatory: Record<string, boolean> = {}
    for (const entry of entries) preMandatory[entry.key] = true
    setParsedEntries(entries)
    setPrivSelection(preChecked)
    setMandSelection(preMandatory)
    setPrivFilter('')
    setConfirmUpload(true)
  }

  const doUpload = () => {
    setConfirmUpload(false)

    const privilegedKeys = parsedEntries
      .filter((entry) => privSelection[entry.key])
      .map((entry) => entry.key)
    const mandatoryKeys = parsedEntries
      .filter((entry) => mandSelection[entry.key])
      .map((entry) => entry.key)

    const payload = {
      os_type: form.os_type,
      file_type: form.file_type,
      version: Number(form.version),
      content: form.content,
      privilege_list: privilegedKeys.join(','),
      mandatory_list: mandatoryKeys.join(','),
      description: form.description,
    }

    api
      .post('/baselines/upload', payload)
      .then((res) => {
        setSuccess(`Created baseline for ${form.os_type} version ${res.data.version}`)
        setForm({ os_type: 'linux', file_type: 'passwd', version: String(availableVersions[0] || ''), content: '', privilege_list: '', description: '' })
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
  const admin = isAdmin()

  return (
    <div className="flex flex-col gap-6">
      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <div className="font-semibold text-sm text-foreground">Upload Master File</div>
          <div className="text-xs text-muted-foreground mt-0.5">
            Paste the full contents of /etc/passwd or /etc/group. The selected OS major version is used as the baseline version; only one baseline per OS major version can be active at a time.
          </div>
        </div>
        {admin ? (
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
          <div className="col-span-2">
            <div className="text-[11px] text-muted-foreground">
              After pressing Upload you will pick which entry keys are privileged (uid/gid compared) via checkboxes.
            </div>
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
        ) : (
          <div className="p-5 text-xs text-muted-foreground">
            Read-only access — only administrators can upload master files.
          </div>
        )}
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
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Approved By</th>
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
                    {v.approval_status === 'pending' ? (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border bg-amber-500/10 text-amber-500 border-amber-500/15">
                        Pending Approval
                      </span>
                    ) : v.approval_status === 'rejected' ? (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border bg-red-500/10 text-red-500 border-red-500/15">
                        Rejected
                      </span>
                    ) : v.is_active ? (
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
                  <td className="px-5 py-3 text-muted-foreground text-xs">{v.approved_by || '—'}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{new Date(v.created_at).toLocaleString()}</td>
                  <td className="px-5 py-3 text-right">
                    <div className="flex gap-2 justify-end">
                      <button
                        onClick={() => viewVersionContent(v)}
                        className="px-2 py-1 bg-background border border-border text-foreground rounded text-xs hover:bg-primary/[0.05] transition-colors"
                      >
                        View
                      </button>
                      {admin && !v.is_active && v.approval_status === 'approved' && (
                        <button
                          onClick={() => activateVersion(v)}
                          className="px-2 py-1 bg-primary text-primary-foreground rounded text-xs font-semibold hover:shadow-lg hover:shadow-primary/20 transition-all"
                        >
                          Activate
                        </button>
                      )}
                      {admin && v.is_active && (
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
                <div className="font-semibold text-sm text-foreground flex items-center gap-2">
                  {viewing.version.file_type === 'passwd' ? '/etc/passwd' : '/etc/group'} — Version {viewing.version.version}
                  {viewing.version.approval_status === 'pending' && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border bg-amber-500/10 text-amber-500 border-amber-500/15">
                      Pending Approval
                    </span>
                  )}
                  {viewing.version.approval_status === 'rejected' && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium border bg-red-500/10 text-red-500 border-red-500/15">
                      Rejected
                    </span>
                  )}
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
              {viewing.version.approval_status === 'pending' && (
                <div className="mb-3 bg-amber-500/10 border border-amber-500/15 rounded-lg px-3 py-2 text-xs text-amber-500">
                  This version is awaiting approval. It stays inactive until another admin approves it on the
                  Approvals page.
                </div>
              )}
              {viewing.version.approval_status === 'rejected' && (
                <div className="mb-3 bg-red-500/10 border border-red-500/15 rounded-lg px-3 py-2 text-xs text-red-500">
                  This version was rejected. It can be approved later from the Approvals page if needed.
                </div>
              )}
              {viewing.loading ? (
                <div className="text-xs text-muted-foreground">Loading…</div>
              ) : viewing.entries.length === 0 ? (
                <div className="text-xs text-muted-foreground">No entries found.</div>
              ) : (
                <>
                  {(() => {
                    const privilegedKeys = viewing.entries.filter((e) => e.check_ids).map((e) => e.entry_key)
                    const optionalKeys = viewing.entries.filter((e) => e.mandatory === false).map((e) => e.entry_key)
                    if (privilegedKeys.length === 0 && optionalKeys.length === 0) return null
                    return (
                      <div className="mb-3 flex flex-col gap-2">
                        {privilegedKeys.length > 0 && (
                          <div className="bg-primary/[0.04] border border-primary/20 rounded-lg px-3 py-2">
                            <div className="text-[11px] font-medium text-primary mb-1">
                              Privilege List ({privilegedKeys.length}) — uid/gid compared for these entries
                            </div>
                            <div className="text-xs font-mono text-foreground break-words">
                              {privilegedKeys.join(', ')}
                            </div>
                          </div>
                        )}
                        {optionalKeys.length > 0 && (
                          <div className="bg-orange-500/[0.04] border border-orange-500/20 rounded-lg px-3 py-2">
                            <div className="text-[11px] font-medium text-orange-500 mb-1">
                              Optional Keys ({optionalKeys.length}) — may be absent on hosts; value must match when present
                            </div>
                            <div className="text-xs font-mono text-foreground break-words">
                              {optionalKeys.join(', ')}
                            </div>
                          </div>
                        )}
                      </div>
                    )
                  })()}
                  <pre className="bg-background border border-border rounded-lg p-3 text-xs font-mono text-foreground whitespace-pre-wrap">
                    {renderFileContent(viewing.version, viewing.entries)}
                  </pre>
                </>
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
            className="bg-card border border-border rounded-xl max-w-2xl w-full shadow-xl flex flex-col max-h-[85vh]"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="px-5 py-4 border-b border-border">
              <div className="font-semibold text-sm text-foreground">Confirm Upload</div>
              <div className="text-xs text-muted-foreground mt-0.5">
                Uploading creates a new version. It stays inactive until another admin approves it. Master file
                versions cannot be edited after creation.
              </div>
            </div>
            <div className="px-5 py-3 border-b border-border">
              <div className="text-xs font-medium text-foreground mb-2">
                Privileged keys (
                {parsedEntries.filter((entry) => privSelection[entry.key]).length} selected) · Mandatory keys (
                {parsedEntries.filter((entry) => mandSelection[entry.key]).length} selected)
              </div>
              <input
                type="text"
                value={privFilter}
                onChange={(e) => setPrivFilter(e.target.value)}
                placeholder="Filter keys..."
                className="w-full bg-background border border-border rounded-lg px-3 py-1.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              />
              <div className="text-[11px] text-muted-foreground mt-1.5">
                Privileged: checked keys get their uid/gid compared against the baseline; unchecked keys ignore
                uid/gid differences. Mandatory: checked keys must exist on every host; unchecked keys may be absent
                but must match the baseline value when present. Group members are always compared.
              </div>
            </div>
            <div className="overflow-auto flex-1">
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-card">
                  <tr className="border-b border-border">
                    <th className="px-5 py-2.5 text-left text-xs font-medium text-muted-foreground">Key</th>
                    <th className="px-5 py-2.5 text-left text-xs font-medium text-muted-foreground">Value</th>
                    <th className="px-5 py-2.5 text-center text-xs font-medium text-muted-foreground w-24">
                      <span className="inline-flex items-center gap-1">
                        Privileged
                        <span
                          className="inline-flex items-center justify-center w-4 h-4 rounded-full border border-border text-[10px] text-muted-foreground cursor-help select-none"
                          title="Privileged keys get their uid/gid compared against the baseline. Unchecked keys ignore uid/gid differences (the rest of the entry is still compared). For /etc/group, members are always compared regardless of this setting. The gecos field of /etc/passwd is always ignored — it's a free-text comment."
                        >
                          ?
                        </span>
                      </span>
                    </th>
                    <th className="px-5 py-2.5 text-center text-xs font-medium text-muted-foreground w-24">
                      <span className="inline-flex items-center gap-1">
                        Mandatory
                        <span
                          className="inline-flex items-center justify-center w-4 h-4 rounded-full border border-border text-[10px] text-muted-foreground cursor-help select-none"
                          title="Mandatory keys must be present on every scanned host; a missing mandatory key raises a deviation. Unchecked (optional) keys may be absent without a deviation, but if they are present their value must match the baseline — including uid/gid when the key is also privileged."
                        >
                          ?
                        </span>
                      </span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {parsedEntries
                    .filter((entry) => !privFilter || entry.key.toLowerCase().includes(privFilter.toLowerCase()))
                    .map((entry) => (
                      <tr key={entry.key} className="border-b border-border/50">
                        <td className="px-5 py-2 text-foreground font-mono text-xs">{entry.key}</td>
                        <td className="px-5 py-2 text-muted-foreground font-mono text-xs truncate max-w-xs" title={entry.value}>
                          {entry.value || '-'}
                        </td>
                        <td className="px-5 py-2 text-center">
                          <input
                            type="checkbox"
                            checked={!!privSelection[entry.key]}
                            onChange={(e) =>
                              setPrivSelection((prev) => ({ ...prev, [entry.key]: e.target.checked }))
                            }
                            className="rounded border-border bg-background text-primary focus:ring-primary"
                          />
                        </td>
                        <td className="px-5 py-2 text-center">
                          <input
                            type="checkbox"
                            checked={!!mandSelection[entry.key]}
                            onChange={(e) =>
                              setMandSelection((prev) => ({ ...prev, [entry.key]: e.target.checked }))
                            }
                            className="rounded border-border bg-background text-primary focus:ring-primary"
                          />
                        </td>
                      </tr>
                    ))}
                </tbody>
              </table>
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
                Upload
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
