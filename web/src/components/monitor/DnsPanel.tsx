import {
  type DnsStats
} from '@/lib/dnsApi'

type StatTileShape = { label: string; value: string }

export function StatTile({ label, value }: StatTileShape) {
  return (
    <div className="rounded-md bg-card px-3 py-2 shadow-[var(--card-shadow)]">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="font-mono text-sm">{value}</div>
    </div>
  )
}

export function statOf(stats: DnsStats | null, name: string): number {
  return (stats?.stats ?? []).find((s) => s.name === name)?.value ?? 0
}

export function hitRatio(stats: DnsStats | null): string {
  const hits = statOf(stats, 'cache hits')
  const misses = statOf(stats, 'cache misses')
  if (hits + misses === 0) return '-'

  return `${Math.round((hits / (hits + misses)) * 100)}%`
}

export { DnsPanel } from './DnsPanelParts/DnsPanel'
export { QueryTool } from './DnsPanelParts/QueryTool'
