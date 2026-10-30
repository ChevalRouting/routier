import { onActivate } from '@/lib/utils'
import type {
  TypesHAStatusResponse as HAStatusResponse,
  TypesNeighborStat as NeighborStat,
  TypesStatsResponse as StatsResponse,
  TypesSystemHistoryPoint as SystemHistoryPoint,
} from '@/api'
import { api } from '@/lib/client'
import { Button, Card, CardContent, CardHeader, CardTitle, fmtBytes, fmtUptime, Spinner, StateChip, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, ReloadButton } from 'cheval-ui'
import { TimeSeriesChart } from 'cheval-ui/charts'
import { Cpu, MemoryStick, Network, Pause, Play } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { InterfaceDetails } from './InterfaceDetails'
import { OnActionChange, VRRPState } from './shared'
import { SERIES } from '@/lib/palette'

type SystemPanelShape = { onActionChange: OnActionChange }

type RatesShape = { rx: number; tx: number }

export function SystemPanel({ onActionChange }: SystemPanelShape) {
  const [selectedInterface, setSelectedInterface] = useState<string | null>(null)
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
      setStats((current) => {
        setPrev(current)
        return data
      })
      setLastUpdate(new Date())
    } catch {}
  }, [])

  useEffect(() => { fetchStats() }, [fetchStats])
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
        <ReloadButton onClick={fetchStats} />
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

  const rates: Record<string, RatesShape> = {}
  if (prev && stats) {
    for (const iface of Object.keys(stats.interfaces ?? {})) {
      const cur = stats.interfaces![iface]
      const prv = prev.interfaces?.[iface]
      if (prv) rates[iface] = { rx: Math.max(0, cur.rx_bytes - prv.rx_bytes) / 2, tx: Math.max(0, cur.tx_bytes - prv.tx_bytes) / 2 }
    }
  }

  const cpuData = useMemo(() => sysHistory.map((p) => ({ ts: p.ts, cpu: p.cpu_pct })), [sysHistory])
  const memData = useMemo(() => sysHistory.map((p) => ({ ts: p.ts, mem: p.mem_total > 0 ? (p.mem_used / p.mem_total) * 100 : 0 })), [sysHistory])

  const v4Reachable = neighborStats.filter((n) => n.family === 'ipv4' && n.state === 'REACHABLE').length
  const v4Total     = neighborStats.filter((n) => n.family === 'ipv4').length
  const v6Reachable = neighborStats.filter((n) => n.family === 'ipv6' && n.state === 'REACHABLE').length
  const v6Total     = neighborStats.filter((n) => n.family === 'ipv6').length

  const sys = stats?.system

  return (
    <div className="space-y-5">
      {selectedInterface && <InterfaceDetails name={selectedInterface} onClose={() => setSelectedInterface(null)} />}
      {!sys ? <Spinner /> : (
        <>
          <div className="grid gap-3 sm:grid-cols-2">
            <Card className="relative overflow-hidden">
              <div className="relative z-10 flex items-center justify-between px-5 pt-5 pb-1">
                <CardTitle className="text-sm font-medium">CPU</CardTitle>
                <Cpu className="h-4 w-4 text-muted-foreground" />
              </div>
              <div className="relative z-10 px-5 pb-7">
                <div className={`text-3xl font-bold ${sys.cpu_percent > 90 ? 'text-danger' : sys.cpu_percent > 70 ? 'text-warning' : ''}`}>
                  {sys.cpu_percent.toFixed(1)}%
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">
                  load {sys.load1.toFixed(2)} / {sys.load5.toFixed(2)} / {sys.load15.toFixed(2)} · {sys.processes} proc · up {fmtUptime(sys.uptime_seconds)}
                </p>
              </div>
              <div className="absolute inset-x-0 bottom-0 h-24 opacity-25 pointer-events-none">
                <TimeSeriesChart mini data={cpuData} series={[{ dataKey: 'cpu', label: 'CPU %', color: SERIES.cpu }]} height={96} />
              </div>
            </Card>

            <Card className="relative overflow-hidden">
              <div className="relative z-10 flex items-center justify-between px-5 pt-5 pb-1">
                <CardTitle className="text-sm font-medium">Memory</CardTitle>
                <MemoryStick className="h-4 w-4 text-muted-foreground" />
              </div>
              <div className="relative z-10 px-5 pb-7">
                <div className="text-3xl font-bold">{fmtBytes(sys.mem_used)}</div>
                <p className="text-xs text-muted-foreground mt-0.5">
                  of {fmtBytes(sys.mem_total)}
                  {sys.swap_total > 0 && ` · swap ${fmtBytes(sys.swap_used)}/${fmtBytes(sys.swap_total)}`}
                  {' · '}buf {fmtBytes(sys.mem_buffers)}
                </p>
              </div>
              <div className="absolute inset-x-0 bottom-0 h-24 opacity-25 pointer-events-none">
                <TimeSeriesChart mini data={memData} series={[{ dataKey: 'mem', label: 'Mem %', color: SERIES.mem }]} height={96} />
              </div>
            </Card>
          </div>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            {[
              { label: 'IPv4 Neighbors', total: v4Total, reachable: v4Reachable },
              { label: 'IPv6 Neighbors', total: v6Total, reachable: v6Reachable },
            ].map(({ label, total, reachable }) => (
              <Card key={label}>
                <CardContent className="px-5 py-4">
                  <p className="text-xs text-muted-foreground font-medium">{label}</p>
                  <p className="mt-1 text-3xl font-bold">{total}</p>
                  {total > 0 && <p className="text-xs text-muted-foreground mt-0.5">{reachable} reachable</p>}
                  {total === 0 && <p className="text-xs text-muted-foreground mt-0.5">No data</p>}
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
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b border-border text-xs text-muted-foreground">
                      <TableHead className="text-left py-2 pr-4">Interface</TableHead>
                      <TableHead className="text-left py-2 pr-4">State</TableHead>
                      <TableHead className="text-right py-2 pr-4">RX Rate</TableHead>
                      <TableHead className="text-right py-2 pr-4">TX Rate</TableHead>
                      <TableHead className="text-right py-2 pr-4">RX Total</TableHead>
                      <TableHead className="text-right py-2 pr-4">TX Total</TableHead>
                      <TableHead className="text-right py-2">Errors</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {Object.entries(stats.interfaces ?? {}).sort(([a], [b]) => a.localeCompare(b)).map(([name, iface]) => (
                      <TableRow key={name} role="button" tabIndex={0} className="cursor-pointer" onClick={() => setSelectedInterface(name)} onKeyDown={onActivate(() => setSelectedInterface(name))}>
                        <TableCell className="py-2 pr-4 font-mono font-medium">{name}</TableCell>
                        <TableCell className="py-2 pr-4">
                          {vrrpByIface[name] ? (
                            <StateChip state={vrrpByIface[name]} />
                          ) : (
                            <StateChip state={iface.operstate} />
                          )}
                        </TableCell>
                        <TableCell className="py-2 pr-4 text-right font-mono text-xs text-primary">{rates[name] ? fmtBytes(rates[name].rx) + '/s' : '-'}</TableCell>
                        <TableCell className="py-2 pr-4 text-right font-mono text-xs text-success">{rates[name] ? fmtBytes(rates[name].tx) + '/s' : '-'}</TableCell>
                        <TableCell className="py-2 pr-4 text-right font-mono text-xs text-muted-foreground">{fmtBytes(iface.rx_bytes)}</TableCell>
                        <TableCell className="py-2 pr-4 text-right font-mono text-xs text-muted-foreground">{fmtBytes(iface.tx_bytes)}</TableCell>
                        <TableCell className="py-2 text-right font-mono text-xs">
                          {iface.rx_errors + iface.tx_errors > 0
                            ? <span className="text-danger">{iface.rx_errors + iface.tx_errors}</span>
                            : <span className="text-muted-foreground">0</span>}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}

