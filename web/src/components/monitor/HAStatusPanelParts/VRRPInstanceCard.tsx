import type { TypesVRRPInstanceStatus as VRRPInstanceStatus } from '@/api'
import { Card, CardContent, CardHeader, CardTitle, StateChip } from 'cheval-ui'

type VRRPInstanceCardShape = { inst: VRRPInstanceStatus }

export function VRRPInstanceCard({ inst }: VRRPInstanceCardShape) {
  const key = `VI_${inst.interface}_${inst.id}`
  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2 min-w-0">
            {inst.name ? (<><CardTitle className="text-base font-bold truncate">{inst.name}</CardTitle><span className="text-xs text-muted-foreground font-mono shrink-0">{key}</span></>) : <CardTitle className="text-base font-mono">{key}</CardTitle>}
          </div>
          <StateChip state={inst.state} />
        </div>
      </CardHeader>
      <CardContent className="space-y-2 text-sm">
        <div className="flex flex-wrap gap-x-5 gap-y-1 text-xs text-muted-foreground">
          <span><span className="font-medium text-foreground">Interface</span>{' '}<span className="font-mono">{inst.interface}</span></span>
          <span><span className="font-medium text-foreground">VRID</span>{' '}{inst.id}</span>
          <span><span className="font-medium text-foreground">Priority</span>{' '}{inst.priority || '-'}</span>
        </div>
        {(inst.vips ?? []).length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {(inst.vips ?? []).map((vip) => <span key={vip} className="inline-block px-1.5 py-0.5 rounded bg-muted font-mono text-[11px]">{vip}</span>)}
          </div>
        )}
        {inst.master_ip && inst.master_ip !== 'this system' && (
          <p className="text-xs text-muted-foreground"><span className="font-medium text-foreground">Master</span>{' '}<span className="font-mono">{inst.master_ip}</span></p>
        )}
        {(inst.peers ?? []).length > 0 && (
          <div className="rounded-md bg-muted/30 p-3">
            <p className="text-xs font-medium text-muted-foreground mb-1.5">Active peers in group</p>
            <ul className="space-y-0.5">
              {inst.peers!.map((peer) => (
                <li key={peer.ip} className="flex items-center gap-3 font-mono text-xs text-muted-foreground">
                  <span>{peer.ip}</span><span className="text-muted-foreground/60">prio {peer.priority}</span>
                  {peer.last_seen && <span className="text-muted-foreground/50">(seen {peer.last_seen} ago)</span>}
                </li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
