import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Activity } from 'lucide-react'
import { AccordionList, Badge, Button, Card, CardContent, CardHeader, CardTitle } from 'cheval-ui'
import { TimeSeriesChart } from 'cheval-ui/charts'
import { api, configLayerRequest } from '@/lib/client'

interface PingProbe { name: string; target: string; interval?: number; timeout?: number }
interface Monitoring { collection?: Record<string, number>; probes?: PingProbe[] }
interface ProbePoint { ts: number; name: string; target: string; reachable: boolean; rtt_avg_ms?: number }

export function ProbePanel() {
  const [monitoring, setMonitoring] = useState<Monitoring>({ probes: [] })
  const [history, setHistory] = useState<Record<string, ProbePoint[]>>({})
  const [latest, setLatest] = useState<Record<string, ProbePoint>>({})

  const loadHistory = () => Promise.all([
    api.apiStatsHistoryGet({ minutes: 1440, series: 'probes' }),
    api.apiStatsHistoryGet({ minutes: 60, series: 'probes' }),
  ]).then(([day, hour]) => {
    setHistory((day as unknown as { probes?: Record<string, ProbePoint[]> }).probes ?? {})
    const recent = (hour as unknown as { probes?: Record<string, ProbePoint[]> }).probes ?? {}
    setLatest(Object.fromEntries(Object.entries(recent).flatMap(([name, points]) => points.length ? [[name, points[points.length - 1]]] : [])))
  }).catch(() => { setHistory({}); setLatest({}) })

  useEffect(() => {
    configLayerRequest<Monitoring | null>('/api/config/monitoring').then((value) => setMonitoring(value ?? { probes: [] })).catch(() => {})
    loadHistory()
    const timer = setInterval(loadHistory, 60_000)
    return () => clearInterval(timer)
  }, [])

  const probes = monitoring.probes ?? []

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-3 space-y-0">
        <div className="flex items-center gap-2"><Activity className="h-4 w-4 text-muted-foreground" /><CardTitle className="text-base">Connectivity probes</CardTitle></div>
        <Button variant="outline" size="sm" asChild><Link to="/monitor?tab=probes">Configure</Link></Button>
      </CardHeader>
      <CardContent>
        <AccordionList
          items={probes}
          getId={(probe, index) => `${probe.name}-${index}`}
          description="Scheduled ICMP checks with average round-trip time"
          emptyTitle="No connectivity probes"
          emptyMessage="Configure a target under Monitor > Probes to track reachability and latency."
          renderSummary={(probe) => {
            const current = latest[probe.name]
            return <div className="flex min-w-0 items-center gap-2"><span className="truncate text-sm font-medium">{probe.name}</span><span className="truncate font-mono text-xs text-muted-foreground">{probe.target}</span>{current && <Badge variant={current.reachable ? 'secondary' : 'destructive'}>{current.reachable && current.rtt_avg_ms != null ? `${current.rtt_avg_ms.toFixed(1)} ms` : 'Down'}</Badge>}</div>
          }}
          renderBody={(probe) => {
            const points = history[probe.name] ?? []
            if (points.length === 0) return <p className="text-sm text-muted-foreground">No samples collected yet.</p>
            return <TimeSeriesChart data={points.map((point) => ({ ts: point.ts, rtt: point.rtt_avg_ms ?? null }))} series={[{ dataKey: 'rtt', label: 'Average RTT (ms)', color: '#3584e4' }]} yFormatter={(value) => `${Number(value).toFixed(1)} ms`} height={160} />
          }}
        />
      </CardContent>
    </Card>
  )
}
