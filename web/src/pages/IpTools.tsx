import { useEffect, useState } from 'react'
import { api } from '@/lib/client'
import { PageHeader } from '@/components/PageHeader'
import { Tabs } from '@/components/ui/tabs'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { CopyButton } from '@/components/CopyButton'
import { Calculator } from 'lucide-react'
import type { IptoolsSubnetInfo, IptoolsReverseDNS, IptoolsRangeCIDRs } from '@/api'

type Tab = 'subnet' | 'reverse' | 'range'

function useDebounced<T>(value: T, delay = 250): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(id)
  }, [value, delay])
  return debounced
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : 'invalid input'
}

function formatBits(raw: string, prefix: number, is4: boolean): [string, string | null] {
  const group = is4 ? 8 : 16
  let out = ''
  let splitAt = -1
  for (let i = 0; i < raw.length; i++) {
    if (i > 0 && i % group === 0) out += '.'
    if (i === prefix) {
      splitAt = out.length
      out += ' '
    }
    out += raw[i]
  }
  if (splitAt < 0) return [out, null]
  return [out.slice(0, splitAt), out.slice(splitAt + 1)]
}

function BinaryRow({ bits, prefix, is4 }: { bits: string; prefix: number; is4: boolean }) {
  const [net, host] = formatBits(bits, prefix, is4)
  return (
    <span className="font-mono text-xs whitespace-pre">
      <span className="text-primary">{net}</span>
      {host !== null && <span className="text-muted-foreground">{' ' + host}</span>}
    </span>
  )
}

interface CalcRow {
  label: string
  value: string
  bits?: string
}

function ResultTable({ rows, prefix, is4 }: { rows: CalcRow[]; prefix: number; is4: boolean }) {
  return (
    <div className="overflow-x-auto rounded-lg border border-border">
      <table className="w-full text-sm">
        <tbody>
          {rows.map((row) => (
            <tr key={row.label} className="border-b border-border last:border-0">
              <td className="py-2 pl-4 pr-3 align-top font-medium text-muted-foreground w-28 whitespace-nowrap">{row.label}</td>
              <td className="py-2 pr-3 align-top font-mono whitespace-nowrap">
                <span className="inline-flex items-center gap-1.5">
                  {row.value}
                  <CopyButton text={row.value} />
                </span>
              </td>
              <td className="py-2 pr-4 align-top">
                {row.bits && <BinaryRow bits={row.bits} prefix={prefix} is4={is4} />}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ToolInput({ label, hint, ...props }: { label: string; hint?: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs">{label}</Label>
      <Input className="font-mono" spellCheck={false} autoComplete="off" {...props} />
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  )
}

function ErrorNote({ message }: { message: string }) {
  return <p className="text-sm text-destructive">{message}</p>
}

function SubnetTool() {
  const [input, setInput] = useState('100.64.12.192/27')
  const query = useDebounced(input.trim())
  const [info, setInfo] = useState<IptoolsSubnetInfo | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!query) {
      setInfo(null)
      setError(null)
      return
    }
    let live = true
    api.apiToolsSubnetGet({ cidr: query })
      .then((res) => { if (live) { setInfo(res as IptoolsSubnetInfo); setError(null) } })
      .catch((err) => { if (live) { setInfo(null); setError(errorMessage(err)) } })
    return () => { live = false }
  }, [query])

  const is4 = info?.family === 'v4'
  const rows: CalcRow[] = info
    ? [
        { label: 'Address', value: info.address, bits: info.address_bits },
        { label: 'Netmask', value: is4 ? `${info.netmask} = ${info.prefix}` : info.netmask, bits: info.netmask_bits },
        { label: 'Wildcard', value: info.wildcard, bits: info.wildcard_bits },
        { label: info.host_route ? 'Hostroute' : 'Network', value: info.network, bits: info.network_bits },
        ...(info.host_route ? [] : [
          { label: 'HostMin', value: info.host_min, bits: info.host_min_bits },
          { label: 'HostMax', value: info.host_max, bits: info.host_max_bits },
        ]),
        ...(info.broadcast ? [{ label: 'Broadcast', value: info.broadcast, bits: info.broadcast_bits }] : []),
      ]
    : []

  return (
    <div className="space-y-4">
      <ToolInput
        label="Address or CIDR"
        hint="e.g. 100.64.12.192/27 or 2001:db8::/64. A bare address is treated as a host route."
        value={input}
        onChange={(e) => setInput(e.target.value)}
        placeholder="100.64.12.192/27"
      />
      {error && <ErrorNote message={error} />}
      {info && (
        <>
          <ResultTable rows={rows} prefix={info.prefix} is4={is4} />
          <div className="flex flex-wrap gap-2">
            <Badge variant="secondary">Hosts/Net: {info.hosts}</Badge>
            {info._class && <Badge variant="outline">Class {info._class}</Badge>}
            <Badge variant="outline">{info.scope}</Badge>
            <Badge variant="outline">{info.family === 'v4' ? 'IPv4' : 'IPv6'}</Badge>
          </div>
        </>
      )}
    </div>
  )
}

function ReverseTool() {
  const [input, setInput] = useState('100.64.12.192')
  const query = useDebounced(input.trim())
  const [result, setResult] = useState<IptoolsReverseDNS | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!query) {
      setResult(null)
      setError(null)
      return
    }
    let live = true
    api.apiToolsReverseGet({ ip: query })
      .then((res) => { if (live) { setResult(res as IptoolsReverseDNS); setError(null) } })
      .catch((err) => { if (live) { setResult(null); setError(errorMessage(err)) } })
    return () => { live = false }
  }, [query])

  return (
    <div className="space-y-4">
      <ToolInput
        label="Address or CIDR"
        hint="A CIDR aligned to an octet (IPv4) or nibble (IPv6) boundary also shows its delegation zone."
        value={input}
        onChange={(e) => setInput(e.target.value)}
        placeholder="100.64.12.192 or 2001:db8::/32"
      />
      {error && <ErrorNote message={error} />}
      {result && (
        <div className="space-y-3 rounded-lg border border-border p-4">
          <div className="space-y-1">
            <Label className="text-xs">PTR name</Label>
            <div className="flex items-center gap-1.5 font-mono text-sm break-all">
              {result.name}
              <CopyButton text={result.name} />
            </div>
          </div>
          {result.zone && (
            <div className="space-y-1">
              <Label className="text-xs">Delegation zone</Label>
              <div className="flex items-center gap-1.5 font-mono text-sm break-all">
                {result.zone}
                <CopyButton text={result.zone} />
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function RangeTool() {
  const [start, setStart] = useState('192.0.2.5')
  const [end, setEnd] = useState('192.0.2.20')
  const qStart = useDebounced(start.trim())
  const qEnd = useDebounced(end.trim())
  const [result, setResult] = useState<IptoolsRangeCIDRs | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!qStart || !qEnd) {
      setResult(null)
      setError(null)
      return
    }
    let live = true
    api.apiToolsRangeGet({ start: qStart, end: qEnd })
      .then((res) => { if (live) { setResult(res as IptoolsRangeCIDRs); setError(null) } })
      .catch((err) => { if (live) { setResult(null); setError(errorMessage(err)) } })
    return () => { live = false }
  }, [qStart, qEnd])

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <ToolInput label="Start" value={start} onChange={(e) => setStart(e.target.value)} placeholder="192.0.2.5" />
        <ToolInput label="End" value={end} onChange={(e) => setEnd(e.target.value)} placeholder="192.0.2.20" />
      </div>
      {error && <ErrorNote message={error} />}
      {result && (
        <div className="space-y-2 rounded-lg border border-border p-4">
          <div className="flex items-center justify-between">
            <Label className="text-xs">{result.cidrs.length} block{result.cidrs.length !== 1 ? 's' : ''}</Label>
            <CopyButton text={result.cidrs.join('\n')} />
          </div>
          <div className="flex flex-wrap gap-1.5">
            {result.cidrs.map((c) => (
              <Badge key={c} variant="outline" className="font-mono">{c}</Badge>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

const TABS: { key: Tab; label: string }[] = [
  { key: 'subnet', label: 'Subnet calculator' },
  { key: 'reverse', label: 'Reverse DNS' },
  { key: 'range', label: 'Range to CIDR' },
]

export default function IpTools() {
  const [tab, setTab] = useState<Tab>('subnet')

  return (
    <div className="space-y-6">
      <PageHeader title="IP Tools" description="Subnet breakdown, reverse-DNS names, and range-to-CIDR conversion for IPv4 and IPv6." />
      <div className="flex items-center gap-2 text-muted-foreground">
        <Calculator className="h-4 w-4" />
        <Tabs tabs={TABS} active={tab} onChange={setTab} />
      </div>
      <div className="max-w-3xl">
        {tab === 'subnet' && <SubnetTool />}
        {tab === 'reverse' && <ReverseTool />}
        {tab === 'range' && <RangeTool />}
      </div>
    </div>
  )
}
