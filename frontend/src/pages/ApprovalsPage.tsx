import { useEffect, useState } from 'react'
import api from '../api/client'
import { isAdmin } from '../auth/permission'
import { useAuth } from '../context/AuthContext'

interface PendingBaselineVersion {
  os_type: string
  file_type: string
  version: number
  created_by?: string
  created_at?: string
  entry_count?: number
}

interface PendingDeviation {
  id: string
  hostname: string
  file_type: string
  entry_key: string
  entry_value?: string
  created_by?: string
  created_at?: string
  approval_status?: string
}

export default function ApprovalsPage() {
  const { user } = useAuth()
  const username = (user?.username || localStorage.getItem('ulas_username') || '').toLowerCase()
  const admin = isAdmin()

  const [pendingVersions, setPendingVersions] = useState<PendingBaselineVersion[]>([])
  const [pendingDeviations, setPendingDeviations] = useState<PendingDeviation[]>([])
  const [error, setError] = useState('')

  const fetchPendingVersions = () => {
    api
      .get('/baselines/versions/pending')
      .then((res) => {
        const data: PendingBaselineVersion[] | { items?: PendingBaselineVersion[] } = res.data
        setPendingVersions(Array.isArray(data) ? data : data?.items || [])
      })
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const fetchPendingDeviations = () => {
    api
      .get('/deviations/pending')
      .then((res) => {
        const data: PendingDeviation[] | { items?: PendingDeviation[] } = res.data
        setPendingDeviations(Array.isArray(data) ? data : data?.items || [])
      })
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  useEffect(() => {
    fetchPendingVersions()
    fetchPendingDeviations()
  }, [])

  const approveVersion = (v: PendingBaselineVersion) => {
    setError('')
    api
      .post('/baselines/versions/approve', {
        os_type: v.os_type,
        file_type: v.file_type,
        version: v.version,
      })
      .then(() => fetchPendingVersions())
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const approveDeviation = (d: PendingDeviation) => {
    setError('')
    api
      .post(`/deviations/${d.id}/approve`)
      .then(() => fetchPendingDeviations())
      .catch((err) => setError(err.response?.data?.error || err.message))
  }

  const isSelf = (createdBy?: string) => !!createdBy && createdBy.toLowerCase() === username

  return (
    <div className="flex flex-col gap-6">
      {error && <div className="text-xs text-red-500">{error}</div>}

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <div className="font-semibold text-sm text-foreground">Pending Baseline Versions</div>
          <div className="text-xs text-muted-foreground mt-0.5">
            Newly uploaded master files awaiting a second administrator's approval before activation
          </div>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">OS Type</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">File</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Version</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Requested By</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Created</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground"></th>
              </tr>
            </thead>
            <tbody>
              {pendingVersions.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-5 py-8 text-center text-xs text-muted-foreground">
                    No baseline versions pending approval
                  </td>
                </tr>
              ) : (
                pendingVersions.map((v) => (
                  <tr
                    key={`${v.os_type}-${v.file_type}-${v.version}`}
                    className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]"
                  >
                    <td className="px-5 py-3 text-foreground text-xs">{v.os_type}</td>
                    <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{v.file_type}</td>
                    <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{v.version}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{v.created_by || '—'}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">
                      {v.created_at ? new Date(v.created_at).toLocaleString() : '—'}
                    </td>
                    <td className="px-5 py-3 text-right">
                      {admin &&
                        (isSelf(v.created_by) ? (
                          <span className="text-xs text-muted-foreground">Awaiting another approver</span>
                        ) : (
                          <button
                            onClick={() => approveVersion(v)}
                            className="px-2 py-1 border border-border rounded-lg text-xs hover:bg-secondary transition-all"
                          >
                            Approve
                          </button>
                        ))}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <div className="font-semibold text-sm text-foreground">Pending Allowed Deviations</div>
          <div className="text-xs text-muted-foreground mt-0.5">
            Newly registered deviations awaiting a second administrator's approval
          </div>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Hostname</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">File</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Entry Key</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Requested By</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Created</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground"></th>
              </tr>
            </thead>
            <tbody>
              {pendingDeviations.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-5 py-8 text-center text-xs text-muted-foreground">
                    No deviations pending approval
                  </td>
                </tr>
              ) : (
                pendingDeviations.map((d) => (
                  <tr key={d.id} className="border-b border-border/50 transition-colors hover:bg-primary/[0.03]">
                    <td className="px-5 py-3 text-foreground font-medium font-mono text-xs">{d.hostname}</td>
                    <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{d.file_type}</td>
                    <td className="px-5 py-3 text-muted-foreground font-mono text-xs">{d.entry_key}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">{d.created_by || '—'}</td>
                    <td className="px-5 py-3 text-muted-foreground text-xs">
                      {d.created_at ? new Date(d.created_at).toLocaleString() : '—'}
                    </td>
                    <td className="px-5 py-3 text-right">
                      {admin &&
                        (isSelf(d.created_by) ? (
                          <span className="text-xs text-muted-foreground">Awaiting another approver</span>
                        ) : (
                          <button
                            onClick={() => approveDeviation(d)}
                            className="px-2 py-1 border border-border rounded-lg text-xs hover:bg-secondary transition-all"
                          >
                            Approve
                          </button>
                        ))}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
