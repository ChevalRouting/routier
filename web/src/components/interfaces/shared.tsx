import { onActivate } from '@/lib/utils'
import type { TypesSystemNic as SystemNic } from '@/api'
import { Badge, Button, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, TableCell, TableRow } from 'cheval-ui'
import {
  Trash2, X
} from 'lucide-react'

type HwFieldShape = { label: string; value: string; mono?: boolean }

export interface Vlan {
  id: number
}

export interface Bridge {
  members: string[]
  stp: boolean
}

export interface Bond {
  members: string[]
  mode?: string
  miimon?: number
  xmit_hash_policy?: string
  lacp_rate?: string
  updelay?: number
  downdelay?: number
  min_links?: number
  primary?: string
}

export interface Vxlan {
  vni: number
  external?: boolean
  vnifilter?: boolean
  local?: string
  remote?: string
  group?: string
  vtep?: string
  port?: number
  learning?: boolean
}

export interface VNet {
  vni: number
  vlan: number
}

export interface Iface {
  select: string
  type?: string
  vrf?: string
  addresses: string[]
  dhcp_options: string[]
  mtu: number
  vlan?: Vlan
  bridge?: Bridge
  bond?: Bond
  vnet?: VNet
  vxlan?: Vxlan
  mode?: string
  local?: string
  remote?: string
  ttl?: number
  friend?: string
}

export const TUNNEL_MODES = ['sit', 'gre', 'gretap', 'vti', 'ipip', 'ip6tnl', 'ip6ip6', 'ip6gre']

export const IFACE_TYPES = [
  { value: 'physical', label: 'Physical (select device)' },
  { value: 'dummy', label: 'Dummy (virtual)' },
  { value: 'bridge', label: 'Bridge (virtual)' },
  { value: 'bond', label: 'Bond (link aggregation)' },
  { value: 'vlan', label: 'VLAN (tagged subinterface)' },
  { value: 'vxlan', label: 'VXLAN (virtual)' },
  { value: 'vxlan-bridge', label: 'VXLAN bridge (EVPN transport)' },
  { value: 'vnet', label: 'VNet (VXLAN network)' },
  { value: 'tunnel', label: 'Tunnel (SIT/GRE/IPIP/IP6)' },
]

export const BOND_MODES = ['balance-rr', 'active-backup', 'balance-xor', 'broadcast', '802.3ad', 'balance-tlb', 'balance-alb']

export const BOND_HASH_POLICIES = ['layer2', 'layer2+3', 'layer3+4', 'encap2+3', 'encap3+4', 'vlan+srcmac']

export const BOND_LACP_RATES = ['slow', 'fast']

export type IfaceMap = Record<string, Iface>

export function emptyIface(): Iface {
  return { select: '', addresses: [], dhcp_options: [], mtu: 0 }
}

export function emptyVxlan(): Vxlan {
  return { vni: 0 }
}

export function emptyBond(): Bond {
  return { members: [], mode: 'balance-rr' }
}

export interface IfaceFormProps {
  name: string
  iface: Iface
  vrfNames: string[]
  transports?: string[]
  parents: string[]
  nics: SystemNic[]
  onChange: (updated: Iface) => void
  onNameChange?: (n: string) => void
  onAdd?: () => void
  onDone: () => void
}

export function nicLabel(nic: SystemNic): string {
  const parts = [nic.name]
  if (nic.mac) parts.push(nic.mac)
  if (nic.driver) parts.push(nic.driver)
  else if (!nic.physical) parts.push(nic.operstate ?? '')
  return parts.filter(Boolean).join(' · ')
}

export function nicSelector(nic: SystemNic): string {
  return nic.mac ? `mac(${nic.mac})` : nic.name
}

const MAC_SELECTOR = /^mac\(([0-9a-fA-F]{2}(?::[0-9a-fA-F]{2}){5})\)$/

export function nicForSelector(nics: SystemNic[], sel: string): SystemNic | undefined {
  const s = sel.trim()
  if (!s) return undefined

  const mac = MAC_SELECTOR.exec(s)
  if (mac) {
    const want = mac[1].toLowerCase()
    return nics.find((n) => (n.mac ?? '').toLowerCase() === want)
  }

  return nics.find((n) => n.name === s)
}

export function selectorLabel(nics: SystemNic[], sel: string): string {
  const nic = nicForSelector(nics, sel)
  return nic ? nicLabel(nic) : sel
}

type SelectorOption = { value: string; label: string }

export function selectorOptions(nics: SystemNic[], extra: string[]): SelectorOption[] {
  const names = new Set(nics.map((n) => n.name))
  const opts: SelectorOption[] = nics.map((n) => ({ value: nicSelector(n), label: nicLabel(n) }))
  for (const e of extra) {
    if (e && !names.has(e)) opts.push({ value: e, label: e })
  }

  return opts
}

type MemberPickerShape = {
  members: string[]
  nics: SystemNic[]
  extra: string[]
  onChange: (members: string[]) => void
}

export function MemberPicker({ members, nics, extra, onChange }: MemberPickerShape) {
  const options = selectorOptions(nics, extra).filter((o) => !members.includes(o.value))

  return (
    <div className="space-y-2">
      {members.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {members.map((m) => (
            <Badge key={m} variant="secondary" className="gap-1 font-mono">
              {selectorLabel(nics, m)}
              <Button variant="ghost" size="icon" onClick={() => onChange(members.filter((x) => x !== m))} className="h-4 w-4 p-0 text-muted-foreground hover:text-destructive" aria-label={`Remove ${m}`} title="Remove">
                <X className="h-3 w-3" />
              </Button>
            </Badge>
          ))}
        </div>
      )}
      <Select value="" onValueChange={(v) => onChange([...members, v])}>
        <SelectTrigger className="h-8 font-mono text-xs">
          <SelectValue placeholder="Add member…" />
        </SelectTrigger>
        <SelectContent>
          {options.map((o) => (
            <SelectItem key={o.value} value={o.value} className="font-mono text-xs">{o.label}</SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

type NicSelectShape = {
  value: string
  nics: SystemNic[]
  extra: string[]
  placeholder?: string
  onChange: (sel: string) => void
}

export function NicSelect({ value, nics, extra, placeholder = '- select -', onChange }: NicSelectShape) {
  const options = selectorOptions(nics, extra)
  const known = options.some((o) => o.value === value)

  return (
    <Select value={value || '__none__'} onValueChange={(v) => onChange(v === '__none__' ? '' : v)}>
      <SelectTrigger className="font-mono">
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="__none__" className="font-mono">{placeholder}</SelectItem>
        {value && !known && <SelectItem value={value} className="font-mono">{value}</SelectItem>}
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value} className="font-mono">{o.label}</SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export interface IfaceCardProps {
  name: string
  iface: Iface
  nic?: SystemNic
  hasVrrp?: boolean
  onEdit: () => void
  onDelete: () => void
}

export function HwField({ label, value, mono }: HwFieldShape) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={`truncate ${mono ? 'font-mono' : ''}`}>{value}</div>
    </div>
  )
}

export const NEW_KEY = '__new__'

export function IfaceRow({ name, iface, nic, onEdit, onDelete }: IfaceCardProps) {
  const addresses = iface.addresses ?? []
  const up = nic ? nic.operstate === 'up' || !!nic.carrier : undefined

  return (
    <TableRow role="button" tabIndex={0} onClick={onEdit} onKeyDown={onActivate(onEdit)} className="cursor-pointer">
      <TableCell className="font-mono font-medium">
        <span className="inline-flex items-center gap-2">
          {up !== undefined && (
            <span className={`h-2 w-2 shrink-0 rounded-full ${up ? 'bg-success' : 'bg-muted-foreground'}`} title={up ? 'link up' : 'link down'} />
          )}
          {name}
        </span>
      </TableCell>
      <TableCell className="font-mono text-xs text-muted-foreground">{nic && nic.name !== name ? nic.name : '-'}</TableCell>
      <TableCell><Badge variant="secondary" className="font-mono">{iface.type || 'physical'}</Badge></TableCell>
      <TableCell className="font-mono text-xs">
        {addresses.length === 0 ? <span className="text-muted-foreground">-</span> : addresses.join(', ')}
      </TableCell>
      <TableCell className="font-mono text-xs">{iface.vrf || <span className="text-muted-foreground">-</span>}</TableCell>
      <TableCell className="w-[1%] whitespace-nowrap text-right">
        <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive"
          onClick={(e) => { e.stopPropagation(); onDelete() }} aria-label="Remove Interface" title="Remove Interface">
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </TableCell>
    </TableRow>
  )
}

export function matchNic(nics: SystemNic[], iface: Iface, name: string): SystemNic | undefined {
  const byKey = nics.find((n) => n.name === name)
  if (byKey) return byKey

  if (iface.type === 'vnet' || iface.type === 'vxlan-bridge') return undefined

  const sel = iface.select?.trim()
  if (!sel) return undefined

  const mac = /^mac\(([0-9a-fA-F]{2}(?::[0-9a-fA-F]{2}){5})\)$/.exec(sel)
  if (mac) {
    const want = mac[1].toLowerCase()
    return nics.find((n) => (n.mac ?? '').toLowerCase() === want)
  }

  const indexed = /^([a-zA-Z]+)\[(\d+)\]$/.exec(sel)
  if (indexed) {
    const [, prefix, idx] = indexed
    return nics
      .filter((n) => n.name.startsWith(prefix))
      .sort((a, b) => a.name.localeCompare(b.name))[Number(idx)]
  }

  return nics.find((n) => n.name === sel)
}
