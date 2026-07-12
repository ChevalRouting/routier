import { useState, useEffect, useCallback } from 'react'
import { api } from '@/lib/client'
import type { TypesHAStatusResponse as HAStatusResponse, TypesVRRPInstanceStatus as VRRPInstanceStatus } from '@/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { StateChip } from '@/components/ui/status-chip'
import { SectionLabel } from '@/components/SectionLabel'
import { Button } from '@/components/ui/button'
import { RefreshCw } from 'lucide-react'
import { Spinner } from '@/components/Spinner'
import { EmptyState } from '@/components/EmptyState'

export const PROTO_COLORS: Record<string, string> = { tcp: 'bg-blue-500', udp: 'bg-amber-500', icmp: 'bg-green-500', icmpv6: 'bg-emerald-500' }
export const STATE_COLORS: Record<string, string> = { ESTABLISHED: 'bg-green-500', TIME_WAIT: 'bg-amber-400', CLOSE_WAIT: 'bg-orange-400', FIN_WAIT: 'bg-orange-500', SYN_SENT: 'bg-blue-400', SYN_RECV: 'bg-blue-500', LAST_ACK: 'bg-amber-500', CLOSE: 'bg-red-500' }

export function VRRPInstanceCard({ inst }: { inst: VRRPInstanceStatus }) {
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
          <div className="pt-2 border-t border-border/50">
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

export function BreakdownRow({ label, count, total, colorClass }: { label: string; count: number; total: number; colorClass: string }) {
  const pct = total > 0 ? (count / total) * 100 : 0
  return (
    <div className="grid grid-cols-[110px_1fr_110px] items-center gap-3">
      <span className="font-mono text-xs text-right truncate">{label}</span>
      <div className="h-1.5 rounded-full bg-muted overflow-hidden">
        <div className={`h-full rounded-full transition-all ${colorClass}`} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs text-muted-foreground text-right font-mono">{count.toLocaleString()} <span className="text-muted-foreground/60">({pct.toFixed(1)}%)</span></span>
    </div>
  )
}

export function HAStatusPanel() {
  const [data, setData] = useState<HAStatusResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)

  const refresh = useCallback(async () => {
    try { const result = await api.apiHaStatusGet(); setData(result); setError(null); setLastUpdated(new Date()) }
    catch (e) { setError((e as Error).message) }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { refresh(); const id = setInterval(refresh, 5000); return () => clearInterval(id) }, [refresh])

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        {lastUpdated && <span className="text-xs text-muted-foreground">{lastUpdated.toLocaleTimeString()}</span>}
        <Button variant="outline" size="sm" onClick={refresh} disabled={loading} className="gap-1.5 ml-auto">
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />Refresh
        </Button>
      </div>

      {error && <div className="rounded-md border border-danger/30 bg-danger/8 px-4 py-3 text-sm text-danger">{error}</div>}

      {loading && !data ? <Spinner /> : data ? (
        <div className="space-y-5">
          <div>
            <SectionLabel className="mb-2">VRRP Instances</SectionLabel>
            {(data.vrrp ?? []).length === 0
              ? <EmptyState className="py-10" title="No VRRP instances" message="Define VRRP on an interface to create a failover group." />
              : <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{(data.vrrp ?? []).map((inst) => <VRRPInstanceCard key={`${inst.interface}-${inst.id}`} inst={inst} />)}</div>
            }
          </div>
          {data.conntrackd ? (
            <Card>
              <CardHeader><CardTitle className="text-base">Connection Tracking (conntrackd)</CardTitle></CardHeader>
              <CardContent className="space-y-5">
                <div className="flex items-center gap-6">
                  <div className="flex items-center gap-2">
                    <span className={`h-2 w-2 rounded-full inline-block ${data.conntrackd.running ? 'bg-success animate-pulse' : 'bg-muted-foreground'}`} />
                    <span className="text-sm font-medium">{data.conntrackd.running ? 'Running' : 'Stopped'}</span>
                  </div>
                  {data.conntrackd.running && <div className="text-sm text-muted-foreground"><span className="font-mono font-medium text-foreground">{data.conntrackd.entries.toLocaleString()}</span> tracked connections</div>}
                </div>
                {Object.entries(data.conntrackd.by_proto ?? {}).sort((a, b) => b[1] - a[1]).length > 0 && (
                  <div className="space-y-2">
                    <p className="text-xs font-semibold text-muted-foreground">By protocol</p>
                    {Object.entries(data.conntrackd.by_proto ?? {}).sort((a, b) => b[1] - a[1]).map(([proto, count]) => <BreakdownRow key={proto} label={proto.toUpperCase()} count={count} total={data.conntrackd!.entries} colorClass={PROTO_COLORS[proto] ?? 'bg-zinc-400'} />)}
                  </div>
                )}
                {Object.entries(data.conntrackd.tcp_states ?? {}).sort((a, b) => b[1] - a[1]).length > 0 && (
                  <div className="space-y-2">
                    <p className="text-xs font-semibold text-muted-foreground">TCP states</p>
                    {Object.entries(data.conntrackd.tcp_states ?? {}).sort((a, b) => b[1] - a[1]).map(([state, count]) => <BreakdownRow key={state} label={state} count={count} total={Object.values(data.conntrackd!.tcp_states ?? {}).reduce((s, n) => s + n, 0)} colorClass={STATE_COLORS[state] ?? 'bg-zinc-400'} />)}
                  </div>
                )}
              </CardContent>
            </Card>
          ) : (
            <Card><CardHeader><CardTitle className="text-base">Connection Tracking (conntrackd)</CardTitle></CardHeader>
              <CardContent><p className="text-sm text-muted-foreground italic">conntrackd not configured, enable it on the Conntrackd config page.</p></CardContent>
            </Card>
          )}
        </div>
      ) : null}
    </div>
  )
}

