const OS_LABELS: Record<string, string> = {
  linux: 'Linux',
  solaris: 'Solaris',
  aix: 'AIX',
}

// OsBadge renders the target OS a scan job was started with
// (captured from the Target OS selector when Start Scan is pressed).
export default function OsBadge({ osType }: { osType?: string }) {
  if (!osType) return <span className="text-muted-foreground text-xs">-</span>
  const label = OS_LABELS[osType.toLowerCase()] ?? osType
  return (
    <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium border bg-secondary/50 text-muted-foreground border-border">
      {label}
    </span>
  )
}
