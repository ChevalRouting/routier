import { useState } from 'react'
import { toast } from 'sonner'
import type { TypesSystemNic as SystemNic } from '@/api'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { useDataRefresh } from '@/lib/dataVersion'
import { usePageSave } from '@/lib/usePageSave'
import { SaveButton } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { Sheet } from 'cheval-ui'
import { TagInput } from 'cheval-ui'
import { Segmented } from 'cheval-ui'
import { Table, TableHeader, TableBody, TableHead, TableRow, TableCell } from 'cheval-ui'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from 'cheval-ui'
import {
  Plus, Trash2, ChevronDown, ChevronRight, Network,
} from 'lucide-react'
import { PageHeader } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { Pagination, usePagination } from 'cheval-ui'
import { checkInterfaceAddress, checkIP, checkMulticastIP, checkPort, checkVLANId } from '@/lib/validate'
import { ReloadButton } from 'cheval-ui'
import { PreferencesGroup, EntryRow, ComboRow } from 'cheval-ui'
import { Switch } from 'cheval-ui'
import { Spinner } from 'cheval-ui'

interface Vlan {
  id: number
}

interface Bridge {
  members: string[]
  stp: boolean
}

interface Bond {
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

interface Vxlan {
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

interface Iface {
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

interface Tunnel {
  mode: string
  local: string
  remote: string
  ttl: number
  addresses: string[]
  mtu: number
}

type TunnelMap = Record<string, Tunnel>

const TUNNEL_MODES = ['sit', 'gre', 'ipip', 'ip6tnl', 'ip6ip6', 'ip6gre']

const IFACE_TYPES = [
  { value: 'physical', label: 'Physical (select device)' },
  { value: 'dummy', label: 'Dummy (virtual)' },
  { value: 'bridge', label: 'Bridge (virtual)' },
  { value: 'bond', label: 'Bond (link aggregation)' },
  { value: 'vlan', label: 'VLAN (tagged subinterface)' },
  { value: 'vxlan', label: 'VXLAN (virtual)' },
  { value: 'tunnel', label: 'Tunnel (SIT/GRE/IPIP/IP6)' },
]

const BOND_MODES = ['balance-rr', 'active-backup', 'balance-xor', 'broadcast', '802.3ad', 'balance-tlb', 'balance-alb']
const BOND_HASH_POLICIES = ['layer2', 'layer2+3', 'layer3+4', 'encap2+3', 'encap3+4', 'vlan+srcmac']
const BOND_LACP_RATES = ['slow', 'fast']

function tunnelToIface(t: Tunnel): Iface {
  return {
    select: '', type: 'tunnel', addresses: t.addresses ?? [], dhcp_options: [], mtu: t.mtu ?? 0,
    mode: t.mode, local: t.local, remote: t.remote, ttl: t.ttl,
  }
}

function ifaceToTunnel(i: Iface): Tunnel {
  return {
    mode: i.mode || 'gre', local: i.local ?? '', remote: i.remote ?? '', ttl: i.ttl ?? 0,
    addresses: i.addresses ?? [], mtu: i.mtu ?? 0,
  }
}

type IfaceMap = Record<string, Iface>

function emptyIface(): Iface {
  return { select: '', addresses: [], dhcp_options: [], mtu: 0 }
}

function emptyVxlan(): Vxlan {
  return { vni: 0 }
}

function emptyBond(): Bond {
  return { members: [], mode: 'balance-rr' }
}

interface IfaceFormProps {
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

function nicLabel(nic: SystemNic): string {
  const parts = [nic.name]
  if (nic.mac) parts.push(nic.mac)
  if (nic.driver) parts.push(nic.driver)
  else if (!nic.physical) parts.push(nic.operstate ?? '')
  return parts.filter(Boolean).join(' · ')
}

function nicSelector(nic: SystemNic): string {
  return nic.mac ? `mac(${nic.mac})` : nic.name
}

function IfaceForm({ name, iface, vrfNames, parents, nics, onChange, onNameChange, onAdd, onDone }: IfaceFormProps) {
  const [bridgeOpen, setBridgeOpen] = useState(false)

  const isVirtual = iface.type === 'dummy' || iface.type === 'bridge' || iface.type === 'vxlan' || iface.type === 'bond'
  const isDummy = iface.type === 'dummy'
  const isVxlan = iface.type === 'vxlan'
  const isBond = iface.type === 'bond'
  const isVlan = iface.type === 'vlan'
  const isTunnel = iface.type === 'tunnel'
  const selectedNic = matchNic(nics, iface, name)
  const parentOptions = parents.filter((p) => p !== name)

  const set = <K extends keyof Iface>(key: K, val: Iface[K]) =>
    onChange({ ...iface, [key]: val })

  const setVxlan = (patch: Partial<Vxlan>) =>
    onChange({ ...iface, vxlan: { ...(iface.vxlan ?? emptyVxlan()), ...patch } })

  const setBond = (patch: Partial<Bond>) =>
    onChange({ ...iface, bond: { ...(iface.bond ?? emptyBond()), ...patch } })

  const hasBridge = !!iface.bridge
  const toggleBridge = () => {
    if (hasBridge) {
      const { bridge: _b, ...rest } = iface
      onChange(rest as Iface)
    } else {
      onChange({ ...iface, bridge: { members: [], stp: false } })
      setBridgeOpen(true)
    }
  }

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        {onNameChange !== undefined && (
          <EntryRow
            title="Name"
            autoFocus
            value={name}
            onChange={(e) => onNameChange(e.target.value)}
            placeholder="eth0"
            className="font-mono"
          />
        )}
        <ComboRow title="Type">
          <Select
            value={iface.type || 'physical'}
            onValueChange={(v) => onChange({
              ...iface,
              type: v === 'physical' ? undefined : v,
              vxlan: v === 'vxlan' ? (iface.vxlan ?? emptyVxlan()) : undefined,
              bond: v === 'bond' ? (iface.bond ?? emptyBond()) : undefined,
              vlan: v === 'vlan' ? (iface.vlan ?? { id: 0 }) : undefined,
              bridge: v === 'vxlan' || v === 'tunnel' || v === 'bond' || v === 'vlan' ? undefined : iface.bridge,
              mode: v === 'tunnel' ? (iface.mode ?? 'gre') : undefined,
              select: (v === 'vlan') !== (iface.type === 'vlan') ? '' : iface.select,
            })}
          >
            <SelectTrigger className="font-mono">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {IFACE_TYPES.map((t) => (
                <SelectItem key={t.value} value={t.value} className="font-mono">{t.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </ComboRow>
        {!isVirtual && !isTunnel && !isVlan && (
          <ComboRow title="Physical device">
            <Select value={selectedNic?.name ?? ''} onValueChange={(v) => set('select', nicSelector(nics.find((n) => n.name === v) ?? { name: v }))}>
              <SelectTrigger className="font-mono">
                <SelectValue placeholder="- select a device -" />
              </SelectTrigger>
              <SelectContent>
                {iface.select && !selectedNic && (
                  <SelectItem value={iface.select} className="font-mono">{iface.select}</SelectItem>
                )}
                {nics.map((nic) => (
                  <SelectItem key={nic.name} value={nic.name} className="font-mono">{nicLabel(nic)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </ComboRow>
        )}
        {isVlan && (
          <ComboRow title="Parent interface">
            <Select value={iface.select || ''} onValueChange={(v) => set('select', v)}>
              <SelectTrigger className="font-mono">
                <SelectValue placeholder="- select a parent -" />
              </SelectTrigger>
              <SelectContent>
                {iface.select && !parentOptions.includes(iface.select) && (
                  <SelectItem value={iface.select} className="font-mono">{iface.select}</SelectItem>
                )}
                {parentOptions.map((p) => (
                  <SelectItem key={p} value={p} className="font-mono">{p}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </ComboRow>
        )}
        <EntryRow
          title="MTU (0 = auto)"
          value={iface.mtu || ''}
          onChange={(e) => set('mtu', Number(e.target.value) || 0)}
          placeholder="0"
          className="font-mono"
        />
        <ComboRow title="VRF">
          <Select
            value={iface.vrf || '__none__'}
            onValueChange={(v) => onChange({ ...iface, vrf: v === '__none__' ? undefined : v })}
          >
            <SelectTrigger className="font-mono">
              <SelectValue placeholder="- none -" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">- none -</SelectItem>
              {vrfNames.map((n) => (
                <SelectItem key={n} value={n}>{n}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </ComboRow>
      </PreferencesGroup>

      {isTunnel && (
        <PreferencesGroup title="Tunnel" description="Encapsulate traffic between two endpoints.">
          <ComboRow title="Mode">
            <Select value={iface.mode || 'gre'} onValueChange={(v) => set('mode', v)}>
              <SelectTrigger className="font-mono"><SelectValue /></SelectTrigger>
              <SelectContent>
                {TUNNEL_MODES.map((m) => <SelectItem key={m} value={m} className="font-mono">{m}</SelectItem>)}
              </SelectContent>
            </Select>
          </ComboRow>
          <EntryRow title="TTL (0 = inherit)" value={iface.ttl || ''} onChange={(e) => set('ttl', Number(e.target.value) || 0)} placeholder="0" className="font-mono" />
          <EntryRow title="Local endpoint" value={iface.local ?? ''} onChange={(e) => set('local', e.target.value)} placeholder="203.0.113.1" className="font-mono" />
          <EntryRow title="Remote endpoint" value={iface.remote ?? ''} onChange={(e) => set('remote', e.target.value)} placeholder="203.0.113.2" className="font-mono" />
        </PreferencesGroup>
      )}

      {isVxlan && (
        <PreferencesGroup title="VXLAN" description="Configure the virtual network identifier and VTEP endpoints.">
          <div className="flex min-h-[3.25rem] items-center gap-3 px-4 py-2.5">
            <div className="min-w-0 flex-1">
              <div className="text-sm font-medium">External (collect metadata)</div>
              <div className="mt-0.5 text-xs text-muted-foreground">Receive every VNI on the UDP port; the VNI comes from tunnel metadata. Only one per port unless VNI filter is on.</div>
            </div>
            <Switch checked={!!iface.vxlan?.external} onCheckedChange={(v) => setVxlan(v ? { external: true, vni: 0 } : { external: undefined, vnifilter: undefined })} />
          </div>
          {iface.vxlan?.external && (
            <div className="flex min-h-[3.25rem] items-center gap-3 px-4 py-2.5">
              <div className="min-w-0 flex-1">
                <div className="text-sm font-medium">VNI filter</div>
                <div className="mt-0.5 text-xs text-muted-foreground">Let several external devices share the UDP port, each filtering its own VNIs.</div>
              </div>
              <Switch checked={!!iface.vxlan?.vnifilter} onCheckedChange={(v) => setVxlan({ vnifilter: v || undefined })} />
            </div>
          )}
          {!iface.vxlan?.external && (
            <EntryRow
              title="VNI"
              inputMode="numeric"
              value={iface.vxlan?.vni || ''}
              onChange={(e) => setVxlan({ vni: Number(e.target.value) || 0 })}
              placeholder="100"
              className="font-mono"
              error={!iface.vxlan?.vni || iface.vxlan.vni < 1 || iface.vxlan.vni > 16777215 ? 'VNI must be 1 to 16777215' : null}
            />
          )}
          <EntryRow title="Local address" value={iface.vxlan?.local ?? ''} onChange={(e) => setVxlan({ local: e.target.value || undefined })} placeholder="192.0.2.1" className="font-mono" error={checkIP(iface.vxlan?.local ?? '')} />
          <EntryRow title="Remote address" value={iface.vxlan?.remote ?? ''} onChange={(e) => setVxlan({ remote: e.target.value || undefined })} placeholder="192.0.2.2" className="font-mono" error={iface.vxlan?.group ? 'Remote and group are mutually exclusive' : checkIP(iface.vxlan?.remote ?? '')} />
          <EntryRow title="Multicast group" value={iface.vxlan?.group ?? ''} onChange={(e) => setVxlan({ group: e.target.value || undefined })} placeholder="239.1.1.1" className="font-mono" error={iface.vxlan?.remote ? 'Remote and group are mutually exclusive' : checkMulticastIP(iface.vxlan?.group ?? '')} />
          <EntryRow title="VTEP interface" value={iface.vxlan?.vtep ?? ''} onChange={(e) => setVxlan({ vtep: e.target.value || undefined })} placeholder="underlay" className="font-mono" />
          <EntryRow title="UDP port (0 = 4789)" inputMode="numeric" value={iface.vxlan?.port || ''} onChange={(e) => setVxlan({ port: Number(e.target.value) || undefined })} placeholder="4789" className="font-mono" error={checkPort(String(iface.vxlan?.port || ''))} />
          <div className="flex min-h-[3.25rem] items-center gap-3 px-4 py-2.5">
            <div className="min-w-0 flex-1">
              <div className="text-sm font-medium">MAC learning</div>
              <div className="mt-0.5 text-xs text-muted-foreground">Enabled by default; EVPN fabrics commonly disable it.</div>
            </div>
            <Switch checked={iface.vxlan?.learning !== false} onCheckedChange={(v) => setVxlan({ learning: v })} />
          </div>
        </PreferencesGroup>
      )}

      {isBond && (
        <PreferencesGroup title="Bond" description="Aggregate several NICs for redundancy or throughput.">
          <div className="px-4 py-3 space-y-1.5">
            <Label className="text-xs">Members</Label>
            <TagInput values={iface.bond?.members ?? []} onChange={(v) => setBond({ members: v })} placeholder="eth0" mono />
          </div>
          <ComboRow title="Mode">
            <Select value={iface.bond?.mode || 'balance-rr'} onValueChange={(v) => setBond({ mode: v })}>
              <SelectTrigger className="font-mono"><SelectValue /></SelectTrigger>
              <SelectContent>
                {BOND_MODES.map((m) => <SelectItem key={m} value={m} className="font-mono">{m}</SelectItem>)}
              </SelectContent>
            </Select>
          </ComboRow>
          <EntryRow title="MII monitor (ms, 0 = off)" inputMode="numeric" value={iface.bond?.miimon || ''} onChange={(e) => setBond({ miimon: Number(e.target.value) || undefined })} placeholder="100" className="font-mono" />
          {(iface.bond?.mode === 'balance-xor' || iface.bond?.mode === '802.3ad') && (
            <ComboRow title="Transmit hash policy">
              <Select value={iface.bond?.xmit_hash_policy || 'layer2'} onValueChange={(v) => setBond({ xmit_hash_policy: v })}>
                <SelectTrigger className="font-mono"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {BOND_HASH_POLICIES.map((p) => <SelectItem key={p} value={p} className="font-mono">{p}</SelectItem>)}
                </SelectContent>
              </Select>
            </ComboRow>
          )}
          {iface.bond?.mode === '802.3ad' && (
            <ComboRow title="LACP rate">
              <Select value={iface.bond?.lacp_rate || 'slow'} onValueChange={(v) => setBond({ lacp_rate: v })}>
                <SelectTrigger className="font-mono"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {BOND_LACP_RATES.map((r) => <SelectItem key={r} value={r} className="font-mono">{r}</SelectItem>)}
                </SelectContent>
              </Select>
            </ComboRow>
          )}
          {iface.bond?.mode === '802.3ad' && (
            <EntryRow title="Minimum links" inputMode="numeric" value={iface.bond?.min_links || ''} onChange={(e) => setBond({ min_links: Number(e.target.value) || undefined })} placeholder="0" className="font-mono" />
          )}
          <EntryRow title="Up delay (ms)" inputMode="numeric" value={iface.bond?.updelay || ''} onChange={(e) => setBond({ updelay: Number(e.target.value) || undefined })} placeholder="0" className="font-mono" />
          <EntryRow title="Down delay (ms)" inputMode="numeric" value={iface.bond?.downdelay || ''} onChange={(e) => setBond({ downdelay: Number(e.target.value) || undefined })} placeholder="0" className="font-mono" />
          {(iface.bond?.mode === 'active-backup' || iface.bond?.mode === 'balance-tlb' || iface.bond?.mode === 'balance-alb') && (
            <EntryRow title="Primary member" value={iface.bond?.primary ?? ''} onChange={(e) => setBond({ primary: e.target.value || undefined })} placeholder="eth0" className="font-mono" />
          )}
        </PreferencesGroup>
      )}

      {isVlan && (
        <PreferencesGroup title="VLAN" description="Tag traffic on the parent interface with a VLAN id.">
          <EntryRow
            title="VLAN ID"
            inputMode="numeric"
            value={iface.vlan?.id || ''}
            onChange={(e) => onChange({ ...iface, vlan: { id: Number(e.target.value) || 0 } })}
            placeholder="100"
            className="font-mono"
            error={checkVLANId(String(iface.vlan?.id || ''))}
          />
        </PreferencesGroup>
      )}

      <PreferencesGroup title="Addresses">
        <div className="px-4 py-3">
          <TagInput values={iface.addresses ?? []} onChange={(v) => set('addresses', v)} placeholder="192.168.1.1/24, dhcp, slaac" mono validate={checkInterfaceAddress} />
        </div>
      </PreferencesGroup>

      {!isTunnel && (
      <PreferencesGroup title="DHCP options">
        <div className="px-4 py-3">
          <TagInput values={iface.dhcp_options ?? []} onChange={(v) => set('dhcp_options', v)} placeholder="option:dns-server,8.8.8.8" />
        </div>
      </PreferencesGroup>
      )}

      {!isDummy && !isTunnel && !isBond && !isVlan && (
      <div className="rounded-md bg-muted/30 px-3 py-2">
        <div className="flex items-center gap-2 py-2">
          <button type="button" onClick={() => setBridgeOpen(!bridgeOpen)} className="flex items-center gap-1 text-sm font-semibold text-foreground hover:text-primary">
            {bridgeOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            Bridge
          </button>
          <Button
            type="button"
            variant={hasBridge ? 'destructive' : 'outline'}
            size="sm"
            className="ml-auto h-7 text-xs"
            onClick={toggleBridge}
          >
            {hasBridge ? 'Remove Bridge' : 'Add Bridge'}
          </Button>
        </div>
        {bridgeOpen && !hasBridge && (
          <p className="text-sm text-muted-foreground italic py-1">No bridge configured.</p>
        )}
        {bridgeOpen && hasBridge && (
          <div className="mt-2 space-y-3">
            <div className="space-y-1.5">
              <Label className="text-xs">Members</Label>
              <TagInput values={iface.bridge!.members} onChange={(v) => set('bridge', { ...iface.bridge!, members: v })} placeholder="eth0" mono />
            </div>
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm">Enable STP (Spanning Tree Protocol)</span>
              <Switch checked={iface.bridge!.stp} onCheckedChange={(v) => set('bridge', { ...iface.bridge!, stp: v })} />
            </div>
          </div>
        )}
      </div>
      )}

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add Interface</Button>}
      </div>
    </div>
  )
}

interface IfaceCardProps {
  name: string
  iface: Iface
  nic?: SystemNic
  hasVrrp?: boolean
  onEdit: () => void
  onDelete: () => void
}

function HwField({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="min-w-0">
      <div className="text-[10px] uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className={`truncate ${mono ? 'font-mono' : ''}`}>{value}</div>
    </div>
  )
}

function IfaceCard({ name, iface, nic, hasVrrp, onEdit, onDelete }: IfaceCardProps) {
  const isVlan = iface.type === 'vlan'
  const addresses = iface.addresses ?? []
  const staticAddrs = addresses.filter((a) => a.includes('/'))
  const modeAddrs = addresses.filter((a) => !a.includes('/'))
  const dynamicAddrs = (nic?.addrs ?? []).filter((a) => !staticAddrs.includes(a) && !a.toLowerCase().startsWith('fe80'))
  const hasModePills = modeAddrs.length > 0 || isVlan || !!iface.bridge || !!iface.bond || hasVrrp
  const up = nic ? nic.operstate === 'up' || !!nic.carrier : undefined
  const model = nic?.pci_vendor && nic?.pci_device ? `${nic.pci_vendor}:${nic.pci_device}` : ''
  const speed = nic?.speed ? `${nic.speed} Mbps${nic.duplex ? ` ${nic.duplex}` : ''}` : ''

  return (
    <div onClick={onEdit} className="flex cursor-pointer flex-col gap-3 rounded-lg bg-card p-4 shadow-[var(--card-shadow)] transition-colors hover:bg-accent/40">
      <div className="flex items-center gap-2">
        {up !== undefined && (
          <span className={`h-2 w-2 shrink-0 rounded-full ${up ? 'bg-success' : 'bg-muted-foreground'}`} title={up ? 'link up' : 'link down'} />
        )}
        <span className="truncate font-mono text-sm font-semibold">{name}</span>
        {nic && nic.name !== name && (
          <span className="shrink-0 font-mono text-xs text-muted-foreground" title="kernel device">→ {nic.name}</span>
        )}
        <div className="ml-auto flex shrink-0 items-center gap-1">
          <Badge variant="secondary" className="font-mono text-xs">{iface.type || 'physical'}</Badge>
          {iface.vrf && <Badge variant="secondary" className="font-mono text-xs">vrf:{iface.vrf}</Badge>}
          <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive"
            onClick={(e) => { e.stopPropagation(); onDelete() }}>
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <div className="flex flex-wrap gap-1">
          {staticAddrs.length === 0 && dynamicAddrs.length === 0
            ? <span className="text-xs text-muted-foreground">no addresses</span>
            : (<>
                {staticAddrs.map((addr) => <Badge key={addr} variant="outline" className="font-mono text-xs">{addr}</Badge>)}
                {dynamicAddrs.map((addr) => <Badge key={addr} variant="info" className="font-mono text-xs" title="obtained via DHCP/SLAAC">{addr}</Badge>)}
              </>)}
          {iface.type === 'tunnel' && (iface.local || iface.remote) && (
            <Badge variant="outline" className="font-mono text-xs">{iface.local || '-'} → {iface.remote || '-'}</Badge>
          )}
        </div>

        {hasModePills && (
          <div className="flex flex-wrap gap-1">
            {modeAddrs.map((m) => <Badge key={m} variant="secondary" className="font-mono text-xs">{m}</Badge>)}
            {isVlan && <Badge variant="outline" className="text-xs">vlan {iface.vlan?.id ?? 0}{iface.select ? ` · ${iface.select}` : ''}</Badge>}
            {iface.bridge && <Badge variant="outline" className="text-xs">bridge</Badge>}
            {iface.bond && <Badge variant="outline" className="text-xs">{iface.bond.mode || 'balance-rr'} · {iface.bond.members?.length ?? 0}</Badge>}
            {hasVrrp && <Badge variant="outline" className="text-xs">VRRP</Badge>}
          </div>
        )}
      </div>

      {nic?.physical && (
        <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 rounded-md bg-muted/30 px-3 py-2 text-xs sm:grid-cols-3">
          <HwField label="Driver" value={nic.driver || '-'} />
          <HwField label="Speed" value={speed || '-'} />
          <HwField label="Model" value={model || '-'} mono />
          <HwField label="MAC" value={nic.mac || '-'} mono />
          <HwField label="Link" value={nic.carrier ? 'up' : 'down'} />
        </div>
      )}
    </div>
  )
}

const NEW_KEY = '__new__'

function IfaceRow({ name, iface, nic, onEdit, onDelete }: IfaceCardProps) {
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

function matchNic(nics: SystemNic[], iface: Iface, name: string): SystemNic | undefined {
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

export default function Interfaces() {
  const { data, isLoading, reload } = useFetch<IfaceMap>(() => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<IfaceMap>)
  const { data: tunData, reload: reloadTun } = useFetch<TunnelMap>(() => api.apiConfigSectionGet({ section: 'tunnels' }) as Promise<TunnelMap>)
  const { data: vrfsData } = useFetch<Record<string, { table: number }>>(
    () => api.apiConfigSectionGet({ section: 'vrfs' }) as Promise<Record<string, { table: number }>>
  )
  const { data: nics } = useFetch<SystemNic[]>(() => api.apiSystemNicsGet())
  const { data: haData } = useFetch<{ vrrp?: { interface: string }[] }>(
    () => api.apiConfigSectionGet({ section: 'ha' }) as Promise<{ vrrp?: { interface: string }[] }>
  )
  const vrrpIfaces = new Set((haData?.vrrp ?? []).map((v) => v.interface))
  const nicFor = (iface: Iface, name: string) => matchNic(nics ?? [], iface, name)
  const vrfNames = Object.keys(vrfsData ?? {})
  const [entries, setEntries] = useState<IfaceMap | null>(null)
  const [newName, setNewName] = useState('')
  const [view, setView] = useState<'cards' | 'list'>(() => (localStorage.getItem('interfaces-view') === 'list' ? 'list' : 'cards'))
  const [openSheet, setOpenSheet] = useState<string | null>(null)
  const [formDraft, setFormDraft] = useState<Iface | null>(null)
  const [pendingName, setPendingName] = useState<string | null>(null)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('interfaces')
  const tun = usePageSave('tunnels')

  useDataRefresh(() => { setEntries(null); reset(); tun.reset() })

  const merged: IfaceMap = {
    ...((data as IfaceMap | null) ?? {}),
    ...Object.fromEntries(Object.entries(tunData ?? {}).map(([n, t]) => [n, tunnelToIface(t)])),
  }
  const current: IfaceMap = entries ?? merged

  const saveAll = () => {
    const ifaceMap: IfaceMap = {}
    const tunMap: TunnelMap = {}
    for (const [name, iface] of Object.entries(current)) {
      if (iface.type === 'tunnel') {
        tunMap[name] = ifaceToTunnel(iface)
      } else {
        ifaceMap[name] = iface
      }
    }
    save(ifaceMap)
    tun.save(tunMap)
  }

  const resetAll = () => { setEntries(null); reset(); tun.reset() }
  const reloadAll = () => { setEntries(null); reload(); reloadTun() }

  const update = (name: string, iface: Iface) => {
    setEntries({ ...current, [name]: iface })
    markDirty()
  }

  const deleteEntry = (name: string) => {
    const next = { ...current }
    delete next[name]
    setEntries(next)
    if (openSheet === name) setOpenSheet(null)
    markDirty()
  }

  const handleAddNew = () => { setFormDraft(emptyIface()); setNewName(''); setOpenSheet(NEW_KEY) }

  const handleCommitNew = () => {
    const n = newName.trim()
    if (!n) { toast.error('Please enter an interface name'); return }
    if (current[n]) { toast.error(`Interface "${n}" already exists`); return }
    setEntries({ ...current, [n]: formDraft ?? emptyIface() })
    markDirty()
    setOpenSheet(null)
    setFormDraft(null)
  }

  const handleCloseSheet = () => {
    if (openSheet && openSheet !== NEW_KEY && pendingName !== null && pendingName.trim() !== openSheet) {
      const n = pendingName.trim()
      if (n && !current[n]) {
        const next: IfaceMap = {}
        for (const [k, v] of Object.entries(current)) {
          next[k === openSheet ? n : k] = v
        }
        setEntries(next)
        markDirty()
      }
    }
    setOpenSheet(null)
    setFormDraft(null)
    setPendingName(null)
  }

  const names = Object.keys(current)
  const parentNames = names.filter((n) => current[n].type !== 'tunnel' && current[n].type !== 'vlan')
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(names, 12)

  const setViewMode = (v: 'cards' | 'list') => { localStorage.setItem('interfaces-view', v); setView(v) }

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      <PageHeader title="Interfaces" description="Manage network interface configuration" action={
        <div className="flex items-center gap-2">
          <SaveButton isDirty={isDirty} saving={saving || tun.saving} onClick={saveAll} onCancel={resetAll} />
          <ReloadButton onClick={reloadAll} />
        </div>
      } />

      {names.length === 0 ? (
        <EmptyState
          className="max-w-2xl mx-auto"
          icon={<Network />}
          title="No interfaces"
          message="Define an interface to assign addresses, VLANs and VRRP."
          action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add interface</Button>}
        />
      ) : (
        <div className="space-y-3">
          <div className="flex items-center justify-between px-1">
            <div className="text-sm font-medium">Interfaces</div>
            <div className="flex items-center gap-2">
              <Segmented
                value={view}
                onChange={setViewMode}
                className="text-xs"
                options={[{ value: 'cards', label: 'Cards' }, { value: 'list', label: 'List' }]}
              />
              <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
                <Plus className="h-4 w-4" />Add
              </Button>
            </div>
          </div>
          {view === 'cards' ? (
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              {pageItems.map((name) => (
                <IfaceCard
                  key={name}
                  name={name}
                  iface={current[name]}
                  nic={nicFor(current[name], name)}
                  hasVrrp={vrrpIfaces.has(name)}
                  onEdit={() => { setOpenSheet(name); setPendingName(name) }}
                  onDelete={() => deleteEntry(name)}
                />
              ))}
            </div>
          ) : (
            <div className="rounded-xl bg-card shadow-[var(--card-shadow)] overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Device</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Addresses</TableHead>
                    <TableHead>VRF</TableHead>
                    <TableHead className="w-[1%]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pageItems.map((name) => (
                    <IfaceRow
                      key={name}
                      name={name}
                      iface={current[name]}
                      nic={nicFor(current[name], name)}
                      onEdit={() => { setOpenSheet(name); setPendingName(name) }}
                      onDelete={() => deleteEntry(name)}
                    />
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      )}

      <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="interfaces" />

      <Sheet
        open={!!openSheet}
        onClose={handleCloseSheet}
        title={openSheet === NEW_KEY ? 'New Interface' : (pendingName ?? openSheet ?? '')}
        className="max-w-2xl"
      >
        {openSheet && (openSheet === NEW_KEY ? formDraft : current[openSheet]) && (
          <IfaceForm
            name={openSheet === NEW_KEY ? newName : (pendingName ?? openSheet ?? '')}
            iface={openSheet === NEW_KEY ? formDraft! : current[openSheet]}
            vrfNames={vrfNames}
            parents={parentNames}
            nics={nics ?? []}
            onChange={openSheet === NEW_KEY ? setFormDraft : (u) => update(openSheet, u)}
            onNameChange={openSheet === NEW_KEY ? setNewName : setPendingName}
            onAdd={openSheet === NEW_KEY ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
          />
        )}
      </Sheet>
    </div>
  )
}
