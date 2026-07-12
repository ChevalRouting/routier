
export type VRRPState = 'MASTER' | 'BACKUP' | 'FAULT' | 'UNKNOWN'
export type TabAction = React.ReactNode
export type OnActionChange = (a: TabAction) => void

export const PERIODS = [
  { label: '15m', minutes: 15 },
  { label: '30m', minutes: 30 },
  { label: '1h',  minutes: 60 },
  { label: '6h',  minutes: 360 },
  { label: '24h', minutes: 1440 },
  { label: '7d',  minutes: 10080 },
  { label: '30d', minutes: 43200 },
]
export const C = { rx: '#3584e4', tx: '#26a269' }

export function Legend({ items }: { items: { color: string; label: string }[] }) {
  return (
    <div className="flex items-center gap-3 text-xs text-muted-foreground">
      {items.map((i) => (
        <span key={i.label} className="flex items-center gap-1">
          <span className="h-2 w-2 rounded-full shrink-0" style={{ background: i.color }} />
          {i.label}
        </span>
      ))}
    </div>
  )
}

