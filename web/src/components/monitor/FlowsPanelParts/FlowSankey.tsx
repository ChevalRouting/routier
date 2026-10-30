import type { TypesFlowPath as FlowPath } from '@/api'
import { SERIES_CYCLE } from '@/lib/palette'
import { EmptyState, fmtBytes, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Layer, Rectangle, Sankey, Tooltip, type SankeyLinkProps, type SankeyNodeProps } from 'recharts'
import { topKeyLabel } from './flowFormat'

export type SankeyView = 'hosts' | 'interfaces' | 'ingress'

type ViewOption = { value: SankeyView; label: string; dimensions: string[] }

export const SANKEY_VIEWS: ViewOption[] = [
  { value: 'hosts', label: 'Source → Service → Destination', dimensions: ['src', 'service', 'dst'] },
  { value: 'interfaces', label: 'Input → Service → Output', dimensions: ['in_if', 'service', 'out_if'] },
  { value: 'ingress', label: 'Input → Source → Destination', dimensions: ['in_if', 'src', 'dst'] },
]

export function sankeyDimensions(view: SankeyView): string[] {
  return SANKEY_VIEWS.find((v) => v.value === view)?.dimensions ?? SANKEY_VIEWS[0].dimensions
}

type SankeyViewSelectShape = {
  value: SankeyView
  onChange: (view: SankeyView) => void
}

export function SankeyViewSelect({ value, onChange }: SankeyViewSelectShape) {
  return (
    <Select value={value} onValueChange={(v) => onChange(v as SankeyView)}>
      <SelectTrigger aria-label="Diagram view" className="h-8 w-60 max-w-full text-xs">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {SANKEY_VIEWS.map((v) => <SelectItem key={v.value} value={v.value} className="text-xs">{v.label}</SelectItem>)}
      </SelectContent>
    </Select>
  )
}

type DiagramNode = { name: string; column: number; color: string; ptr?: string }
type DiagramLink = { source: number; target: number; value: number }
type Diagram = { nodes: DiagramNode[]; links: DiagramLink[]; rows: number }

type KeyLookup = (dimension: string, key: string) => string | undefined

function buildDiagram(paths: FlowPath[], dimensions: string[], label: (dimension: string, key: string) => string, ptr: KeyLookup): Diagram {
  const index = new Map<string, number>()
  const nodes: DiagramNode[] = []
  const links = new Map<string, DiagramLink>()
  const perColumn = dimensions.map(() => 0)

  const nodeFor = (column: number, key: string) => {
    const id = `${column}\u0000${key}`
    const known = index.get(id)
    if (known !== undefined) return known
    index.set(id, nodes.length)
    nodes.push({ name: label(dimensions[column], key), column, color: SERIES_CYCLE[perColumn[column]++ % SERIES_CYCLE.length], ptr: ptr(dimensions[column], key) })
    return nodes.length - 1
  }

  for (const path of paths) {
    const ids = path.keys.map((key, column) => nodeFor(column, key))
    for (let i = 0; i + 1 < ids.length; i++) {
      const id = `${ids[i]}>${ids[i + 1]}`
      const link = links.get(id) ?? { source: ids[i], target: ids[i + 1], value: 0 }
      link.value += path.bytes
      links.set(id, link)
    }
  }

  return { nodes, links: [...links.values()], rows: Math.max(0, ...perColumn) }
}

function useWidth<T extends HTMLElement>() {
  const ref = useRef<T>(null)
  const [width, setWidth] = useState(0)
  useEffect(() => {
    if (!ref.current) return
    const observer = new ResizeObserver(([entry]) => setWidth(Math.floor(entry.contentRect.width)))
    observer.observe(ref.current)
    return () => observer.disconnect()
  }, [])
  return { ref, width }
}

function truncate(text: string, max: number): string {
  return text.length > max ? `${text.slice(0, Math.max(1, max - 1))}…` : text
}

type NodeLayout = { columns: number; maxChars: number }
type NodeShape = SankeyNodeProps & NodeLayout

function DiagramNodeShape({ x, y, width, height, payload, columns, maxChars }: NodeShape) {
  const node = payload as unknown as DiagramNode
  const last = node.column === columns - 1
  const sharesGap = columns > 2 && node.column > 0
  const chars = sharesGap ? Math.max(6, Math.floor(maxChars / 2)) : maxChars
  return (
    <Layer>
      <Rectangle x={x} y={y} width={width} height={height} fill={node.color} fillOpacity={0.9} radius={2} />
      <text
        x={last ? x - 6 : x + width + 6}
        y={y + height / 2}
        textAnchor={last ? 'end' : 'start'}
        dominantBaseline="middle"
        fontSize={11}
        className="fill-foreground"
      >
        {truncate(node.name, chars)}
      </text>
    </Layer>
  )
}

function DiagramLinkShape({ sourceX, sourceY, sourceControlX, targetX, targetY, targetControlX, linkWidth, payload }: SankeyLinkProps) {
  const source = payload.source as unknown as DiagramNode
  return (
    <path
      d={`M${sourceX},${sourceY} C${sourceControlX},${sourceY} ${targetControlX},${targetY} ${targetX},${targetY}`}
      fill="none"
      stroke={source.color}
      strokeOpacity={0.3}
      strokeWidth={Math.max(1, linkWidth)}
    />
  )
}

type LinkEnds = { source?: DiagramNode; target?: DiagramNode }
type TooltipTarget = Partial<DiagramNode> & LinkEnds
type TooltipItem = { payload?: TooltipTarget }
type TooltipEntry = { name?: string; value?: number; payload?: TooltipItem }
type DiagramTooltipShape = { active?: boolean; payload?: TooltipEntry[] }

function DiagramTooltip({ active, payload }: DiagramTooltipShape) {
  const entry = payload?.[0]
  if (!active || !entry) return null
  const inner = entry.payload?.payload
  const link = inner?.source && inner.target
  const title = link ? `${inner.source?.name} → ${inner.target?.name}` : inner?.name ?? entry.name
  const ends = link ? [inner.source, inner.target] : [inner]
  const ptrs = ends.filter((end) => end?.ptr).map((end) => link ? `${end?.name}: ${end?.ptr}` : end?.ptr)
  return (
    <div className="space-y-1 rounded-lg border border-border bg-popover px-3 py-2 text-xs text-popover-foreground shadow-lg">
      <div className="break-all font-mono">{title}</div>
      {ptrs.map((line) => <div key={line} className="break-all font-mono text-muted-foreground">{line}</div>)}
      <div className="tabular-nums text-muted-foreground">{fmtBytes(entry.value ?? 0)}</div>
    </div>
  )
}

type FlowSankeyShape = {
  paths: FlowPath[]
  dimensions: string[]
  total: number
  names: Record<string, string>
  ifaceLabel: (device: string) => string
}

export function FlowSankey({ paths, dimensions, total, names, ifaceLabel }: FlowSankeyShape) {
  const { ref, width } = useWidth<HTMLDivElement>()
  const diagram = useMemo(() => {
    const label = (dimension: string, key: string) => (dimension === 'in_if' || dimension === 'out_if') && key ? ifaceLabel(key) : topKeyLabel(dimension, key)
    const ptr = (dimension: string, key: string) => dimension === 'src' || dimension === 'dst' ? names[key] : undefined
    return buildDiagram(paths, dimensions, label, ptr)
  }, [paths, dimensions, names, ifaceLabel])

  if (paths.length === 0) return <EmptyState className="py-8" title="No Flows" message="No traffic was sampled in this period" />

  const shown = paths.reduce((sum, p) => sum + p.bytes, 0)
  const height = Math.max(240, diagram.rows * 30)
  const columns = dimensions.length
  const maxChars = Math.max(6, Math.floor((width / Math.max(1, columns - 1) - 30) / 6.5))

  return (
    <div className="space-y-2">
      <div ref={ref} className="w-full">
        {width > 0 && (
          <Sankey
            width={width}
            height={height}
            data={diagram}
            nodeWidth={10}
            nodePadding={14}
            margin={{ top: 4, right: 4, bottom: 4, left: 4 }}
            node={(props: SankeyNodeProps) => <DiagramNodeShape {...props} columns={columns} maxChars={maxChars} />}
            link={(props: SankeyLinkProps) => <DiagramLinkShape {...props} />}
          >
            <Tooltip content={<DiagramTooltip />} isAnimationActive={false} />
          </Sankey>
        )}
      </div>
      <p className="text-xs text-muted-foreground">
        {`Top ${paths.length} paths, ${total > 0 ? Math.round((shown / total) * 100) : 0} % of ${fmtBytes(total)} sampled`}
      </p>
    </div>
  )
}
