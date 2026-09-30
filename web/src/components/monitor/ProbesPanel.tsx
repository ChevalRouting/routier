import { useEffect, useState } from 'react'
import { Activity, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { AccordionList, Badge, Button, Card, CardContent, CardHeader, CardTitle, Input, Label, SaveButton } from 'cheval-ui'
import { TimeSeriesChart } from 'cheval-ui/charts'
import { api, configLayerRequest } from '@/lib/client'
import { OnActionChange } from './shared'

interface PingProbe { name: string; target: string; interval?: number; timeout?: number }
interface Monitoring { collection?: Record<string, number>; probes?: PingProbe[] }
interface ProbePoint { ts: number; name: string; target: string; reachable: boolean; rtt_avg_ms?: number }

export function ProbesPanel({ onActionChange }: { onActionChange: OnActionChange }) {
  const [monitoring, setMonitoring] = useState<Monitoring>({ probes: [] })
  const [history, setHistory] = useState<Record<string, ProbePoint[]>>({})
  const [latest, setLatest] = useState<Record<string, ProbePoint>>({})
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    onActionChange(null)
  }, [onActionChange])

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
  const update = (probe: PingProbe, patch: Partial<PingProbe>) => {
    setMonitoring({ ...monitoring, probes: probes.map((item) => item === probe ? { ...item, ...patch } : item) })
    setDirty(true)
  }
  const remove = (probe: PingProbe) => { setMonitoring({ ...monitoring, probes: probes.filter((item) => item !== probe) }); setDirty(true) }
  const save = async () => {
    setSaving(true)
    try {
      await configLayerRequest('/api/config/monitoring', { method: 'PUT', body: JSON.stringify({ ...monitoring, probes }) })
      setDirty(false)
      toast.success('Ping probes staged; apply the configuration to start collecting')
    } catch (error) { toast.error((error as Error).message) } finally { setSaving(false) }
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-3 space-y-0">
        <div className="flex items-center gap-2"><Activity className="h-4 w-4 text-muted-foreground" /><CardTitle className="text-base">Connectivity probes</CardTitle></div>
        <SaveButton isDirty={dirty} saving={saving} onClick={save} />
      </CardHeader>
      <CardContent>
        <AccordionList
          items={probes}
          getId={(_probe, index) => String(index)}
          description="Scheduled ICMP checks with average round-trip time"
          addLabel="Add probe"
          onAdd={() => { setMonitoring({ ...monitoring, probes: [...probes, { name: `probe-${probes.length + 1}`, target: '1.1.1.1', interval: 300, timeout: 5000 }] }); setDirty(true) }}
          emptyTitle="No connectivity probes"
          emptyMessage="Add a target such as 1.1.1.1 to track reachability and latency."
          renderSummary={(probe) => {
            const current = latest[probe.name]
            return <div className="flex min-w-0 items-center gap-2"><span className="truncate text-sm font-medium">{probe.name}</span><span className="truncate font-mono text-xs text-muted-foreground">{probe.target}</span>{current && <Badge variant={current.reachable ? 'secondary' : 'destructive'}>{current.reachable && current.rtt_avg_ms != null ? `${current.rtt_avg_ms.toFixed(1)} ms` : 'Down'}</Badge>}</div>
          }}
          renderActions={(probe) => <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => remove(probe)}><Trash2 className="h-3.5 w-3.5" /></Button>}
          renderBody={(probe) => {
            const points = history[probe.name] ?? []
            return <div className="space-y-4"><div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4"><Field label="Name"><Input value={probe.name} onChange={(event) => update(probe, { name: event.target.value })} /></Field><Field label="Target"><Input className="font-mono" value={probe.target} placeholder="1.1.1.1" onChange={(event) => update(probe, { target: event.target.value })} /></Field><Field label="Interval (seconds)"><Input type="number" min={60} step={60} value={probe.interval ?? 300} onChange={(event) => update(probe, { interval: Math.max(60, Number(event.target.value)) })} /></Field><Field label="Timeout (milliseconds)"><Input type="number" min={100} max={55000} step={100} value={probe.timeout ?? 5000} onChange={(event) => update(probe, { timeout: Math.min(55000, Math.max(100, Number(event.target.value))) })} /></Field></div>{points.length > 0 && <TimeSeriesChart data={points.map((point) => ({ ts: point.ts, rtt: point.rtt_avg_ms ?? null }))} series={[{ dataKey: 'rtt', label: 'Average RTT (ms)', color: '#3584e4' }]} yFormatter={(value) => `${Number(value).toFixed(1)} ms`} height={160} />}</div>
          }}
        />
      </CardContent>
    </Card>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="space-y-1.5"><Label>{label}</Label>{children}</div>
}
