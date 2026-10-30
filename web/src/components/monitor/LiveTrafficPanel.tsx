import { DirectionCard, type DirectionSample } from '@/components/monitor/LiveTrafficParts/DirectionCard'
import { LIVE_WINDOW } from '@/components/monitor/LiveTrafficParts/ThroughputGraph'
import { C } from '@/components/monitor/shared'
import { api } from '@/lib/client'
import { readEventStream } from '@/lib/eventStream'
import { EmptyState, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Spinner, StateChip } from 'cheval-ui'
import { EthernetPort } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'

interface LiveSample {
  ts: number
  rx_bytes_ps: number
  tx_bytes_ps: number
  rx_pps: number
  tx_pps: number
  rx_bytes: number
  tx_bytes: number
  operstate: string
}

type IfaceEntry = { name: string; alias?: string; operstate: string }

function ifaceLabel(entry: IfaceEntry): string {
  return entry.alias ? `${entry.alias} (${entry.name})` : entry.name
}

function preferredIface(ifaces: IfaceEntry[]): string {
  return (ifaces.find((i) => i.name !== 'lo' && i.operstate === 'up') ?? ifaces[0])?.name ?? ''
}

function parseSample(data: string): LiveSample | null {
  try {
    return JSON.parse(data) as LiveSample
  } catch {
    return null
  }
}

function rxSample(s: LiveSample): DirectionSample {
  return { bps: s.rx_bytes_ps, pps: s.rx_pps, total: s.rx_bytes }
}

function txSample(s: LiveSample): DirectionSample {
  return { bps: s.tx_bytes_ps, pps: s.tx_pps, total: s.tx_bytes }
}

export function LiveTrafficPanel() {
  const [ifaces, setIfaces] = useState<IfaceEntry[] | null>(null)
  const [iface, setIface] = useState('')
  const [samples, setSamples] = useState<LiveSample[]>([])
  const [status, setStatus] = useState('Connecting…')

  useEffect(() => {
    api.apiStatsGet()
      .then((data) => {
        const list = Object.entries(data.interfaces ?? {})
          .map(([name, s]) => ({ name, alias: s.alias, operstate: s.operstate }))
          .sort((a, b) => ifaceLabel(a).localeCompare(ifaceLabel(b)))
        setIfaces(list)
        setIface((prev) => prev || preferredIface(list))
      })
      .catch(() => setIfaces([]))
  }, [])

  useEffect(() => {
    if (!iface) return
    setSamples([])
    const controller = new AbortController()
    let retry: ReturnType<typeof setTimeout>

    const addSample = (data: string) => {
      const sample = parseSample(data)
      if (sample) setSamples((prev) => [...prev, sample].slice(-LIVE_WINDOW))
    }

    const run = async () => {
      setStatus('Connecting…')
      try {
        await readEventStream(`/api/stats/live?iface=${encodeURIComponent(iface)}`, controller.signal, addSample, () => setStatus('Live'))
        throw new Error('Stream disconnected')
      } catch (err) {
        if (!controller.signal.aborted) {
          setStatus(`${err instanceof Error ? err.message : 'Connection failed'}, retrying…`)
          retry = setTimeout(run, 3000)
        }
      }
    }

    run()
    return () => { controller.abort(); clearTimeout(retry) }
  }, [iface])

  const rx = useMemo(() => samples.map(rxSample), [samples])
  const tx = useMemo(() => samples.map(txSample), [samples])
  const operstate = samples[samples.length - 1]?.operstate

  if (ifaces === null) return <div className="flex justify-center py-8"><Spinner /></div>
  if (ifaces.length === 0) return <EmptyState icon={<EthernetPort />} title="No Interfaces" />

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center gap-3">
        <Select value={iface} onValueChange={setIface}>
          <SelectTrigger className="h-7 w-56 text-xs" aria-label="Interface">
            <SelectValue placeholder="Select interface" />
          </SelectTrigger>
          <SelectContent>
            {ifaces.map((i) => <SelectItem key={i.name} value={i.name} className="text-xs font-mono">{ifaceLabel(i)}</SelectItem>)}
          </SelectContent>
        </Select>
        {operstate && <StateChip state={operstate} />}
        <span role="status" className="ml-auto text-xs text-muted-foreground">{status}</span>
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <DirectionCard title="Incoming" color={C.rx} samples={rx} />
        <DirectionCard title="Outgoing" color={C.tx} samples={tx} />
      </div>
    </div>
  )
}
