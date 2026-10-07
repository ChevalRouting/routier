import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'

type NameSelectShape = {
  value?: string
  onChange: (v: string | undefined) => void
  names: string[]
  placeholder?: string
}

type TABSShape = { key: Tab; label: string }

let nextId = 1

export function newId(): number {
  return nextId++
}

export function NameSelect({
  value, onChange, names, placeholder = '- none -',
}: NameSelectShape) {
  return (
    <Select
      value={value ?? '__none__'}
      onValueChange={(v) => onChange(v === '__none__' ? undefined : v)}
    >
      <SelectTrigger className="h-8 text-xs font-mono">
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="__none__">{placeholder}</SelectItem>
        {names.map((n) => (
          <SelectItem key={n} value={n}>{n}</SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export type KVPair = { _id: number; key: string; value: string }

export function toRecord(pairs: KVPair[]): Record<string, string> {
  const r: Record<string, string> = {}
  for (const { key, value } of pairs) {
    if (key) r[key] = value
  }
  return r
}

export function fromRecord(rec: Record<string, string> | undefined): KVPair[] {
  return Object.entries(rec ?? {}).map(([key, value]) => ({ _id: newId(), key, value }))
}

export type Tab = 'static' | 'bgp' | 'ospf' | 'ospf6' | 'bfd' | 'radvd' | 'vrfs' | 'pbr' | 'nat'

export const TABS: TABSShape[] = [
  { key: 'static', label: 'Static Routes' },
  { key: 'bgp', label: 'BGP' },
  { key: 'ospf', label: 'OSPF' },
  { key: 'ospf6', label: 'OSPFv3' },
  { key: 'bfd', label: 'BFD' },
  { key: 'radvd', label: 'RADVD' },
  { key: 'vrfs', label: 'VRFs' },
  { key: 'pbr', label: 'PBR' },
  { key: 'nat', label: 'NAT' },
]
