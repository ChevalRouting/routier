import type { BGPSubTab, OSPFSubTab, PBRSubTab } from '@/components/routing/types'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'

type NameSelectShape = {
  value?: string
  onChange: (v: string | undefined) => void
  names: string[]
  placeholder?: string
}

type TABSShape = { key: Tab; label: string; group: string; title?: string; badge?: number }

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
      <SelectTrigger className="font-mono">
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

export type Tab = 'static' | 'pbr' | 'pbr-maps' | 'pbr-policies' | 'nat' | 'bgp' | 'bgp-neighbors' | 'bgp-afs' | 'bgp-prefix-lists' | 'bgp-route-maps' | 'ospf' | 'ospf-areas' | 'ospf-interfaces' | 'ospf6' | 'ospf6-areas' | 'ospf6-interfaces' | 'vrfs' | 'bfd' | 'radvd'

export const BGP_SECTIONS: Record<string, BGPSubTab> = {
  'bgp': 'general',
  'bgp-neighbors': 'neighbors',
  'bgp-afs': 'afs',
  'bgp-prefix-lists': 'prefix-lists',
  'bgp-route-maps': 'route-maps',
}

export const OSPF_SECTIONS: Record<string, OSPFSubTab> = {
  'ospf': 'general',
  'ospf-areas': 'areas',
  'ospf-interfaces': 'interfaces',
}

export const OSPF6_SECTIONS: Record<string, OSPFSubTab> = {
  'ospf6': 'general',
  'ospf6-areas': 'areas',
  'ospf6-interfaces': 'interfaces',
}

export const PBR_SECTIONS: Record<string, PBRSubTab> = {
  'pbr': 'nexthop-groups',
  'pbr-maps': 'maps',
  'pbr-policies': 'policies',
}

export const TABS: TABSShape[] = [
  { key: 'static', label: 'Static Routes', group: 'Routes' },
  { key: 'nat', label: 'NAT', group: 'Routes' },
  { key: 'bgp', label: 'General', group: 'BGP', title: 'BGP' },
  { key: 'bgp-neighbors', label: 'Neighbors', group: 'BGP' },
  { key: 'bgp-afs', label: 'Address Families', group: 'BGP' },
  { key: 'bgp-prefix-lists', label: 'Prefix Lists', group: 'BGP' },
  { key: 'bgp-route-maps', label: 'Route Maps', group: 'BGP' },
  { key: 'ospf', label: 'General', group: 'OSPF', title: 'OSPF' },
  { key: 'ospf-areas', label: 'Areas', group: 'OSPF' },
  { key: 'ospf-interfaces', label: 'Interfaces', group: 'OSPF' },
  { key: 'ospf6', label: 'General', group: 'OSPFv3', title: 'OSPFv3' },
  { key: 'ospf6-areas', label: 'Areas', group: 'OSPFv3' },
  { key: 'ospf6-interfaces', label: 'Interfaces', group: 'OSPFv3' },
  { key: 'pbr', label: 'Nexthop Groups', group: 'Policy Routing' },
  { key: 'pbr-maps', label: 'Maps', group: 'Policy Routing' },
  { key: 'pbr-policies', label: 'Policies', group: 'Policy Routing' },
  { key: 'vrfs', label: 'VRFs', group: 'Infrastructure' },
  { key: 'bfd', label: 'BFD', group: 'Infrastructure' },
  { key: 'radvd', label: 'RADVD', group: 'Infrastructure' },
]
