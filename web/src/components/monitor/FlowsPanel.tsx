import type { TypesFlow as Flow, TypesFlowSankeyResponse as FlowSankeyResponse, TypesFlowSeriesResponse as FlowSeriesResponse, TypesFlowStatus as FlowStatus, TypesFlowTopEntry as FlowTopEntry } from '@/api'
import { FlowCollectors } from '@/components/monitor/FlowsPanelParts/FlowCollectors'
import { FlowRecentTable } from '@/components/monitor/FlowsPanelParts/FlowRecentTable'
import { FlowSankey, sankeyDimensions, SankeyViewSelect, type SankeyView } from '@/components/monitor/FlowsPanelParts/FlowSankey'
import { FlowDimensionSelect, FlowTopTable, type FlowDimension } from '@/components/monitor/FlowsPanelParts/FlowTopTable'
import { Legend, PERIODS } from '@/components/monitor/shared'
import { api } from '@/lib/client'
import { useIfaceLabels } from '@/lib/ifaceNames'
import { SERIES_CYCLE } from '@/lib/palette'
import { Button, Card, CardContent, CardHeader, CardTitle, EmptyState, fmtBitrate, ReloadButton, Segmented, Spinner, StatCard } from 'cheval-ui'
import { TimeSeriesChart } from 'cheval-ui/charts'
import { ArrowDownUp, Database, Gauge, Send } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'

const FLOW_PERIODS = PERIODS.filter((p) => p.minutes <= 10080)
const REFRESH_MS = 30_000

type FlowData = { status: FlowStatus; series: FlowSeriesResponse; sankey: FlowSankeyResponse; top: FlowTopEntry[]; flows: Flow[] }
type SeriesRow = Record<string, number | null>

type PivotedSeries = { rows: SeriesRow[]; ifaces: string[] }

function pivotSeries(series: FlowSeriesResponse): PivotedSeries {
  const ifaces = [...new Set(series.points.map((p) => p.iface))].sort()
  const byTs = new Map<number, SeriesRow>()
  for (const p of series.points) {
    const ts = Math.floor(p.ts / series.bucket) * series.bucket
    const row = byTs.get(ts) ?? { ts }
    row[p.iface || 'local'] = ((row[p.iface || 'local'] ?? 0) as number) + p.bytes / series.bucket
    byTs.set(ts, row)
  }

  return { rows: [...byTs.values()].sort((a, b) => (a.ts as number) - (b.ts as number)), ifaces: ifaces.map((i) => i || 'local') }
}

function datagramsSent(status: FlowStatus): string {
  return status.collectors.reduce((sum, c) => sum + c.sent, 0).toLocaleString()
}

export function FlowsPanel() {
  const [periodIdx, setPeriodIdx] = useState(2)
  const [dimension, setDimension] = useState<FlowDimension>('src')
  const [view, setView] = useState<SankeyView>('hosts')
  const [address, setAddress] = useState('')
  const [debouncedAddress, setDebouncedAddress] = useState('')
  const [data, setData] = useState<FlowData | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const ifaceLabel = useIfaceLabels()
  const minutes = FLOW_PERIODS[periodIdx].minutes

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const [status, series, sankey, top, recent] = await Promise.all([
        api.apiFlowsStatusGet(),
        api.apiFlowsSeriesGet({ minutes }),
        api.apiFlowsSankeyGet({ dimensions: sankeyDimensions(view).join(','), minutes, limit: 15 }),
        api.apiFlowsTopGet({ dimension, minutes, limit: 15 }),
        api.apiFlowsGet({ minutes, address: debouncedAddress || undefined, limit: 500 }),
      ])
      setData({ status, series, sankey, top: top.entries, flows: recent.flows })
      setError(null)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [minutes, view, dimension, debouncedAddress])

  useEffect(() => {
    const id = setTimeout(() => setDebouncedAddress(address), 300)
    return () => clearTimeout(id)
  }, [address])

  useEffect(() => {
    void refresh()
    const id = setInterval(() => { void refresh() }, REFRESH_MS)
    return () => clearInterval(id)
  }, [refresh])

  const chart = useMemo(() => data ? pivotSeries(data.series) : { rows: [], ifaces: [] }, [data])
  const series = chart.ifaces.map((iface, i) => ({ dataKey: iface, label: iface === 'local' ? iface : ifaceLabel(iface), color: SERIES_CYCLE[i % SERIES_CYCLE.length] }))

  if (!data && loading) return <Spinner />

  if (!data) {
    return (
      <div role="alert" className="space-y-3">
        <p className="text-sm text-danger">{error}</p>
        <Button variant="outline" onClick={() => { void refresh() }}>Retry</Button>
      </div>
    )
  }

  if (!data.status.enabled && data.status.stored === 0) {
    return (
      <EmptyState
        icon={<ArrowDownUp />}
        title="Flow Sampling Off"
        message="Sampled flows are exported over sFlow and kept here"
        action={<Button variant="outline" asChild><Link to="/system?system-config=monitoring">Configure…</Link></Button>}
      />
    )
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <Segmented value={String(periodIdx)} onChange={(v) => setPeriodIdx(Number(v))} options={FLOW_PERIODS.map((p, i) => ({ value: String(i), label: p.label }))} />
        <ReloadButton onClick={() => { void refresh() }} loading={loading} />
      </div>

      {error && <p role="alert" className="text-sm text-danger">{error}</p>}
      {data.status.error && <p role="alert" className="text-sm text-danger break-all">{data.status.error}</p>}

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <StatCard icon={Gauge} label="Sampling" value={data.status.sampling ? `1:${data.status.sampling}` : '-'} caption={data.status.running ? 'Agent running' : 'Agent stopped'} warn={data.status.enabled && !data.status.running} />
        <StatCard icon={Database} label="Stored Flows" value={data.status.stored.toLocaleString()} caption={`${data.status.samples.toLocaleString()} samples since start`} />
        <StatCard icon={Send} label="sFlow Datagrams" value={datagramsSent(data.status)} caption={data.status.collectors.length === 1 ? '1 collector' : `${data.status.collectors.length} collectors`} />
      </div>

      <FlowCollectors status={data.status} />

      <Card>
        <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3 space-y-0 pb-3">
          <CardTitle className="text-sm">Traffic Flow</CardTitle>
          <SankeyViewSelect value={view} onChange={setView} />
        </CardHeader>
        <CardContent>
          <FlowSankey paths={data.sankey.paths} dimensions={data.sankey.dimensions} total={data.sankey.total} names={data.sankey.names} ifaceLabel={ifaceLabel} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between gap-3 flex-wrap">
            <CardTitle className="text-sm">Traffic by Input Interface</CardTitle>
            <Legend items={series.map((s) => ({ color: s.color, label: s.label }))} />
          </div>
        </CardHeader>
        <CardContent>
          <TimeSeriesChart data={chart.rows} series={series} yFormatter={fmtBitrate} height={220} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3 space-y-0 pb-3">
          <CardTitle className="text-sm">Top Traffic</CardTitle>
          <FlowDimensionSelect value={dimension} onChange={setDimension} />
        </CardHeader>
        <CardContent>
          <FlowTopTable dimension={dimension} entries={data.top} ifaceLabel={ifaceLabel} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3"><CardTitle className="text-sm">Recent Flows</CardTitle></CardHeader>
        <CardContent>
          <FlowRecentTable flows={data.flows} address={address} onAddressChange={setAddress} ifaceLabel={ifaceLabel} />
        </CardContent>
      </Card>
    </div>
  )
}
