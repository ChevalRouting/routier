import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

export function ValueNode({ val, depth = 0 }: { val: unknown; depth?: number }) {
  const [open, setOpen] = useState(depth < 2)

  if (val === null || val === undefined) return <span className="text-muted-foreground italic">null</span>
  if (typeof val === 'boolean') return <Badge variant={val ? 'default' : 'secondary'}>{String(val)}</Badge>
  if (typeof val === 'number') return <span className="text-blue-500 font-mono">{val}</span>
  if (typeof val === 'string') {
    if (val === '') return <span className="text-muted-foreground italic">empty</span>
    return <span className="font-mono text-emerald-600 dark:text-emerald-400 break-all">{val}</span>
  }
  if (Array.isArray(val)) {
    if (val.length === 0) return <span className="text-muted-foreground italic">[]</span>
    return (
      <div>
        <button
          type="button"
          onClick={() => setOpen((o) => !o)}
          className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          {open ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
          {val.length} item{val.length !== 1 ? 's' : ''}
        </button>
        {open && (
          <div className="mt-1 pl-4 border-l border-border space-y-0.5">
            {val.map((item, i) => (
              <div key={i} className="flex gap-2 items-start text-sm">
                <span className="text-muted-foreground shrink-0">{i}:</span>
                <ValueNode val={item} depth={depth + 1} />
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }
  if (typeof val === 'object') {
    const entries = Object.entries(val as Record<string, unknown>).filter(
      ([, v]) => v !== null && v !== undefined
    )
    if (entries.length === 0) return <span className="text-muted-foreground italic">{'{}'}</span>
    return (
      <div>
        <button
          type="button"
          onClick={() => setOpen((o) => !o)}
          className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          {open ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
          {entries.length} field{entries.length !== 1 ? 's' : ''}
        </button>
        {open && (
          <div className="mt-1 pl-4 border-l border-border space-y-1">
            {entries.map(([k, v]) => (
              <div key={k} className="flex gap-2 items-start text-sm">
                <span className="text-foreground font-medium shrink-0">{k}:</span>
                <ValueNode val={v} depth={depth + 1} />
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }
  return <span>{String(val)}</span>
}

export function SectionCard({ name, value, defaultOpen = true }: { name: string; value: unknown; defaultOpen?: boolean }) {
  const [open, setOpen] = useState(defaultOpen)
  const isEmpty =
    value === null ||
    value === undefined ||
    (Array.isArray(value) && value.length === 0) ||
    (typeof value === 'object' && !Array.isArray(value) && Object.keys(value as object).length === 0)

  return (
    <Card className={cn(isEmpty && 'opacity-50')}>
      <CardHeader className="py-3 px-4">
        <div className="flex items-center justify-between">
          <button
            type="button"
            onClick={() => setOpen((o) => !o)}
            className="flex items-center gap-2 text-sm font-semibold hover:text-primary"
          >
            {open ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            <CardTitle className="text-sm">{name}</CardTitle>
          </button>
          {isEmpty && <Badge variant="outline" className="text-xs">empty</Badge>}
        </div>
      </CardHeader>
      {open && !isEmpty && (
        <CardContent className="px-4 pb-4 pt-0">
          <ValueNode val={value} depth={1} />
        </CardContent>
      )}
    </Card>
  )
}

const SECTION_ORDER = [
  'version', 'hostname', 'interfaces', 'tunnels', 'routing', 'wireguard',
  'nftables', 'sysctl', 'dns', 'users', 'services', 'logging',
]

export function orderedSections(data: Record<string, unknown> | null | undefined): [string, unknown][] {
  if (!data) return []
  return [
    ...SECTION_ORDER.filter((k) => k in data),
    ...Object.keys(data).filter((k) => !SECTION_ORDER.includes(k)),
  ].map((k) => [k, data[k]] as [string, unknown])
}

export function ConfigSections({
  data,
  defaultOpen = true,
}: {
  data: Record<string, unknown> | null | undefined
  defaultOpen?: boolean
}) {
  const sections = orderedSections(data)
  if (sections.length === 0) return <div className="text-sm text-muted-foreground">empty</div>
  return (
    <div className="md:columns-2 [column-gap:0.75rem]">
      {sections.map(([k, v]) => (
        <div key={k} className="mb-3 break-inside-avoid">
          <SectionCard name={k} value={v} defaultOpen={defaultOpen} />
        </div>
      ))}
    </div>
  )
}
