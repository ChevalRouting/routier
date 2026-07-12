import { useState, useEffect, useCallback, useMemo } from 'react'
import { api } from '@/lib/client'
import type {
  TypesHAStatusResponse as HAStatusResponse, TypesStatsResponse as StatsResponse,
  TypesSystemHistoryPoint as SystemHistoryPoint, TypesNeighborStat as NeighborStat,
} from '@/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { StateChip } from '@/components/ui/status-chip'
import { Button } from '@/components/ui/button'
import { RefreshCw, Pause, Play, Cpu, MemoryStick, Network } from 'lucide-react'
import { Spinner } from '@/components/Spinner'
import { TimeSeriesChart } from '@/components/TimeSeriesChart'
import { fmtBytes, fmtUptime } from '@/lib/fmt'
import { OnActionChange, VRRPState } from './shared'

export function SystemPanel({ onActionChange }: { onActionChange: OnActionChange }) {
  const [stats, setStats] = useState<StatsResponse | null>(null)
  const [prev, setPrev] = useState<StatsResponse | null>(null)
  const [paused, setPaused] = useState(false)
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null)
  const [haStatus, setHAStatus] = useState<HAStatusResponse | null>(null)
  const [sysHistory, setSysHistory] = useState<SystemHistoryPoint[]>([])
  const [neighborStats, setNeighborStats] = useState<NeighborStat[]>([])

  const fetchStats = useCallback(async () => {
    try {
      const data = await api.apiStatsGet()
      setPrev((p) => (stats !== null ? stats : p))
      setStats(data)
      setLastUpdate(new Date())
    } catch {}
  }, [stats])

  useEffect(() => { fetchStats() }, [])
  useEffect(() => {
    if (paused) return
    const id = setInterval(fetchStats, 2000)
    return () => clearInterval(id)
  }, [paused, fetchStats])
  useEffect(() => {
    const run = async () => { try { setHAStatus(await api.apiHaStatusGet()) } catch {} }
    run(); const id = setInterval(run, 5000); return () => clearInterval(id)
  }, [])
  useEffect(() => {
    const load = () => api.apiStatsHistoryGet({ minutes: 60, series: 'system' }).then((d) => setSysHistory(d.system ?? [])).catch(() => {})
    load(); const id = setInterval(load, 60_000); return () => clearInterval(id)
  }, [])
  useEffect(() => {
    const load = () => api.apiStatsNeighborsGet().then((d) => { setNeighborStats(d.neighbors ?? []) }).catch(() => {})
    load(); const id = setInterval(load, 60_000); return () => clearInterval(id)
  }, [])
  useEffect(() => {
    onActionChange(
      <div className="flex items-center gap-2">
        {lastUpdate && <span className="text-xs text-muted-foreground hidden sm:inline">{lastUpdate.toLocaleTimeString()}</span>}
        <Button variant="outline" size="sm" onClick={() => setPaused((p) => !p)} className="gap-1.5">
          {paused ? <Play className="h-3.5 w-3.5" /> : <Pause className="h-3.5 w-3.5" />}
          {paused ? 'Resume' : 'Pause'}
        </Button>
        <Button variant="outline" size="icon" className="h-8 w-8" onClick={fetchStats}>
          <RefreshCw className="h-3.5 w-3.5" />
        </Button>
      </div>
    )
  }, [paused, lastUpdate, fetchStats, onActionChange])

  const vrrpByIface = useMemo<Record<string, VRRPState>>(() => {
    const map: Record<string, VRRPState> = {}
    for (const inst of haStatus?.vrrp ?? []) {
      const prev = map[inst.interface]
      if (!prev || prev === 'UNKNOWN' || (prev !== 'MASTER' && inst.state === 'MASTER'))
        map[inst.interface] = inst.state as VRRPState
    }
    return map
  }, [haStatus])

  const rates: Record<string, { rx: number; tx: number }> = {}
  if (prev && stats) {
    for (const iface of Object.keys(stats.interfaces ?? {})) {
      const cur = stats.interfaces![iface]
      const prv = prev.interfaces?.[iface]
      if (prv) rates[iface] = { rx: Math.max(0, cur.rx_bytes - prv.rx_bytes) / 2, tx: Math.max(0, cur.tx_bytes - prv.tx_bytes) / 2 }
    }
  }

  const v4Reachable = neighborStats.filter((n) => n.family === 'ipv4' && n.state === 'REACHABLE').length
  const v4Total     = neighborStats.filter((n) => n.family === 'ipv4').length
  const v6Reachable = neighborStats.filter((n) => n.family === 'ipv6' && n.state === 'REACHABLE').length
  const v6Total     = neighborStats.filter((n) => n.family === 'ipv6').length

  const sys = stats?.system

  return (
    <div className="space-y-5">
      {!sys ? <Spinner /> : (
        <>
          <div className="grid gap-3 sm:grid-cols-2">
            <Card className="relative overflow-hidden">
              <div className="relative z-10 flex items-center justify-between px-5 pt-5 pb-1">
                <CardTitle className="text-sm font-medium">CPU</CardTitle>
                <Cpu className="h-4 w-4 text-muted-foreground" />
              </div>
              <div className="relative z-10 px-5 pb-7">
                <div className={`text-3xl font-bold ${sys.cpu_percent > 90 ? 'text-danger' : sys.cpu_percent > 70 ? 'text-warning' : 'text-primary'}`}>
                  {sys.cpu_percent.toFixed(1)}%
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">
                  load {sys.load1.toFixed(2)} / {sys.load5.toFixed(2)} / {sys.load15.toFixed(2)} · {sys.processes} proc · up {fmtUptime(sys.uptime_seconds)}
                </p>
              </div>
              <div className="absolute inset-x-0 bottom-0 h-24 opacity-25 pointer-events-none">
                <TimeSeriesChart mini data={sysHistory.map((p) => ({ ts: p.ts, cpu: p.cpu_pct }))} series={[{ dataKey: 'cpu', label: 'CPU %', color: '#3584e4' }]} height={96} />
              </div>
            </Card>

            <Card className="relative overflow-hidden">
              <div className="relative z-10 flex items-center justify-between px-5 pt-5 pb-1">
                <CardTitle className="text-sm font-medium">Memory</CardTitle>
                <MemoryStick className="h-4 w-4 text-muted-foreground" />
              </div>
              <div className="relative z-10 px-5 pb-7">
                <div className="text-3xl font-bold text-[#9141ac]">{fmtBytes(sys.mem_used)}</div>
                <p className="text-xs text-muted-foreground mt-0.5">
                  of {fmtBytes(sys.mem_total)}
                  {sys.swap_total > 0 && ` · swap ${fmtBytes(sys.swap_used)}/${fmtBytes(sys.swap_total)}`}
                  {' · '}buf {fmtBytes(sys.mem_buffers)}
                </p>
              </div>
              <div className="absolute inset-x-0 bottom-0 h-24 opacity-25 pointer-events-none">
                <TimeSeriesChart mini data={sysHistory.map((p) => ({ ts: p.ts, mem: p.mem_total > 0 ? (p.mem_used / p.mem_total) * 100 : 0 }))} series={[{ dataKey: 'mem', label: 'Mem %', color: '#9141ac' }]} height={96} />
              </div>
            </Card>
          </div>

          <div className="grid grid-cols-2 gap-3">
            {[
              { label: 'IPv4 Neighbors', total: v4Total, reachable: v4Reachable, color: 'text-primary' },
              { label: 'IPv6 Neighbors', total: v6Total, reachable: v6Reachable, color: 'text-[#26a269]' },
            ].map(({ label, total, reachable, color }) => (
              <Card key={label}>
                <CardContent className="px-5 py-4">
                  <p className="text-xs text-muted-foreground font-medium">{label}</p>
                  <p className={`text-3xl font-bold mt-1 ${color}`}>{total}</p>
                  {total > 0 && <p className="text-xs text-muted-foreground mt-0.5">{reachable} reachable</p>}
                  {total === 0 && <p className="text-xs text-muted-foreground mt-0.5 italic">no data yet</p>}
                </CardContent>
              </Card>
            ))}
          </div>

          <Card>
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <Network className="h-4 w-4 text-muted-foreground" />
                <CardTitle className="text-sm">Network Interfaces</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border text-xs text-muted-foreground">
                      <th className="text-left py-2 pr-4">Interface</th>
                      <th className="text-left py-2 pr-4">State</th>
                      <th className="text-right py-2 pr-4">RX Rate</th>
                      <th className="text-right py-2 pr-4">TX Rate</th>
                      <th className="text-right py-2 pr-4">RX Total</th>
                      <th className="text-right py-2 pr-4">TX Total</th>
                      <th className="text-right py-2">Errors</th>
                    </tr>
                  </thead>
                  <tbody>
                    {Object.entries(stats.interfaces ?? {}).sort(([a], [b]) => a.localeCompare(b)).map(([name, iface]) => (
                      <tr key={name} className="border-b border-border/50 hover:bg-muted/30">
                        <td className="py-2 pr-4 font-mono font-medium">{name}</td>
                        <td className="py-2 pr-4">
                          {vrrpByIface[name] ? (
                            <StateChip state={vrrpByIface[name]} />
                          ) : (
                            <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${iface.operstate === 'up' ? 'bg-success text-success-foreground' : 'bg-secondary text-secondary-foreground'}`}>
                              {iface.operstate}
                            </span>
                          )}
                        </td>
                        <td className="py-2 pr-4 text-right font-mono text-xs text-primary">{rates[name] ? fmtBytes(rates[name].rx) + '/s' : '-'}</td>
                        <td className="py-2 pr-4 text-right font-mono text-xs text-success">{rates[name] ? fmtBytes(rates[name].tx) + '/s' : '-'}</td>
                        <td className="py-2 pr-4 text-right font-mono text-xs text-muted-foreground">{fmtBytes(iface.rx_bytes)}</td>
                        <td className="py-2 pr-4 text-right font-mono text-xs text-muted-foreground">{fmtBytes(iface.tx_bytes)}</td>
                        <td className="py-2 text-right font-mono text-xs">
                          {iface.rx_errors + iface.tx_errors > 0
                            ? <span className="text-danger">{iface.rx_errors + iface.tx_errors}</span>
                            : <span className="text-muted-foreground">0</span>}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}

