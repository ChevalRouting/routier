import { useState, useEffect, useMemo } from 'react'
import { api } from '@/lib/client'
import type { TypesStatsHistoryResponse as StatsHistoryResponse } from '@/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Spinner } from '@/components/Spinner'
import { TimeSeriesChart } from '@/components/TimeSeriesChart'
import { fmtBitrate, fmtPps } from '@/lib/fmt'
import { cn } from '@/lib/utils'
import { useIfaceLabels } from '@/lib/ifaceNames'
import { PERIODS, C, Legend } from './shared'

export function TrafficPanel() {
  const [history, setHistory] = useState<StatsHistoryResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [periodIdx, setPeriodIdx] = useState(0)
  const [selectedIface, setSelectedIface] = useState<string>('__all__')
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
      if (selectedIface === '__all__') api.apiStatsHistoryGet({ minutes: period.minutes }).then(setHistory).catch(() => {})
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
    return rows.map((p) => ({ ts: p.ts, rx_bps: p.rx_bps, tx_bps: p.tx_bps, rx_pps: p.rx_pps, tx_pps: p.tx_pps }))
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
        <div className="flex rounded-md border border-input overflow-hidden text-xs ml-auto">
          {PERIODS.map((p, i) => (
            <button key={p.label} onClick={() => setPeriodIdx(i)} className={cn('px-3 py-1.5 transition-colors', periodIdx === i ? 'bg-primary text-primary-foreground font-medium' : 'bg-background text-muted-foreground hover:bg-muted')}>
              {p.label}
            </button>
          ))}
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
                  <span className="text-xs font-medium text-muted-foreground">Bandwidth</span>
                  <Legend items={[{ color: C.rx, label: 'RX' }, { color: C.tx, label: 'TX' }]} />
                </div>
                <TimeSeriesChart data={ifaceData} series={[{ dataKey: 'rx_bps', label: 'RX', color: C.rx }, { dataKey: 'tx_bps', label: 'TX', color: C.tx }]} yFormatter={fmtBitrate} height={200} />
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

