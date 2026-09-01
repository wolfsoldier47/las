import { useEffect, useState } from 'react'
import { SectionCards } from '../components/SectionCards'
import { DataTable } from '../components/DataTable'
import { ScanDetail } from '../components/ScanDetail'
import api from '../api/client'

interface Incident {
  id: string
  file_type: 'passwd' | 'group'
  entry_key: string
  actual_value: string
  expected_value?: string
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
  deviations_found: number
  allowed_deviations: AllowedDeviation[]
  incidents: Incident[]
}

interface ScanJob {
  id: string
  status: string
}

interface ScanDetailData {
  job: ScanJob
  results: HostResult[]
}

function ActiveJobBanner({ job }: { job?: ScanJob }) {
  if (!job || (job.status !== 'running' && job.status !== 'initiating')) return null
  return (
    <div
      className="rounded-xl p-4 flex items-center justify-between border"
      style={{
        background: 'linear-gradient(135deg, rgba(250,204,21,0.08), rgba(250,204,21,0.02))',
        borderColor: 'rgba(250,204,21,0.2)',
      }}
    >
      <div className="flex items-center gap-3.5">
        <div
          className="w-9 h-9 rounded-xl flex items-center justify-center"
          style={{ background: 'rgba(250,204,21,0.15)' }}
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#facc15" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12 2v4"/><path d="m16.2 7.8 2.9-2.9"/><path d="M18 12h4"/><path d="m16.2 16.2 2.9 2.9"/><path d="M12 18v4"/><path d="m4.9 19.1 2.9-2.9"/><path d="M2 12h4"/><path d="m4.9 4.9 2.9 2.9"/>
          </svg>
        </div>
        <div>
          <div className="text-sm font-semibold text-primary">Scan Job {job.id?.slice(0, 8) || '—'} is Running</div>
          <div className="text-xs text-muted-foreground mt-0.5">Results will appear when the job completes</div>
        </div>
      </div>
    </div>
  )
}

export default function Dashboard() {
  const [selectedHost, setSelectedHost] = useState<string | undefined>()
  const [detail, setDetail] = useState<ScanDetailData | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get('/scans')
      .then((res) => {
        const scans: ScanJob[] = res.data || []
        const running = scans.find((s) => s.status === 'running' || s.status === 'initiating')
        const latest = running || scans[0]
        if (!latest) return
        return api.get(`/scans/${latest.id}?include_incidents=true`)
      })
      .then((res) => {
        if (res) {
          const data: ScanDetailData = res.data
          const results = (data.results || []).map((r) => ({
            ...r,
            incidents: r.incidents || [],
            allowed_deviations: r.allowed_deviations || [],
          }))
          setDetail({ ...data, results })
          if (results.length > 0) {
            setSelectedHost(results[0].hostname)
          }
        }
      })
      .catch((err) => setError(err.message))
  }, [])

  const selectedResult = (detail?.results || []).find((r) => r.hostname === selectedHost)

  return (
    <div className="flex flex-col gap-6">
      <SectionCards />
      <ActiveJobBanner job={detail?.job} />
      {error && <div className="text-xs text-red-500">{error}</div>}
      <div className="scan-results-export">
        <DataTable selectedHost={selectedHost} onSelectHost={setSelectedHost} />
      </div>
      {selectedResult && <ScanDetail host={selectedResult} />}
      <div className="text-center text-xs text-muted-foreground py-2">Ulas Compliance Engine v0.1.0</div>
    </div>
  )
}
