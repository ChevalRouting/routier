import type { TypesFlowTopEntry as FlowTopEntry } from '@/api'
import { EmptyState, fmtBytes, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'
import { topKeyLabel } from './flowFormat'

export type FlowDimension = 'src' | 'dst' | 'service' | 'conversation' | 'in_if'

type DimensionOption = { value: FlowDimension; label: string; column: string }

export const DIMENSIONS: DimensionOption[] = [
  { value: 'src', label: 'Sources', column: 'Source' },
  { value: 'dst', label: 'Destinations', column: 'Destination' },
  { value: 'service', label: 'Services', column: 'Service' },
  { value: 'conversation', label: 'Conversations', column: 'Conversation' },
  { value: 'in_if', label: 'Interfaces', column: 'Input Interface' },
]

type FlowDimensionSelectShape = {
  value: FlowDimension
  onChange: (dimension: FlowDimension) => void
}

export function FlowDimensionSelect({ value, onChange }: FlowDimensionSelectShape) {
  return (
    <Select value={value} onValueChange={(v) => onChange(v as FlowDimension)}>
      <SelectTrigger aria-label="Group by" className="h-8 w-40 text-xs">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {DIMENSIONS.map((d) => <SelectItem key={d.value} value={d.value} className="text-xs">{d.label}</SelectItem>)}
      </SelectContent>
    </Select>
  )
}

type FlowTopTableShape = {
  dimension: FlowDimension
  entries: FlowTopEntry[]
  ifaceLabel: (device: string) => string
}

export function FlowTopTable({ dimension, entries, ifaceLabel }: FlowTopTableShape) {
  const column = DIMENSIONS.find((d) => d.value === dimension)?.column ?? 'Key'
  const total = entries.reduce((sum, e) => sum + e.bytes, 0)
  const label = (key: string) => dimension === 'in_if' && key ? ifaceLabel(key) : topKeyLabel(dimension, key)

  if (entries.length === 0) return <EmptyState className="py-8" title="No Flows" message="No traffic was sampled in this period" />

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{column}</TableHead>
          <TableHead className="text-right">Bytes</TableHead>
          <TableHead className="text-right">Share</TableHead>
          <TableHead className="text-right">Packets</TableHead>
          <TableHead className="text-right">Flows</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {entries.map((e) => (
          <TableRow key={e.key}>
            <TableCell className="font-mono text-xs break-all">{label(e.key)}</TableCell>
            <TableCell className="text-right tabular-nums">{fmtBytes(e.bytes)}</TableCell>
            <TableCell className="text-right tabular-nums">{total > 0 ? `${Math.round((e.bytes / total) * 100)} %` : '-'}</TableCell>
            <TableCell className="text-right tabular-nums">{e.packets.toLocaleString()}</TableCell>
            <TableCell className="text-right tabular-nums">{e.flows.toLocaleString()}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
