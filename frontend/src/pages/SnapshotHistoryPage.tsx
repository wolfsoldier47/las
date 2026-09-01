import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import api from '../api/client'

interface Snapshot {
  id: string
  scan_job_id: string
  raw_content: string
  line_count: number
  snapshot_at: string
  changed: boolean
}

interface Change {
  change_type: string
  previous_content?: string
  current_content?: string
  detected_at: string
}

export default function SnapshotHistoryPage() {
  const { hostId, fileType } = useParams<{ hostId: string; fileType: string }>()
  const [snapshots, setSnapshots] = useState<Snapshot[]>([])
  const [changes, setChanges] = useState<Change[]>([])
  const [error, setError] = useState('')
  const [viewing, setViewing] = useState<Snapshot | null>(null)
  const [viewContent, setViewContent] = useState('')
  const [viewLoading, setViewLoading] = useState(false)

  useEffect(() => {
    if (!hostId || !fileType) return
    api
      .get(`/snapshots/${hostId}/${fileType}/history`)
      .then((res) => setSnapshots(res.data || []))
      .catch((err) => setError(err.message))
    api
      .get(`/snapshots/${hostId}/${fileType}/changes`)
      .then((res) => setChanges(res.data || []))
      .catch((err) => setError(err.message))
  }, [hostId, fileType])

  const viewSnapshot = (snapshot: Snapshot) => {
    setViewing(snapshot)
    setViewContent('')
    setViewLoading(true)
    api
      .get(`/snapshots/detail/${snapshot.id}`)
      .then((res) => setViewContent(res.data?.raw_content || ''))
      .catch((err) => setError(err.response?.data?.error || err.message))
      .finally(() => setViewLoading(false))
  }

  if (error) return <p className="text-red-500 text-sm">{error}</p>

  return (
    <div className="flex flex-col gap-6">
      <div className="bg-card border border-border rounded-xl p-5">
        <div className="font-semibold text-sm text-foreground">Snapshot History</div>
        <div className="text-xs text-muted-foreground mt-0.5">Host: {hostId} | File: {fileType}</div>
      </div>

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <div className="font-semibold text-sm text-foreground">Changes</div>
          <div className="text-xs text-muted-foreground mt-0.5">{changes.length} changes detected</div>
        </div>
        {changes.length === 0 && <div className="px-5 py-4 text-sm text-muted-foreground">No changes detected.</div>}
        {changes.map((change, idx) => (
          <div key={idx} className="p-5 border-b border-border/50 last:border-b-0">
            <div className="flex items-center gap-2 mb-3">
              <span className="text-xs font-medium text-foreground">{change.change_type.toUpperCase()}</span>
              <span className="text-xs text-muted-foreground">{new Date(change.detected_at).toLocaleString()}</span>
            </div>
            {change.previous_content && (
              <div className="mb-3">
                <div className="text-xs text-muted-foreground mb-1">Previous</div>
                <pre className="bg-background border border-border rounded-lg p-3 text-xs font-mono text-red-400 overflow-auto">{change.previous_content}</pre>
              </div>
            )}
            {change.current_content && (
              <div>
                <div className="text-xs text-muted-foreground mb-1">Current</div>
                <pre className="bg-background border border-border rounded-lg p-3 text-xs font-mono text-green-400 overflow-auto">{change.current_content}</pre>
              </div>
            )}
          </div>
        ))}
      </div>

      <div className="bg-card border border-border rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-border">
          <div className="font-semibold text-sm text-foreground">Snapshots</div>
          <div className="text-xs text-muted-foreground mt-0.5">{snapshots.length} snapshots — click a row to view its content</div>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Scan Job</th>
                <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">Lines</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Changed</th>
                <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">Snapshot At</th>
              </tr>
            </thead>
            <tbody>
              {snapshots.map((snapshot, idx) => (
                <tr
                  key={idx}
                  onClick={() => viewSnapshot(snapshot)}
                  className="border-b border-border/50 transition-colors hover:bg-primary/[0.03] cursor-pointer"
                >
                  <td className="px-5 py-3 text-foreground font-mono text-xs">{snapshot.scan_job_id.slice(0, 8)}</td>
                  <td className="px-5 py-3 text-right text-muted-foreground text-xs">{snapshot.line_count}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{snapshot.changed ? 'Yes' : 'No'}</td>
                  <td className="px-5 py-3 text-muted-foreground text-xs">{new Date(snapshot.snapshot_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {viewing && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          onClick={() => setViewing(null)}
        >
          <div
            className="bg-card border border-border rounded-xl max-w-4xl w-full max-h-[85vh] flex flex-col shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="px-5 py-4 border-b border-border flex justify-between items-center">
              <div>
                <div className="font-semibold text-sm text-foreground">
                  {fileType} snapshot — {new Date(viewing.snapshot_at).toLocaleString()}
                </div>
                <div className="text-xs text-muted-foreground">Scan job {viewing.scan_job_id.slice(0, 8)} • {viewing.line_count} lines</div>
              </div>
              <button
                onClick={() => setViewing(null)}
                className="text-muted-foreground hover:text-foreground text-lg leading-none"
              >
                ×
              </button>
            </div>
            <div className="p-5 overflow-auto flex-1">
              {viewLoading ? (
                <div className="text-xs text-muted-foreground">Loading…</div>
              ) : (
                <pre className="bg-background border border-border rounded-lg p-3 text-xs font-mono text-foreground whitespace-pre-wrap">
                  {viewContent}
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
    </div>
  )
}
