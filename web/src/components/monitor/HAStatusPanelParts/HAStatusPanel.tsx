import type { TypesHAStatusResponse as HAStatusResponse } from '@/api'
import { VRRPInstanceCard } from '@/components/monitor/HAStatusPanel'
import { api } from '@/lib/client'
import { BreakdownRow, Card, CardContent, CardHeader, CardTitle, EmptyState, SectionLabel, Spinner, ReloadButton } from 'cheval-ui'
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
        <ReloadButton onClick={refresh} loading={loading} />
      </div>

      {error && <div className="rounded-md border border-danger/30 bg-danger/8 px-4 py-3 text-sm text-danger">{error}</div>}

      {loading && !data ? <Spinner /> : data ? (
        <div className="space-y-5">
          <div>
            <SectionLabel className="mb-2">VRRP Instances</SectionLabel>
            {(data.vrrp ?? []).length === 0
              ? <EmptyState className="py-10" title="No VRRP Instances" />
              : <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{(data.vrrp ?? []).map((inst) => <VRRPInstanceCard key={`${inst.interface}-${inst.id}`} inst={inst} />)}</div>
            }
          </div>
          {data.conntrackd ? (
            <Card>
              <CardHeader><CardTitle className="text-base">Connection Tracking (conntrackd)</CardTitle></CardHeader>
              <CardContent className="space-y-5">
                <div className="flex items-center gap-6">
                  <div className="flex items-center gap-2">
                    <span className={`h-2 w-2 rounded-full inline-block ${data.conntrackd.running ? 'bg-success' : 'bg-muted-foreground'}`} />
                    <span className="text-sm font-medium">{data.conntrackd.running ? 'Running' : 'Stopped'}</span>
                  </div>
                  {data.conntrackd.running && <div className="text-sm text-muted-foreground"><span className="font-mono font-medium text-foreground">{data.conntrackd.entries.toLocaleString()}</span> tracked connections</div>}
                </div>
                {Object.entries(data.conntrackd.by_proto ?? {}).sort((a, b) => b[1] - a[1]).length > 0 && (
                  <div className="space-y-2">
                    <p className="text-xs font-semibold text-muted-foreground">By Protocol</p>
                    {Object.entries(data.conntrackd.by_proto ?? {}).sort((a, b) => b[1] - a[1]).map(([proto, count]) => <BreakdownRow key={proto} label={proto.toUpperCase()} count={count} total={data.conntrackd!.entries} colorClass={'bg-primary'} />)}
                  </div>
                )}
                {Object.entries(data.conntrackd.tcp_states ?? {}).sort((a, b) => b[1] - a[1]).length > 0 && (
                  <div className="space-y-2">
                    <p className="text-xs font-semibold text-muted-foreground">TCP states</p>
                    {Object.entries(data.conntrackd.tcp_states ?? {}).sort((a, b) => b[1] - a[1]).map(([state, count]) => <BreakdownRow key={state} label={state} count={count} total={Object.values(data.conntrackd!.tcp_states ?? {}).reduce((s, n) => s + n, 0)} colorClass={'bg-primary'} />)}
                  </div>
                )}
              </CardContent>
            </Card>
          ) : (
            <Card><CardHeader><CardTitle className="text-base">Connection Tracking (conntrackd)</CardTitle></CardHeader>
              <CardContent><p className="text-sm text-muted-foreground">Not configured</p></CardContent>
            </Card>
          )}
        </div>
      ) : null}
    </div>
  )
}
