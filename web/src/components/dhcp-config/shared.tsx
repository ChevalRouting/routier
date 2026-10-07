import { SubnetBody } from '@/components/dhcp-config/SubnetBody'
import { AccordionList } from '@/components/ui/AccordionList'
import { Button, Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'

type ControlagentShape = { url?: string; user?: string; password?: string }

type InterfaceSelectShape = {
  value?: string
  options: string[]
  onChange: (v: string | undefined) => void
}

type SplitPoolShape = { start: string; end: string }

type PoolRowsShape = {
  v6: boolean
  pools: string[]
  onChange: (p: string[]) => void
}

type NextShape = { start: string; end: string }

type PatchShape = { start: string; end: string }

type SubnetSummaryShape = { subnet: KeaSubnet }

type SubnetListShape = {
  v6: boolean
  subnets: KeaSubnet[]
  networks: string[]
  ifaceNames: string[]
  onChange: (s: KeaSubnet[]) => void
}

export const ANY_IFACE = '*'

export interface KeaReservation {
  hostname?: string
  hw_address?: string
  duid?: string
  ip_address?: string
}

export interface KeaSubnet {
  subnet: string
  interface?: string
  pools?: string[]
  exclusions?: string[]
  gateway?: string
  dns?: string[]
  reservations?: KeaReservation[]
}

export interface DhcpDDNS {
  enabled?: boolean
  domain?: string
  ttl?: number
  reverse?: boolean
}

export interface DhcpConfigData {
  enabled?: boolean
  control_agent?: ControlagentShape
  subnets4?: KeaSubnet[]
  subnets6?: KeaSubnet[]
  ddns?: DhcpDDNS
}

export function InterfaceSelect({ value, options, onChange }: InterfaceSelectShape) {
  const extra = value && !options.includes(value) ? [value] : []
  return (
    <Select value={value || ANY_IFACE} onValueChange={(v) => onChange(v === ANY_IFACE ? undefined : v)}>
      <SelectTrigger className="h-8 text-xs">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ANY_IFACE}>All interfaces</SelectItem>
        {[...options, ...extra].map((o) => (
          <SelectItem key={o} value={o} className="font-mono text-xs">{o}</SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export function splitPool(pool: string): SplitPoolShape {
  const i = pool.indexOf('-')
  if (i < 0) return { start: pool.trim(), end: '' }
  return { start: pool.slice(0, i).trim(), end: pool.slice(i + 1).trim() }
}

export function PoolRows({ v6, pools, onChange }: PoolRowsShape) {
  const rows = pools.map(splitPool)
  const write = (next: NextShape[]) =>
    onChange(next.map((r) => `${r.start.trim()}-${r.end.trim()}`))
  const upd = (i: number, patch: Partial<PatchShape>) =>
    write(rows.map((r, j) => (j === i ? { ...r, ...patch } : r)))
  const add = () => write([...rows, { start: '', end: '' }])
  const remove = (i: number) => write(rows.filter((_, j) => j !== i))

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <Label className="text-[11px] text-muted-foreground">Pools (address ranges)</Label>
        <Button variant="ghost" size="sm" className="h-6 gap-1 text-xs" onClick={add}>
          <Plus className="h-3 w-3" />Add pool
        </Button>
      </div>
      {rows.map((r, i) => (
        <div key={i} className="grid grid-cols-[1fr_auto_1fr_28px] items-center gap-2">
          <Input value={r.start} onChange={(e) => upd(i, { start: e.target.value })} placeholder={v6 ? '2001:db8::100' : '10.0.0.100'} className="h-8 text-xs font-mono" />
          <span className="text-xs text-muted-foreground">to</span>
          <Input value={r.end} onChange={(e) => upd(i, { end: e.target.value })} placeholder={v6 ? '2001:db8::200' : '10.0.0.200'} className="h-8 text-xs font-mono" />
          <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => remove(i)}>
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      ))}
    </div>
  )
}

export function SubnetSummary({ subnet }: SubnetSummaryShape) {
  const pools = (subnet.pools ?? []).length
  const dns = (subnet.dns ?? []).length
  return (
    <div className="flex min-w-0 flex-1 items-center gap-3">
      <span className="w-52 shrink-0 truncate font-mono text-sm font-medium">{subnet.subnet || 'new subnet'}</span>
      <span className="w-28 shrink-0 truncate font-mono text-xs text-muted-foreground">{subnet.interface || 'all interfaces'}</span>
      <span className="shrink-0 text-xs text-muted-foreground">{pools} pool{pools === 1 ? '' : 's'}</span>
      {dns > 0 && <span className="shrink-0 text-xs text-muted-foreground">{dns} DNS</span>}
    </div>
  )
}

export function SubnetList({ v6, subnets, networks, ifaceNames, onChange }: SubnetListShape) {
  return (
    <div className="space-y-2">
      <div className="px-1 text-sm font-medium">{v6 ? 'IPv6 subnets' : 'IPv4 subnets'}</div>
      <AccordionList
        items={subnets}
        description={`Serve leases on an interface (${subnets.length}).`}
        addLabel="Add subnet"
        onAdd={() => onChange([...subnets, { subnet: '' }])}
        onRemove={(s) => onChange(subnets.filter((x) => x !== s))}
        emptyTitle={v6 ? 'No IPv6 subnets' : 'No IPv4 subnets'}
        emptyMessage="Add a subnet to serve leases on an interface."
        renderSummary={(s) => <SubnetSummary subnet={s} />}
        renderBody={(s) => (
          <SubnetBody
            v6={v6}
            subnet={s}
            networks={networks}
            ifaceNames={ifaceNames}
            onChange={(next) => onChange(subnets.map((x) => (x === s ? next : x)))}
          />
        )}
      />
    </div>
  )
}

export interface SubnetRef {
  cidr: string
  v6: boolean
  si: number
}

export interface ReservationRef extends SubnetRef {
  ri: number
  res: KeaReservation
}

export function familyKey(v6: boolean): 'subnets6' | 'subnets4' {
  return v6 ? 'subnets6' : 'subnets4'
}

export interface IfaceData {
  addresses?: string[]
}

export type DhcpTab = 'subnets' | 'reservations' | 'ddns'
