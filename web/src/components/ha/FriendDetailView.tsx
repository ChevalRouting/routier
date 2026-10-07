import { ValueNode } from '@/components/ConfigView'
import { Card, CardContent, CardHeader, CardTitle } from 'cheval-ui'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useState, type ReactNode } from 'react'

type FieldShape = { label: string; value?: ReactNode; mono?: boolean }

type TABSShape = { key: Tab; label: string }

type WgCardShape = { name: string; wg: WgInstance }

type TunCardShape = { name: string; tun: TunInstance }

export function Field({ label, value, mono }: FieldShape) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={`truncate ${mono ? 'font-mono text-xs' : ''}`}>{value ?? '-'}</div>
    </div>
  )
}

export type Tab = 'overview' | 'wireguard' | 'tunnel' | 'conntrack' | 'vrrp'
export const TABS: TABSShape[] = [
  { key: 'overview',  label: 'Overview' },
  { key: 'wireguard', label: 'WireGuard' },
  { key: 'tunnel',    label: 'Tunnel' },
  { key: 'conntrack', label: 'Conntrack' },
  { key: 'vrrp',      label: 'VRRP' },
]

export interface WgPeer { name?: string; public_key?: string; endpoint?: string; allowed_ips?: string[]; keepalive?: number }
export interface WgInstance {
  friend?: string
  listen_port?: number
  addresses?: string[]
  table?: string
  mtu?: number
  peers?: WgPeer[]
}

export function WgCard({ name, wg }: WgCardShape) {
  const [open, setOpen] = useState(false)
  const endpoints = (wg.peers ?? []).map((p) => p.endpoint).filter(Boolean).join(', ')
  const allowed = (wg.peers ?? []).flatMap((p) => p.allowed_ips ?? []).join(', ')
  return (
    <Card>
      <button type="button" onClick={() => setOpen((o) => !o)} className="w-full text-left">
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center justify-between text-base">
            <span className="font-mono">{name}</span>
            {open ? <ChevronDown className="h-4 w-4 text-muted-foreground" /> : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
          </CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm md:grid-cols-4">
          <Field label="Addresses" value={(wg.addresses ?? []).join(', ') || undefined} mono />
          <Field label="Listen port" value={wg.listen_port} />
          <Field label="Peer endpoint" value={endpoints || undefined} mono />
          <Field label="Allowed IPs" value={allowed || undefined} mono />
        </CardContent>
      </button>
      {open && (
        <CardContent className="pt-4">
          <ValueNode val={wg} depth={1} />
        </CardContent>
      )}
    </Card>
  )
}

export interface TunInstance {
  friend?: string
  mode?: string
  local?: string
  remote?: string
  ttl?: number
  addresses?: string[]
  mtu?: number
}

export function TunCard({ name, tun }: TunCardShape) {
  const [open, setOpen] = useState(false)
  return (
    <Card>
      <button type="button" onClick={() => setOpen((o) => !o)} className="w-full text-left">
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center justify-between text-base">
            <span className="font-mono">{name}</span>
            {open ? <ChevronDown className="h-4 w-4 text-muted-foreground" /> : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
          </CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm md:grid-cols-4">
          <Field label="Mode" value={tun.mode} />
          <Field label="Addresses" value={(tun.addresses ?? []).join(', ') || undefined} mono />
          <Field label="Local" value={tun.local || undefined} mono />
          <Field label="Remote" value={tun.remote || undefined} mono />
        </CardContent>
      </button>
      {open && (
        <CardContent className="pt-4">
          <ValueNode val={tun} depth={1} />
        </CardContent>
      )}
    </Card>
  )
}

export interface VrrpConfigInstance { name?: string; friend?: string; id?: number; interface?: string; vips?: string[]; priority?: number }

export { ConntrackTab } from './FriendDetailViewParts/ConntrackTab'
export { FriendDetailView } from './FriendDetailViewParts/FriendDetailView'
export { VrrpTab } from './FriendDetailViewParts/VrrpTab'
