import type { TypesStatsHistoryResponse as StatsHistoryResponse } from '@/api'
import { api } from '@/lib/client'
import { useIfaceLabels } from '@/lib/ifaceNames'
import { Card, CardContent, CardHeader, CardTitle, fmtBitrate, fmtBytes, fmtPps, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Spinner, Segmented } from 'cheval-ui'
import { TimeSeriesChart } from 'cheval-ui/charts'
import { useEffect, useMemo, useRef, useState } from 'react'
import { C, Legend, PERIODS } from './shared'

export function TrafficPanel() {
  const [history, setHistory] = useState<StatsHistoryResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [periodIdx, setPeriodIdx] = useState(0)
  const [selectedIface, setSelectedIface] = useState<string>('__all__')
  const selectedIfaceRef = useRef(selectedIface)
  useEffect(() => { selectedIfaceRef.current = selectedIface }, [selectedIface])
  const [ifaceNames, setIfaceNames] = useState<string[]>([])
  const ifaceLabel = useIfaceLabels()
  const period = PERIODS[periodIdx]

  useEffect(() => {
    setLoading(true)
    api.apiStatsHistoryGet({ minutes: period.minutes })
      .then((data) => {
        setHistory(data)
        const keys = Object.keys(data.interfaces ?? {}).sort()
        setIfaceNames(keys)
        setSelectedIface((prev) => (prev === '__all__' || keys.includes(prev) ? prev : '__all__'))
      })
      .catch(() => {}).finally(() => setLoading(false))
    const id = setInterval(() => {
      if (selectedIfaceRef.current === '__all__') api.apiStatsHistoryGet({ minutes: period.minutes }).then(setHistory).catch(() => {})
    }, 60_000)
    return () => clearInterval(id)
  }, [period.minutes])

  useEffect(() => {
    if (selectedIface === '__all__') return
    setLoading(true)
    api.apiStatsHistoryGet({ minutes: period.minutes, iface: selectedIface }).then(setHistory).catch(() => {}).finally(() => setLoading(false))
    const id = setInterval(() => { api.apiStatsHistoryGet({ minutes: period.minutes, iface: selectedIface }).then(setHistory).catch(() => {}) }, 60_000)
    return () => clearInterval(id)
  }, [selectedIface, period.minutes])

  const ifaceData = useMemo(() => {
    if (!history) return []
    const rows = selectedIface === '__all__' ? (history.total ?? []) : (history.interfaces?.[selectedIface] ?? [])
    return rows.map((p) => ({ ts: p.ts, rx_bytes_ps: p.rx_bytes_ps, tx_bytes_ps: p.tx_bytes_ps, rx_pps: p.rx_pps, tx_pps: p.tx_pps }))
  }, [history, selectedIface])

  const usageData = useMemo(() => {
    const points = []
    let rx = 0, tx = 0
    for (let i = 0; i < ifaceData.length; i++) {
      const p = ifaceData[i]
      if (i > 0) {
        const dt = p.ts - ifaceData[i - 1].ts
        rx += (p.rx_bytes_ps ?? 0) * dt
        tx += (p.tx_bytes_ps ?? 0) * dt
      }
      points.push({ ts: p.ts, rx_bytes: rx, tx_bytes: tx })
    }
    return points
  }, [ifaceData])

  const usageTotal = useMemo(() => {
    if (!history?.usage) return null
    return selectedIface === '__all__' ? history.usage.total : history.usage.per_iface?.[selectedIface]
  }, [history, selectedIface])

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        {ifaceNames.length > 0 && (
          <Select value={selectedIface} onValueChange={setSelectedIface}>
            <SelectTrigger className="h-7 w-44 text-xs">
              <SelectValue placeholder="Select interface" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__" className="text-xs">All interfaces</SelectItem>
              {ifaceNames.map((name) => <SelectItem key={name} value={name} className="text-xs font-mono">{ifaceLabel(name)}</SelectItem>)}
            </SelectContent>
          </Select>
        )}
        <div className="ml-auto">
          <Segmented value={String(periodIdx)} onChange={(v) => setPeriodIdx(Number(v))} options={PERIODS.map((p, i) => ({ value: String(i), label: p.label }))} />
        </div>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm">Interface Traffic</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          {loading ? <div className="flex justify-center py-8"><Spinner /></div> : (
            <>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium text-muted-foreground">Data consumption</span>
                  <div className="flex items-center gap-4 text-xs">
                    <span className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full" style={{ background: C.rx }} /><span className="text-muted-foreground">RX</span><span className="font-medium tabular-nums text-foreground">{fmtBytes(usageTotal?.rx_bytes ?? 0)}</span></span>
                    <span className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full" style={{ background: C.tx }} /><span className="text-muted-foreground">TX</span><span className="font-medium tabular-nums text-foreground">{fmtBytes(usageTotal?.tx_bytes ?? 0)}</span></span>
                  </div>
                </div>
                <TimeSeriesChart data={usageData} series={[{ dataKey: 'rx_bytes', label: 'RX', color: C.rx }, { dataKey: 'tx_bytes', label: 'TX', color: C.tx }]} yFormatter={fmtBytes} height={200} />
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium text-muted-foreground">Bandwidth</span>
                  <Legend items={[{ color: C.rx, label: 'RX' }, { color: C.tx, label: 'TX' }]} />
                </div>
                <TimeSeriesChart data={ifaceData} series={[{ dataKey: 'rx_bytes_ps', label: 'RX', color: C.rx }, { dataKey: 'tx_bytes_ps', label: 'TX', color: C.tx }]} yFormatter={fmtBitrate} height={200} />
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium text-muted-foreground">Packets / second</span>
                  <Legend items={[{ color: C.rx, label: 'RX' }, { color: C.tx, label: 'TX' }]} />
                </div>
                <TimeSeriesChart data={ifaceData} series={[{ dataKey: 'rx_pps', label: 'RX pkt/s', color: C.rx }, { dataKey: 'tx_pps', label: 'TX pkt/s', color: C.tx }]} yFormatter={fmtPps} height={200} />
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export type LogSource = 'messages' | 'dmesg'
export type LogSubTab = LogSource | 'settings'

