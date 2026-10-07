import { VRRPInstance } from '@/components/ha/types'

type VRRPSummaryShape = { v: VRRPInstance }

export interface Row {
  uid: string
  v: VRRPInstance
}

export function emptyInstance(iface: string): VRRPInstance {
  return { name: '', id: 0, interface: iface, vips: [], priority: 100, password: '' }
}

export function VRRPSummary({ v }: VRRPSummaryShape) {
  return (
    <div className="flex min-w-0 flex-1 items-center gap-3">
      <span className="w-40 shrink-0 truncate text-sm font-medium">{v.name || `VRID ${v.id}`}</span>
      <span className="w-24 shrink-0 font-mono text-xs text-muted-foreground">{v.interface}</span>
      <span className="w-16 shrink-0 text-xs text-muted-foreground">VRID {v.id}</span>
      <span className="w-16 shrink-0 text-xs text-muted-foreground">prio {v.priority}</span>
      <div className="flex min-w-0 flex-1 flex-wrap gap-1">
        {(v.vips ?? []).map((vip) => (
          <span key={vip} className="inline-block rounded bg-muted px-1.5 py-0.5 font-mono text-[11px]">{vip}</span>
        ))}
        {v.friend && <span className="text-[11px] text-muted-foreground">from {v.friend}</span>}
      </div>
    </div>
  )
}

export { VRRPBody } from './VRRPTabParts/VRRPBody'
export { VRRPTab } from './VRRPTabParts/VRRPTab'
