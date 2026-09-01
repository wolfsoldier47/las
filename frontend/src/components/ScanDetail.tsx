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

interface ScanDetailProps {
  host?: HostResult
}

export function ScanDetail({ host }: ScanDetailProps) {
  if (!host) return null

  const incidents = host.incidents || []
  const allowedDeviations = host.allowed_deviations || []

  const statusConfig: Record<string, { bg: string; border: string; text: string; label: string }> = {
    deviation_found: { bg: 'bg-red-500/10', border: 'border-red-500/15', text: 'text-red-500', label: '⚠️ Deviation Found' },
    allowed_deviation: { bg: 'bg-primary/10', border: 'border-primary/15', text: 'text-primary', label: '✓ Allowed' },
    success: { bg: 'bg-green-500/10', border: 'border-green-500/15', text: 'text-green-500', label: '✓ Clean' },
    failed: { bg: 'bg-red-500/10', border: 'border-red-500/15', text: 'text-red-500', label: '⚠️ Failed' },
    pending: { bg: 'bg-primary/10', border: 'border-primary/15', text: 'text-primary', label: '⏳ Pending' },
  }

  const config = statusConfig[host.status] || statusConfig.pending

  const passwdIncidents = incidents.filter((i) => i.file_type === 'passwd')
  const groupIncidents = incidents.filter((i) => i.file_type === 'group')
  const passwdAllowed = allowedDeviations.filter((d) => d.file_type === 'passwd')
  const groupAllowed = allowedDeviations.filter((d) => d.file_type === 'group')

  return (
    <div className="bg-card border border-border rounded-xl p-5 flex flex-col gap-4">
      <div className="flex justify-between items-center">
        <div className="flex items-center gap-2.5">
          <span className="font-semibold text-sm text-foreground">{host.hostname}</span>
          <span className={`inline-flex items-center gap-1 px-2 py-0.5 ${config.bg} ${config.text} rounded text-[11px] font-medium border ${config.border}`}>
            {config.label}
          </span>
        </div>
        <span className="text-xs text-muted-foreground font-mono">Result {host.id.slice(0, 8)} • {host.deviations_found} deviations</span>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <FileDiffCard
          file="/etc/passwd"
          status={passwdIncidents.length > 0 ? 'mismatch' : passwdAllowed.length > 0 ? 'allowed' : 'match'}
          incidents={passwdIncidents}
          allowed={passwdAllowed}
        />
        <FileDiffCard
          file="/etc/group"
          status={groupIncidents.length > 0 ? 'mismatch' : groupAllowed.length > 0 ? 'allowed' : 'match'}
          incidents={groupIncidents}
          allowed={groupAllowed}
        />
      </div>

      <div className="p-3.5 bg-background border border-border rounded-lg">
        <div className="text-xs text-muted-foreground uppercase tracking-wider font-medium mb-3">Deviation Resolution Logic</div>
        <div className="flex items-center gap-2 flex-wrap">
          <FlowStep
            color={host.status === 'success' ? 'green' : 'yellow'}
            text={host.status === 'success' ? 'Host compliant' : 'Deviation detected'}
          />
          <span className="text-muted-foreground text-sm">→</span>
          <FlowStep
            color={host.status === 'allowed_deviation' ? 'yellow' : 'yellow'}
            text="Check allowed list"
          />
          <span className="text-muted-foreground text-sm">→</span>
          {host.status === 'allowed_deviation' ? (
            <>
              <FlowStep color="yellow" text="Found in allowed list" />
              <span className="text-muted-foreground text-sm">→</span>
              <div className="flex items-center gap-1.5 px-3 py-2 bg-primary/15 border border-primary/30 rounded-lg">
                <div className="w-1.5 h-1.5 bg-primary rounded-full shadow-[0_0_6px_rgba(250,204,21,0.4)]" />
                <span className="text-xs text-primary font-bold">WHITELISTED</span>
              </div>
            </>
          ) : host.status === 'success' ? (
            <>
              <FlowStep color="green" text="No deviations" />
              <span className="text-muted-foreground text-sm">→</span>
              <div className="flex items-center gap-1.5 px-3 py-2 bg-green-500/15 border border-green-500/30 rounded-lg">
                <div className="w-1.5 h-1.5 bg-green-500 rounded-full shadow-[0_0_6px_rgba(34,197,94,0.4)]" />
                <span className="text-xs text-green-500 font-bold">COMPLIANT</span>
              </div>
            </>
          ) : (
            <>
              <FlowStep color="red" text="Not in allowed list" />
              <span className="text-muted-foreground text-sm">→</span>
              <div className="flex items-center gap-1.5 px-3 py-2 bg-red-500/15 border border-red-500/30 rounded-lg">
                <div className="w-1.5 h-1.5 bg-red-500 rounded-full shadow-[0_0_6px_rgba(239,68,68,0.4)]" />
                <span className="text-xs text-red-500 font-bold">FLAGGED</span>
              </div>
            </>
          )}
        </div>
      </div>

      <div className="p-3 bg-background border border-border rounded-lg">
        <div className="text-xs text-muted-foreground uppercase tracking-wider font-medium mb-2">Summary</div>
        <div className="text-sm text-foreground">
          {incidents.length === 0 && allowedDeviations.length === 0
            ? 'Host is fully compliant with baseline.'
            : [
                ...incidents.map((i) => `${i.file_type} entry '${i.entry_key}' actual: ${i.actual_value}${i.expected_value ? `, expected: ${i.expected_value}` : ''}`),
                ...allowedDeviations.map((d) => `${d.file_type} entry '${d.entry_key}' allowed deviation (actual: ${d.actual_value}${d.expected_value ? `, expected: ${d.expected_value}` : ''})`),
              ].join('; ')}
        </div>
      </div>
    </div>
  )
}

function FileDiffCard({
  file,
  status,
  incidents,
  allowed,
}: {
  file: string
  status: 'match' | 'mismatch' | 'allowed'
  incidents: Incident[]
  allowed: AllowedDeviation[]
}) {
  const config = {
    match: { badge: 'bg-green-500/10 text-green-500 border-green-500/15', label: 'Match' },
    mismatch: { badge: 'bg-red-500/10 text-red-500 border-red-500/15', label: 'Mismatch' },
    allowed: { badge: 'bg-primary/10 text-primary border-primary/15', label: 'Allowed Deviation' },
  }
  const c = config[status]
  return (
    <div className="bg-background border border-border rounded-lg overflow-hidden">
      <div className="px-3.5 py-2.5 bg-card border-b border-border flex justify-between items-center">
        <span className="text-xs font-medium text-foreground font-mono">{file}</span>
        <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium border ${c.badge}`}>
          {c.label}
        </span>
      </div>
      <div className="p-3 font-mono text-[11px] leading-relaxed">
        {incidents.length === 0 && allowed.length === 0 ? (
          <div className="text-green-500">✓ No deviations</div>
        ) : (
          <>
            {incidents.map((incident) => (
              <div key={incident.id} className="inline-block px-1.5 py-0.5 rounded text-red-500 bg-red-500/5 border border-red-500/10 mb-1">
                {incident.entry_key}: {incident.actual_value} <span className="font-bold">← NOT IN MASTER</span>
              </div>
            ))}
            {allowed.map((d, idx) => (
              <div key={idx} className="inline-block px-1.5 py-0.5 rounded text-primary bg-primary/5 border border-primary/10 mb-1">
                {d.entry_key}: {d.actual_value} <span className="font-bold">← ALLOWED EXCEPTION</span>
              </div>
            ))}
          </>
        )}
      </div>
    </div>
  )
}

function FlowStep({ color, text }: { color: 'red' | 'yellow' | 'green'; text: string }) {
  const colors = {
    red: { bg: 'bg-red-500/10', border: 'border-red-500/20', dot: 'bg-red-500', text: 'text-red-400' },
    yellow: { bg: 'bg-primary/10', border: 'border-primary/20', dot: 'bg-primary', text: 'text-primary' },
    green: { bg: 'bg-green-500/10', border: 'border-green-500/20', dot: 'bg-green-500', text: 'text-green-400' },
  }
  const c = colors[color]
  return (
    <div className={`flex items-center gap-1.5 px-3 py-2 ${c.bg} border ${c.border} rounded-lg`}>
      <div className={`w-1.5 h-1.5 ${c.dot} rounded-full`} />
      <span className={`text-xs ${c.text} font-medium`}>{text}</span>
    </div>
  )
}
