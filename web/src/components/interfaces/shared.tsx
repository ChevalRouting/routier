import type { TypesSystemNic as SystemNic } from '@/api'
import { Badge, Button, TableCell, TableRow } from 'cheval-ui'
import {
  Trash2
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
  vxlan?: Vxlan
  mode?: string
  local?: string
  remote?: string
  ttl?: number
}

export interface Tunnel {
  mode: string
  local: string
  remote: string
  ttl: number
  addresses: string[]
  mtu: number
}

export type TunnelMap = Record<string, Tunnel>

export const TUNNEL_MODES = ['sit', 'gre', 'ipip', 'ip6tnl', 'ip6ip6', 'ip6gre']

export const IFACE_TYPES = [
  { value: 'physical', label: 'Physical (select device)' },
  { value: 'dummy', label: 'Dummy (virtual)' },
  { value: 'bridge', label: 'Bridge (virtual)' },
  { value: 'bond', label: 'Bond (link aggregation)' },
  { value: 'vlan', label: 'VLAN (tagged subinterface)' },
  { value: 'vxlan', label: 'VXLAN (virtual)' },
  { value: 'tunnel', label: 'Tunnel (SIT/GRE/IPIP/IP6)' },
]

export const BOND_MODES = ['balance-rr', 'active-backup', 'balance-xor', 'broadcast', '802.3ad', 'balance-tlb', 'balance-alb']

export const BOND_HASH_POLICIES = ['layer2', 'layer2+3', 'layer3+4', 'encap2+3', 'encap3+4', 'vlan+srcmac']

export const BOND_LACP_RATES = ['slow', 'fast']

export function tunnelToIface(t: Tunnel): Iface {
  return {
    select: '', type: 'tunnel', addresses: t.addresses ?? [], dhcp_options: [], mtu: t.mtu ?? 0,
    mode: t.mode, local: t.local, remote: t.remote, ttl: t.ttl,
  }
}

export function ifaceToTunnel(i: Iface): Tunnel {
  return {
    mode: i.mode || 'gre', local: i.local ?? '', remote: i.remote ?? '', ttl: i.ttl ?? 0,
    addresses: i.addresses ?? [], mtu: i.mtu ?? 0,
  }
}

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
      <div className="text-[10px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className={`truncate ${mono ? 'font-mono' : ''}`}>{value}</div>
    </div>
  )
}

export const NEW_KEY = '__new__'

export function IfaceRow({ name, iface, nic, onEdit, onDelete }: IfaceCardProps) {
  const addresses = iface.addresses ?? []
  const up = nic ? nic.operstate === 'up' || !!nic.carrier : undefined

  return (
    <TableRow onClick={onEdit} className="cursor-pointer">
      <TableCell className="font-mono font-medium">
        <span className="inline-flex items-center gap-2">
          {up !== undefined && (
            <span className={`h-2 w-2 shrink-0 rounded-full ${up ? 'bg-success' : 'bg-muted-foreground'}`} title={up ? 'link up' : 'link down'} />
          )}
          {name}
        </span>
      </TableCell>
      <TableCell className="font-mono text-xs text-muted-foreground">{nic && nic.name !== name ? nic.name : '-'}</TableCell>
      <TableCell><Badge variant="secondary" className="font-mono text-xs">{iface.type || 'physical'}</Badge></TableCell>
      <TableCell className="font-mono text-xs">
        {addresses.length === 0 ? <span className="text-muted-foreground">-</span> : addresses.join(', ')}
      </TableCell>
      <TableCell className="font-mono text-xs">{iface.vrf || <span className="text-muted-foreground">-</span>}</TableCell>
      <TableCell className="w-[1%] whitespace-nowrap text-right">
        <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive"
          onClick={(e) => { e.stopPropagation(); onDelete() }}>
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </TableCell>
    </TableRow>
  )
}

export function matchNic(nics: SystemNic[], iface: Iface, name: string): SystemNic | undefined {
  const byKey = nics.find((n) => n.name === name)
  if (byKey) return byKey

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
