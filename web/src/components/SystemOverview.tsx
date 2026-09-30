import { useState, useEffect, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/client'
import type { TypesStatsResponse as StatsResponse, TypesStatsHistoryResponse as StatsHistoryResponse } from '@/api'
import { useFetch } from '@/lib/useFetch'
import { useDataVersion } from '@/lib/dataVersion'
import { Card, CardContent, CardHeader, CardTitle } from 'cheval-ui'
import { TimeSeriesChart } from 'cheval-ui/charts'
import { fmtBitrate, fmtBytes, fmtPps, fmtUptime } from 'cheval-ui'
import { cn } from 'cheval-ui'
import { Cpu, MemoryStick, BarChart2, ArrowRight } from 'lucide-react'

const TRAFFIC_PERIODS = [
  { label: '15m', minutes: 15 },
  { label: '30m', minutes: 30 },
  { label: '1h',  minutes: 60 },
  { label: '6h',  minutes: 360 },
  { label: '24h', minutes: 1440 },
  { label: '7d',  minutes: 10080 },
]

export function SystemOverview({ trafficHref }: { trafficHref?: string }) {
  const { data: stats } = useFetch<StatsResponse>(() => api.apiStatsGet())
  const [history, setHistory] = useState<StatsHistoryResponse | null>(null)
  const [trafficPeriodIdx, setTrafficPeriodIdx] = useState(0)
  const [trafficHistory, setTrafficHistory] = useState<StatsHistoryResponse | null>(null)
  const { version } = useDataVersion()

  useEffect(() => {
    const load = () => api.apiStatsHistoryGet({ minutes: 60, series: 'system' }).then(setHistory).catch(() => {})
    load()
    const id = setInterval(load, 60_000)
    return () => clearInterval(id)
  }, [version])

  useEffect(() => {
    const minutes = TRAFFIC_PERIODS[trafficPeriodIdx].minutes
    const load = () => api.apiStatsHistoryGet({ minutes, series: 'total,usage' }).then(setTrafficHistory).catch(() => {})
    load()
    const id = setInterval(load, 60_000)
    return () => clearInterval(id)
  }, [trafficPeriodIdx, version])

  const trafficChartData = useMemo(() => {
    return (trafficHistory?.total ?? []).map((p) => ({
      ts:     p.ts,
      rx_bytes_ps: p.rx_bytes_ps,
      tx_bytes_ps: p.tx_bytes_ps,
      rx_pps: p.rx_pps,
      tx_pps: p.tx_pps,
    }))
  }, [trafficHistory])

  const sys = stats?.system
  const sysHistory = history?.system ?? []
  const trafficUsage = trafficHistory?.usage

  return (
    <div className="space-y-5">
      {sys && (
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
                load {sys.load1.toFixed(2)} · {sys.processes} proc · up {fmtUptime(sys.uptime_seconds)}
              </p>
            </div>
            <div className="absolute inset-x-0 bottom-0 h-20 opacity-25 pointer-events-none">
              <TimeSeriesChart
                mini
                data={sysHistory.map((p) => ({ ts: p.ts, cpu: p.cpu_pct }))}
                series={[{ dataKey: 'cpu', label: 'CPU %', color: '#3584e4' }]}
                height={80}
              />
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
              </p>
            </div>
            <div className="absolute inset-x-0 bottom-0 h-20 opacity-25 pointer-events-none">
              <TimeSeriesChart
                mini
                data={sysHistory.map((p) => ({
                  ts: p.ts,
                  mem: p.mem_total > 0 ? (p.mem_used / p.mem_total) * 100 : 0,
                }))}
                series={[{ dataKey: 'mem', label: 'Mem %', color: '#9141ac' }]}
                height={80}
              />
            </div>
          </Card>
        </div>
      )}

      <div className="space-y-3">
        <div className="flex items-center justify-between gap-2 flex-wrap">
          <div className="flex items-center gap-2">
            <BarChart2 className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm font-medium">Traffic</span>
          </div>
          <div className="flex items-center gap-2 ml-auto">
            <div className="flex rounded-md border border-input overflow-hidden text-xs">
              {TRAFFIC_PERIODS.map((p, i) => (
                <button
                  key={p.label}
                  onClick={() => setTrafficPeriodIdx(i)}
                  className={cn(
                    'px-2.5 py-1 transition-colors',
                    trafficPeriodIdx === i
                      ? 'bg-primary text-primary-foreground font-medium'
                      : 'bg-background text-muted-foreground hover:bg-muted',
                  )}
                >
                  {p.label}
                </button>
              ))}
            </div>
            {trafficHref && (
              <Link
                to={trafficHref}
                className="flex items-center gap-1 text-xs text-primary hover:underline whitespace-nowrap"
              >
                View all <ArrowRight className="h-3 w-3" />
              </Link>
            )}
          </div>
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <Card>
            <CardHeader className="pb-2 pt-4 px-4">
              <CardTitle className="text-xs font-medium text-muted-foreground">Bandwidth</CardTitle>
              {trafficUsage && (
                <div className="flex gap-4 text-xs text-muted-foreground mt-0.5">
                  <span>
                    <span className="font-medium text-foreground">{fmtBytes(trafficUsage.total.rx_bytes)}</span>
                    {' '}in
                  </span>
                  <span>
                    <span className="font-medium text-foreground">{fmtBytes(trafficUsage.total.tx_bytes)}</span>
                    {' '}out
                  </span>
                </div>
              )}
            </CardHeader>
            <CardContent className="px-4 pb-4">
              <TimeSeriesChart
                data={trafficChartData}
                series={[
                  { dataKey: 'rx_bytes_ps', label: 'RX', color: '#3584e4' },
                  { dataKey: 'tx_bytes_ps', label: 'TX', color: '#26a269' },
                ]}
                yFormatter={fmtBitrate}
                height={200}
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2 pt-4 px-4">
              <CardTitle className="text-xs font-medium text-muted-foreground">Packets / second</CardTitle>
            </CardHeader>
            <CardContent className="px-4 pb-4">
              <TimeSeriesChart
                data={trafficChartData}
                series={[
                  { dataKey: 'rx_pps', label: 'RX pkt/s', color: '#3584e4' },
                  { dataKey: 'tx_pps', label: 'TX pkt/s', color: '#26a269' },
                ]}
                yFormatter={fmtPps}
                height={200}
              />
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
