import { SERIES } from '@/lib/palette'
import { DraftControls } from '@/components/DraftControls'
import { api, apiRequest } from '@/lib/client'
import { AccordionList, Button, Card, CardContent, CardHeader, CardTitle, EntryRow, IntegerEntryRow, PreferencesGroup, StatusChip } from 'cheval-ui'
import { TimeSeriesChart } from 'cheval-ui/charts'
import { Activity, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { OnActionChange } from './shared'

type ProbesPanelShape = { onActionChange: OnActionChange }

type ProbesShape2 = { probes?: Record<string, ProbePoint[]> }

type ProbesShape = { probes?: Record<string, ProbePoint[]> }

type ProbeBodyShape = { probe: PingProbe; points: ProbePoint[]; onUpdate: (patch: Partial<PingProbe>) => void }

interface PingProbe { name: string; target: string; interval?: number; timeout?: number }
interface Monitoring { collection?: Record<string, number>; probes?: PingProbe[] }
interface ProbePoint { ts: number; name: string; target: string; reachable: boolean; rtt_avg_ms?: number }

export function ProbesPanel({ onActionChange }: ProbesPanelShape) {
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
    setHistory((day as unknown as ProbesShape2).probes ?? {})
    const recent = (hour as unknown as ProbesShape).probes ?? {}
    setLatest(Object.fromEntries(Object.entries(recent).flatMap(([name, points]) => points.length ? [[name, points[points.length - 1]]] : [])))
  }).catch(() => { setHistory({}); setLatest({}) })

  useEffect(() => {
    apiRequest<Monitoring | null>('/api/config/monitoring').then((value) => setMonitoring(value ?? { probes: [] })).catch(() => {})
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
      await apiRequest('/api/config/monitoring', { method: 'PUT', body: JSON.stringify({ ...monitoring, probes }) })
      setDirty(false)
      toast.success('Ping probes staged; apply the configuration to start collecting')
    } catch (error) { toast.error((error as Error).message) } finally { setSaving(false) }
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-3 space-y-0">
        <div className="flex items-center gap-2"><Activity className="h-4 w-4 text-muted-foreground" /><CardTitle className="text-base">Connectivity Probes</CardTitle></div>
        <DraftControls isDirty={dirty} saving={saving} onClick={save} />
      </CardHeader>
      <CardContent>
        <AccordionList
          items={probes}
          getId={(_probe, index) => String(index)}
          addLabel="Add Probe"
          onAdd={() => { setMonitoring({ ...monitoring, probes: [...probes, { name: `probe-${probes.length + 1}`, target: '1.1.1.1', interval: 300, timeout: 5000 }] }); setDirty(true) }}
          emptyTitle="No Connectivity Probes"
          renderSummary={(probe) => {
            const current = latest[probe.name]
            return <div className="flex min-w-0 items-center gap-2"><span className="truncate text-sm font-medium">{probe.name}</span><span className="truncate font-mono text-xs text-muted-foreground">{probe.target}</span>{current && <StatusChip intent={current.reachable ? 'success' : 'danger'} label={current.reachable && current.rtt_avg_ms != null ? `${current.rtt_avg_ms.toFixed(1)} ms` : 'Down'} />}</div>
          }}
          renderActions={(probe) => <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-destructive" onClick={() => remove(probe)} aria-label="Remove Probe" title="Remove Probe"><Trash2 className="h-4 w-4" /></Button>}
          renderBody={(probe) => <ProbeBody probe={probe} points={history[probe.name] ?? []} onUpdate={(patch) => update(probe, patch)} />}
        />
      </CardContent>
    </Card>
  )
}

function ProbeBody({ probe, points, onUpdate }: ProbeBodyShape) {
  return (
    <div className="space-y-4">
      <PreferencesGroup>
        <EntryRow title="Name" value={probe.name} onChange={(event) => onUpdate({ name: event.target.value })} />
        <EntryRow title="Target" value={probe.target} onChange={(event) => onUpdate({ target: event.target.value })} placeholder="1.1.1.1" className="font-mono" />
        <IntegerEntryRow title="Interval (s)" min={60} value={probe.interval ?? 300} onValueChange={(interval) => onUpdate({ interval })} />
        <IntegerEntryRow title="Timeout (ms)" min={100} max={55000} value={probe.timeout ?? 5000} onValueChange={(timeout) => onUpdate({ timeout })} />
      </PreferencesGroup>
      {points.length > 0 && (
        <TimeSeriesChart
          data={points.map((point) => ({ ts: point.ts, rtt: point.rtt_avg_ms ?? null }))}
          series={[{ dataKey: 'rtt', label: 'Average RTT (ms)', color: SERIES.rtt }]}
          yFormatter={(value) => `${Number(value).toFixed(1)} ms`}
          height={160}
        />
      )}
    </div>
  )
}
