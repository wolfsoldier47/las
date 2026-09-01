import { useEffect, useState } from 'react'
import api from '../api/client'

interface ScanSchedule {
  id: string
  name: string
  frequency: 'daily' | 'weekly' | 'monthly'
  limit: string
  enabled: boolean
  next_run_at: string | null
  last_run_at: string | null
  created_by: string
  created_at: string
}

interface ScanScheduleRun {
  id: string
  schedule_id: string
  scan_job_id: string | null
  status: 'success' | 'failed'
  error_message: string
  started_at: string
  created_at: string
}

interface FormData {
  name: string
  frequency: 'daily' | 'weekly' | 'monthly'
  limit: string
  enabled: boolean
  start_at: string
}

const emptyForm: FormData = {
  name: '',
  frequency: 'daily',
  limit: '',
  enabled: true,
  start_at: '',
}

function formatDate(value: string | null) {
  if (!value) return '—'
  return new Date(value).toLocaleString()
}

export default function SchedulesPage() {
  const [schedules, setSchedules] = useState<ScanSchedule[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState<FormData>(emptyForm)
  const [submitting, setSubmitting] = useState(false)
  const [expanded, setExpanded] = useState<string | null>(null)
  const [runs, setRuns] = useState<Record<string, ScanScheduleRun[]>>({})
  const [runsLoading, setRunsLoading] = useState<Record<string, boolean>>({})

  const fetchSchedules = () => {
    setLoading(true)
    api
      .get('/scan-schedules')
      .then((res) => setSchedules(res.data || []))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchSchedules()
  }, [])

  const toggleExpanded = (id: string) => {
    if (expanded === id) {
      setExpanded(null)
      return
    }
    setExpanded(id)
    if (!runs[id]) {
      setRunsLoading((prev) => ({ ...prev, [id]: true }))
      api
        .get(`/scan-schedules/${id}/runs`)
        .then((res) => {
          setRuns((prev) => ({ ...prev, [id]: res.data || [] }))
        })
        .catch((err) => setError(err.message))
        .finally(() => setRunsLoading((prev) => ({ ...prev, [id]: false })))
    }
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    setError('')

    const payload: Record<string, unknown> = {
      name: form.name,
      frequency: form.frequency,
      limit: form.limit,
      enabled: form.enabled,
    }
    if (form.start_at) {
      payload.start_at = new Date(form.start_at).toISOString()
    }

    api
      .post('/scan-schedules', payload)
      .then(() => {
        setForm(emptyForm)
        setShowForm(false)
        fetchSchedules()
      })
      .catch((err) => setError(err.response?.data?.error || err.message))
      .finally(() => setSubmitting(false))
  }

  const toggleEnabled = (schedule: ScanSchedule) => {
    api
      .put(`/scan-schedules/${schedule.id}`, {
        name: schedule.name,
        frequency: schedule.frequency,
        limit: schedule.limit,
        enabled: !schedule.enabled,
        next_run_at: schedule.next_run_at,
      })
      .then(() => fetchSchedules())
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const handleDelete = (id: string) => {
    if (!confirm('Are you sure you want to delete this schedule?')) return
    api
      .delete(`/scan-schedules/${id}`)
      .then(() => fetchSchedules())
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex justify-between items-center">
        <div>
          <div className="font-semibold text-sm text-foreground">Scan Schedules</div>
          <div className="text-xs text-muted-foreground mt-0.5">
            Create recurring scans that run daily, weekly, or monthly.
          </div>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="px-3 py-1.5 bg-primary text-primary-foreground rounded-md text-xs font-semibold hover:shadow-lg hover:shadow-primary/20 transition-all"
        >
          + New Schedule
        </button>
      </div>

      {error && <div className="text-xs text-red-500">{error}</div>}

      {showForm && (
        <div className="bg-card border border-border rounded-xl p-5 flex flex-col gap-4">
          <div className="font-semibold text-sm text-foreground">New Schedule</div>
          <form onSubmit={handleSubmit} className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-xs text-muted-foreground">Name</label>
              <input
                type="text"
                required
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="e.g. Weekly RHEL 8 scan"
                className="bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-xs text-muted-foreground">Frequency</label>
              <select
                value={form.frequency}
                onChange={(e) => setForm({ ...form, frequency: e.target.value as FormData['frequency'] })}
                className="bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              >
                <option value="daily">Daily</option>
                <option value="weekly">Weekly</option>
                <option value="monthly">Monthly</option>
              </select>
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-xs text-muted-foreground">Host limit (optional)</label>
              <input
                type="text"
                value={form.limit}
                onChange={(e) => setForm({ ...form, limit: e.target.value })}
                placeholder="e.g. host-*-prod"
                className="bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-xs text-muted-foreground">Start at (optional)</label>
              <input
                type="datetime-local"
                value={form.start_at}
                onChange={(e) => setForm({ ...form, start_at: e.target.value })}
                className="bg-background border border-border rounded-lg px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/15"
              />
            </div>
            <label className="flex items-center gap-2 text-sm text-foreground">
              <input
                type="checkbox"
                checked={form.enabled}
                onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
                className="rounded border-border bg-background text-primary focus:ring-primary"
              />
              Enabled
            </label>
            <div className="flex justify-end gap-2 sm:col-span-2">
              <button
                type="button"
                onClick={() => {
                  setShowForm(false)
                  setForm(emptyForm)
                }}
                className="px-3 py-1.5 bg-secondary border border-[#3f3f46] rounded-md text-xs text-foreground hover:bg-[#3f3f46] transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={submitting}
                className="px-3 py-1.5 bg-primary text-primary-foreground rounded-md text-xs font-semibold hover:shadow-lg hover:shadow-primary/20 transition-all disabled:opacity-50"
              >
                {submitting ? 'Saving...' : 'Save Schedule'}
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border/50">
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Name</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Frequency</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Limit</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Enabled</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Next Run</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Last Run</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Created By</th>
                <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground"></th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={8} className="px-5 py-8 text-center text-muted-foreground text-sm">Loading...</td>
                </tr>
              ) : schedules.length === 0 ? (
                <tr>
                  <td colSpan={8} className="px-5 py-8 text-center text-muted-foreground text-sm">
                    No schedules yet. Click <strong>New Schedule</strong> to create one.
                  </td>
                </tr>
              ) : (
                schedules.map((schedule) => (
                  <>
                    <tr key={schedule.id} className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]">
                      <td className="px-5 py-3">
                        <button
                          onClick={() => toggleExpanded(schedule.id)}
                          className="text-foreground font-medium text-xs hover:text-primary flex items-center gap-1"
                        >
                          <span
                            className={`transition-transform ${expanded === schedule.id ? 'rotate-90' : ''}`}
                          >
                            ▶
                          </span>
                          {schedule.name}
                        </button>
                      </td>
                      <td className="px-5 py-3 text-muted-foreground text-xs capitalize">{schedule.frequency}</td>
                      <td className="px-5 py-3 text-muted-foreground text-xs font-mono">{schedule.limit || '—'}</td>
                      <td className="px-5 py-3">
                        <button
                          onClick={() => toggleEnabled(schedule)}
                          className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                            schedule.enabled ? 'bg-primary' : 'bg-secondary'
                          }`}
                        >
                          <span
                            className={`inline-block h-3.5 w-3.5 transform rounded-full bg-background transition-transform ${
                              schedule.enabled ? 'translate-x-5' : 'translate-x-1'
                            }`}
                          />
                        </button>
                      </td>
                      <td className="px-5 py-3 text-muted-foreground text-xs">{formatDate(schedule.next_run_at)}</td>
                      <td className="px-5 py-3 text-muted-foreground text-xs">{formatDate(schedule.last_run_at)}</td>
                      <td className="px-5 py-3 text-muted-foreground text-xs">{schedule.created_by}</td>
                      <td className="px-5 py-3 text-right">
                        <button
                          onClick={() => handleDelete(schedule.id)}
                          className="text-xs text-red-500 hover:text-red-400 transition-colors"
                        >
                          Delete
                        </button>
                      </td>
                    </tr>
                    {expanded === schedule.id && (
                      <tr>
                        <td colSpan={8} className="px-5 py-0 border-b border-border/50">
                          <div className="py-4">
                            <div className="font-semibold text-xs text-foreground mb-2">Run History</div>
                            {runsLoading[schedule.id] ? (
                              <div className="text-xs text-muted-foreground">Loading...</div>
                            ) : (runs[schedule.id] || []).length === 0 ? (
                              <div className="text-xs text-muted-foreground">No runs recorded yet.</div>
                            ) : (
                              <table className="w-full text-xs">
                                <thead>
                                  <tr className="border-b border-border/50">
                                    <th className="py-2 text-left text-muted-foreground">Started At</th>
                                    <th className="py-2 text-left text-muted-foreground">Status</th>
                                    <th className="py-2 text-left text-muted-foreground">Scan Job</th>
                                    <th className="py-2 text-left text-muted-foreground">Error</th>
                                  </tr>
                                </thead>
                                <tbody>
                                  {(runs[schedule.id] || []).map((run) => (
                                    <tr key={run.id} className="border-b border-border/30">
                                      <td className="py-2 text-muted-foreground">{formatDate(run.started_at)}</td>
                                      <td className="py-2">
                                        <span
                                          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium border ${
                                            run.status === 'success'
                                              ? 'bg-green-500/10 text-green-500 border-green-500/15'
                                              : 'bg-red-500/10 text-red-500 border-red-500/15'
                                          }`}
                                        >
                                          {run.status === 'success' ? 'Success' : 'Failed'}
                                        </span>
                                      </td>
                                      <td className="py-2 text-muted-foreground font-mono">
                                        {run.scan_job_id ? run.scan_job_id.slice(0, 8) : '—'}
                                      </td>
                                      <td className="py-2 text-red-400 max-w-md truncate" title={run.error_message}>
                                        {run.error_message || '—'}
                                      </td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            )}
                          </div>
                        </td>
                      </tr>
                    )}
                  </>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
