import type { TypesHAStatusResponse as HAStatusResponse } from '@/api'
import { PROTO_COLORS, STATE_COLORS, VRRPInstanceCard } from '@/components/monitor/HAStatusPanel'
import { api } from '@/lib/client'
import { BreakdownRow, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, SectionLabel, Spinner } from 'cheval-ui'
import { RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'

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
