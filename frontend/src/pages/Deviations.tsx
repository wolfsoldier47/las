import { useEffect, useState } from 'react'
import api from '../api/client'

interface Deviation {
  id: string
  hostname: string
  file_type: string
  entry_key: string
  entry_value?: string
  justification: string
  approved_by: string
  approved_at: string
  is_active: boolean
  expires_at?: string
}

interface PaginatedDeviations {
  items: Deviation[]
  total: number
  page: number
  limit: number
  active_total: number
  inactive_total: number
}

const emptyForm = {
  hostname: '',
  file_type: 'passwd',
  entry_line: '',
  justification: '',
  approved_by: '',
  expires_at: '',
}

function buildEntryLine(d: Deviation): string {
  if (d.entry_value) {
    return `${d.entry_key}:${d.entry_value}`
  }
  return d.entry_key
}

export default function Deviations() {
  const [deviations, setDeviations] = useState<Deviation[]>([])
  const [error, setError] = useState('')
  const [form, setForm] = useState(emptyForm)

  const [editing, setEditing] = useState<Deviation | null>(null)
  const [editForm, setEditForm] = useState(emptyForm)
  const [deleting, setDeleting] = useState<Deviation | null>(null)

  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [totalDeviations, setTotalDeviations] = useState(0)
  const [activeTotal, setActiveTotal] = useState(0)
  const [inactiveTotal, setInactiveTotal] = useState(0)

  const fetchData = (nextPage = page, nextSize = pageSize, nextSearch = search) => {
    api
      .get(`/deviations?page=${nextPage}&limit=${nextSize}&search=${encodeURIComponent(nextSearch)}`)
      .then((res) => {
        const data: PaginatedDeviations | Deviation[] = res.data
        if (Array.isArray(data)) {
          setDeviations(data)
          setTotalDeviations(data.length)
          const active = data.filter((d) => d.is_active).length
          setActiveTotal(active)
          setInactiveTotal(data.length - active)
        } else {
          const normalized = data || {
            items: [],
            total: 0,
            page: 1,
            limit: nextSize,
            active_total: 0,
            inactive_total: 0,
          }
          setDeviations(normalized.items || [])
          setTotalDeviations(normalized.total || 0)
          setActiveTotal(normalized.active_total || 0)
          setInactiveTotal(normalized.inactive_total || 0)
        }
      })
      .catch((err) => setError(err.message))
  }

  useEffect(() => {
    fetchData()
  }, [page, pageSize, search])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const payload = {
      hostname: form.hostname,
      file_type: form.file_type,
      entry_line: form.entry_line,
      justification: form.justification,
      approved_by: form.approved_by,
      expires_at: form.expires_at || undefined,
    }
    api
      .post('/deviations', payload)
      .then(() => {
        setForm(emptyForm)
        setPage(1)
        fetchData(1, pageSize, search)
      })
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const toggleDeviation = (d: Deviation) => {
    const payload = {
      hostname: d.hostname,
      file_type: d.file_type,
      entry_line: buildEntryLine(d),
      justification: d.justification,
      approved_by: d.approved_by,
      expires_at: d.expires_at,
      is_active: !d.is_active,
    }
    api
      .put(`/deviations/${d.id}`, payload)
      .then(() => fetchData())
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const openEdit = (d: Deviation) => {
    setEditing(d)
    setEditForm({
      hostname: d.hostname,
      file_type: d.file_type,
      entry_line: buildEntryLine(d),
      justification: d.justification,
      approved_by: d.approved_by,
      expires_at: d.expires_at || '',
    })
  }

  const handleEditSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!editing) return
    const payload = {
      hostname: editForm.hostname,
      file_type: editForm.file_type,
      entry_line: editForm.entry_line,
      justification: editForm.justification,
      approved_by: editForm.approved_by,
      expires_at: editForm.expires_at || undefined,
      is_active: editing.is_active,
    }
    api
      .put(`/deviations/${editing.id}`, payload)
      .then(() => {
        setEditing(null)
        setEditForm(emptyForm)
        fetchData()
      })
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const handleDelete = () => {
    if (!deleting) return
    api
      .delete(`/deviations/${deleting.id}`)
      .then(() => {
        setDeleting(null)
        fetchData()
      })
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const totalPages = Math.max(1, Math.ceil(totalDeviations / pageSize))

  return (
    <div className="flex flex-col gap-6">
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-card border border-border rounded-xl p-5 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(239,68,68,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Total Deviations</div>
          <div className="text-[28px] font-bold text-red-500 tracking-tight">{totalDeviations}</div>
          <div className="text-xs text-muted-foreground">Registered exceptions</div>
        </div>
        <div className="bg-card border border-border rounded-xl p-5 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(34,197,94,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Active</div>
          <div className="text-[28px] font-bold text-green-500 tracking-tight">{activeTotal}</div>
          <div className="text-xs text-muted-foreground">Currently whitelisted</div>
        </div>
        <div className="bg-card border border-border rounded-xl p-5 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-20 h-20" style={{ background: 'radial-gradient(circle at top right, rgba(250,204,21,0.06), transparent 70%)' }} />
          <div className="text-xs text-muted-foreground font-medium">Inactive</div>
          <div className="text-[28px] font-bold text-primary tracking-tight">{inactiveTotal}</div>
          <div className="text-xs text-muted-foreground">Disabled entries</div>
        </div>
      </div>

      {error && <div className="text-xs text-red-500">{error}</div>}

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <div className="font-semibold text-sm text-foreground">Register Allowed Deviation</div>
          <div className="text-xs text-muted-foreground mt-0.5">Pre-approve a deviation for a specific host and file</div>
        </div>
        <form onSubmit={handleSubmit} className="p-5 grid grid-cols-3 gap-4">
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">Host</label>
            <input
              type="text"
              className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={form.hostname}
              onChange={(e) => setForm({ ...form, hostname: e.target.value })}
              placeholder="e.g., host001.example.com"
              required
            />
            <div className="text-[11px] text-muted-foreground mt-1">Enter the full hostname. The host does not need to exist in the inventory.</div>
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
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">Entry Line</label>
            <input
              type="text"
              className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={form.entry_line}
              onChange={(e) => setForm({ ...form, entry_line: e.target.value })}
              placeholder="e.g., root:x:0:0:root:/root:/bin/bash"
              required
            />
            <div className="text-[11px] text-muted-foreground mt-1">Paste the full /etc/passwd or /etc/group line. The key is parsed before the first colon.</div>
          </div>
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">Justification</label>
            <input
              type="text"
              className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={form.justification}
              onChange={(e) => setForm({ ...form, justification: e.target.value })}
              required
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-1">Approved By</label>
            <input
              type="text"
              className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              value={form.approved_by}
              onChange={(e) => setForm({ ...form, approved_by: e.target.value })}
              required
            />
          </div>
          <div className="col-span-3">
            <button
              type="submit"
              className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-semibold hover:shadow-lg hover:shadow-primary/20 transition-all"
            >
              Create Deviation
            </button>
          </div>
        </form>
      </div>

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
          <div>
            <div className="font-semibold text-sm text-foreground">Allowed Deviations</div>
            <div className="text-xs text-muted-foreground mt-0.5">Whitelisted entries that bypass deviation checks</div>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value)
                setPage(1)
              }}
              placeholder="Search by host..."
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
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Host</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">File</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Entry Line</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Justification</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Approved By</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Actions</th>
              </tr>
            </thead>
            <tbody>
              {deviations.map((d) => (
                <tr key={d.id} className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]">
                  <td className="px-5 py-3 text-foreground font-medium font-mono text-xs">{d.hostname}</td>
                  <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{d.file_type}</td>
                  <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{buildEntryLine(d)}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{d.justification}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{d.approved_by}</td>
                  <td className="px-5 py-3">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => toggleDeviation(d)}
                        className={`px-3 py-1 rounded-lg text-xs font-semibold transition-all ${
                          d.is_active
                            ? 'bg-orange-500/10 text-orange-500 border border-orange-500/15 hover:bg-orange-500/20'
                            : 'bg-green-500/10 text-green-500 border border-green-500/15 hover:bg-green-500/20'
                        }`}
                      >
                        {d.is_active ? 'Disable' : 'Enable'}
                      </button>
                      <button
                        onClick={() => openEdit(d)}
                        className="px-3 py-1 rounded-lg text-xs font-semibold bg-primary/10 text-primary border border-primary/15 hover:bg-primary/20 transition-all"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => setDeleting(d)}
                        className="px-3 py-1 rounded-lg text-xs font-semibold bg-red-500/10 text-red-500 border border-red-500/15 hover:bg-red-500/20 transition-all"
                      >
                        Delete
                      </button>
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

      {editing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-card border border-border rounded-xl w-full max-w-3xl max-h-[90vh] overflow-y-auto">
            <div className="px-5 py-4 border-b border-border flex justify-between items-center">
              <div>
                <div className="font-semibold text-sm text-foreground">Edit Allowed Deviation</div>
                <div className="text-xs text-muted-foreground">Update the deviation details</div>
              </div>
              <button onClick={() => setEditing(null)} className="text-muted-foreground hover:text-foreground text-lg">&times;</button>
            </div>
            <form onSubmit={handleEditSubmit} className="p-5 grid grid-cols-3 gap-4">
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">Host</label>
                <input
                  type="text"
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
                  value={editForm.hostname}
                  onChange={(e) => setEditForm({ ...editForm, hostname: e.target.value })}
                  required
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">File Type</label>
                <select
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
                  value={editForm.file_type}
                  onChange={(e) => setEditForm({ ...editForm, file_type: e.target.value })}
                >
                  <option value="passwd">passwd</option>
                  <option value="group">group</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">Entry Line</label>
                <input
                  type="text"
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
                  value={editForm.entry_line}
                  onChange={(e) => setEditForm({ ...editForm, entry_line: e.target.value })}
                  placeholder="e.g., root:x:0:0:root:/root:/bin/bash"
                  required
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">Justification</label>
                <input
                  type="text"
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
                  value={editForm.justification}
                  onChange={(e) => setEditForm({ ...editForm, justification: e.target.value })}
                  required
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">Approved By</label>
                <input
                  type="text"
                  className="w-full bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
                  value={editForm.approved_by}
                  onChange={(e) => setEditForm({ ...editForm, approved_by: e.target.value })}
                  required
                />
              </div>
              <div className="col-span-3 flex justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setEditing(null)}
                  className="px-4 py-2 rounded-lg text-sm font-semibold border border-border hover:bg-primary/[0.03] transition-all"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-semibold hover:shadow-lg hover:shadow-primary/20 transition-all"
                >
                  Save Changes
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {deleting && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-card border border-border rounded-xl w-full max-w-md">
            <div className="px-5 py-4 border-b border-border">
              <div className="font-semibold text-sm text-foreground">Delete Allowed Deviation</div>
            </div>
            <div className="p-5 text-sm text-muted-foreground">
              Are you sure you want to delete the deviation for <span className="font-medium text-foreground">{deleting.hostname}</span> —{' '}
              <span className="font-medium text-foreground">{deleting.file_type}</span> / <span className="font-medium text-foreground">{deleting.entry_key}</span>?
              This action cannot be undone.
            </div>
            <div className="px-5 py-4 border-t border-border flex justify-end gap-2">
              <button
                onClick={() => setDeleting(null)}
                className="px-4 py-2 rounded-lg text-sm font-semibold border border-border hover:bg-primary/[0.03] transition-all"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                className="px-4 py-2 bg-red-500 text-white rounded-lg text-sm font-semibold hover:bg-red-600 transition-all"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
