import { useEffect, useState } from 'react'
import { Plus, Trash2, Search } from 'lucide-react'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { useTabState } from 'cheval-ui'
import { SaveButton } from 'cheval-ui'
import { Spinner } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { Input } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Switch } from 'cheval-ui'
import { Tabs } from 'cheval-ui'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { Card, CardHeader, CardTitle, CardContent } from 'cheval-ui'
import { TagInput } from 'cheval-ui'
import { VarPickerInput } from '@/components/VarPickerInput'
import { AccordionList } from '@/components/ui/AccordionList'
import { cidrNetwork } from '@/lib/cidr'

const ANY_IFACE = '*'

interface KeaReservation {
  hostname?: string
  hw_address?: string
  duid?: string
  ip_address?: string
}

interface KeaSubnet {
  subnet: string
  interface?: string
  pools?: string[]
  exclusions?: string[]
  gateway?: string
  dns?: string[]
  reservations?: KeaReservation[]
}

interface DhcpDDNS {
  enabled?: boolean
  domain?: string
  ttl?: number
  reverse?: boolean
}

interface DhcpConfigData {
  enabled?: boolean
  control_agent?: { url?: string; user?: string; password?: string }
  subnets4?: KeaSubnet[]
  subnets6?: KeaSubnet[]
  ddns?: DhcpDDNS
}

function InterfaceSelect({ value, options, onChange }: {
  value?: string
  options: string[]
  onChange: (v: string | undefined) => void
}) {
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

function splitPool(pool: string): { start: string; end: string } {
  const i = pool.indexOf('-')
  if (i < 0) return { start: pool.trim(), end: '' }
  return { start: pool.slice(0, i).trim(), end: pool.slice(i + 1).trim() }
}

function PoolRows({ v6, pools, onChange }: {
  v6: boolean
  pools: string[]
  onChange: (p: string[]) => void
}) {
  const rows = pools.map(splitPool)
  const write = (next: { start: string; end: string }[]) =>
    onChange(next.map((r) => `${r.start.trim()}-${r.end.trim()}`))
  const upd = (i: number, patch: Partial<{ start: string; end: string }>) =>
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

function SubnetSummary({ subnet }: { subnet: KeaSubnet }) {
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

function SubnetBody({ v6, subnet, networks, ifaceNames, onChange }: {
  v6: boolean
  subnet: KeaSubnet
  networks: string[]
  ifaceNames: string[]
  onChange: (s: KeaSubnet) => void
}) {
  const set = <K extends keyof KeaSubnet>(k: K, val: KeaSubnet[K]) => onChange({ ...subnet, [k]: val })

  return (
    <>
      <div className="space-y-2">
        <div className="space-y-1.5">
          <Label className="text-[11px] text-muted-foreground">Network (CIDR)</Label>
          <VarPickerInput
            value={subnet.subnet}
            onChange={(v) => set('subnet', v)}
            vars={networks}
            prefix=""
            label="Configured networks"
            placeholder={v6 ? '2001:db8::/64' : '10.0.0.0/24'}
            mono
            wrapperClassName="flex-1"
            className="h-8 text-xs"
          />
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-1.5">
            <Label className="text-[11px] text-muted-foreground">Interface</Label>
            <InterfaceSelect value={subnet.interface} options={ifaceNames} onChange={(v) => set('interface', v)} />
          </div>
          {!v6 && (
            <div className="space-y-1.5">
              <Label className="text-[11px] text-muted-foreground">Gateway</Label>
              <Input value={subnet.gateway ?? ''} onChange={(e) => set('gateway', e.target.value || undefined)} placeholder="10.0.0.1" className="h-8 text-xs font-mono" />
            </div>
          )}
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <PoolRows v6={v6} pools={subnet.pools ?? []} onChange={(v) => set('pools', v.length ? v : undefined)} />
        <div className="space-y-1.5">
          <Label className="text-[11px] text-muted-foreground">DNS servers</Label>
          <TagInput values={subnet.dns ?? []} onChange={(v) => set('dns', v.length ? v : undefined)} placeholder={v6 ? '2001:db8::53' : '1.1.1.1'} mono />
        </div>
        <div className="space-y-1.5 md:col-span-2">
          <Label className="text-[11px] text-muted-foreground">Reserved for manual use (excluded from auto-suggested addresses)</Label>
          <TagInput values={subnet.exclusions ?? []} onChange={(v) => set('exclusions', v.length ? v : undefined)} placeholder={v6 ? '2001:db8::1-2001:db8::ff' : '10.0.0.1-10.0.0.20 or 10.0.0.5'} mono />
        </div>
      </div>
    </>
  )
}

function SubnetList({ v6, subnets, networks, ifaceNames, onChange }: {
  v6: boolean
  subnets: KeaSubnet[]
  networks: string[]
  ifaceNames: string[]
  onChange: (s: KeaSubnet[]) => void
}) {
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

interface SubnetRef {
  cidr: string
  v6: boolean
  si: number
}

interface ReservationRef extends SubnetRef {
  ri: number
  res: KeaReservation
}

function familyKey(v6: boolean): 'subnets6' | 'subnets4' {
  return v6 ? 'subnets6' : 'subnets4'
}

function ReservationsTab({ cfg, onChange }: {
  cfg: DhcpConfigData
  onChange: (next: DhcpConfigData) => void
}) {
  const [filter, setFilter] = useState<string>('all')
  const [query, setQuery] = useState('')

  const subnets: SubnetRef[] = [
    ...(cfg.subnets4 ?? []).map((s, si) => ({ cidr: s.subnet, v6: false, si })),
    ...(cfg.subnets6 ?? []).map((s, si) => ({ cidr: s.subnet, v6: true, si })),
  ].filter((s) => s.cidr)

  const rows: ReservationRef[] = [
    ...(cfg.subnets4 ?? []).flatMap((s, si) => (s.reservations ?? []).map((res, ri) => ({ cidr: s.subnet, v6: false, si, ri, res }))),
    ...(cfg.subnets6 ?? []).flatMap((s, si) => (s.reservations ?? []).map((res, ri) => ({ cidr: s.subnet, v6: true, si, ri, res }))),
  ]

  const withReservations = (data: DhcpConfigData, v6: boolean, si: number, next: KeaReservation[] | undefined) => {
    const key = familyKey(v6)
    const arr = (data[key] ?? []).map((s, j) => (j === si ? { ...s, reservations: next && next.length ? next : undefined } : s))
    return { ...data, [key]: arr }
  }

  const patch = (row: ReservationRef, p: Partial<KeaReservation>) => {
    const list = (cfg[familyKey(row.v6)]?.[row.si].reservations ?? []).map((r, j) => (j === row.ri ? { ...r, ...p } : r))
    onChange(withReservations(cfg, row.v6, row.si, list))
  }

  const remove = (row: ReservationRef) => {
    const list = (cfg[familyKey(row.v6)]?.[row.si].reservations ?? []).filter((_, j) => j !== row.ri)
    onChange(withReservations(cfg, row.v6, row.si, list))
  }

  const append = (target: SubnetRef, res: KeaReservation) => {
    const list = [...(cfg[familyKey(target.v6)]?.[target.si].reservations ?? []), res]
    onChange(withReservations(cfg, target.v6, target.si, list))
  }

  const move = (row: ReservationRef, target: SubnetRef) => {
    if (target.v6 === row.v6 && target.si === row.si) return
    const carried: KeaReservation = { hostname: row.res.hostname, ip_address: row.res.ip_address }
    if (target.v6) carried.duid = row.res.duid
    else carried.hw_address = row.res.hw_address
    const cleared = withReservations(cfg, row.v6, row.si, (cfg[familyKey(row.v6)]?.[row.si].reservations ?? []).filter((_, j) => j !== row.ri))
    const list = [...(cleared[familyKey(target.v6)]?.[target.si].reservations ?? []), carried]
    onChange(withReservations(cleared, target.v6, target.si, list))
  }

  const add = () => {
    const target = filter !== 'all' ? parseRef(filter) : subnets[0]
    if (target) append(target, {})
  }

  const autoIP = async (row: ReservationRef) => {
    if (!row.cidr) { toast.error('Set the subnet first'); return }
    try {
      const { status } = await api.apiDhcpFreeIpGet({ target: row.cidr })
      patch(row, { ip_address: status })
    } catch (e) {
      toast.error((e as Error).message)
    }
  }

  const refKey = (v6: boolean, si: number) => `${v6 ? '6' : '4'}:${si}`
  const parseRef = (v: string): SubnetRef | undefined => subnets.find((s) => refKey(s.v6, s.si) === v)

  const q = query.trim().toLowerCase()
  const matches = (row: ReservationRef) => {
    if (filter !== 'all' && refKey(row.v6, row.si) !== filter) return false
    if (!q) return true
    const hay = [row.res.hostname, row.res.ip_address, row.res.hw_address, row.res.duid, row.cidr]
      .filter(Boolean).join(' ').toLowerCase()
    return hay.includes(q)
  }
  const visible = rows.filter(matches)

  return (
    <Card>
      <CardHeader className="space-y-3">
        <div className="flex flex-row items-center justify-between">
          <CardTitle className="text-sm">Reservations ({rows.length})</CardTitle>
          <Button variant="outline" size="sm" className="gap-1.5" onClick={add} disabled={subnets.length === 0}>
            <Plus className="h-3.5 w-3.5" />Add reservation
          </Button>
        </div>
        {rows.length > 0 && (
          <div className="flex flex-wrap items-center gap-2">
            <Select value={filter} onValueChange={setFilter}>
              <SelectTrigger className="h-8 w-48 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All subnets</SelectItem>
                {subnets.map((s) => (
                  <SelectItem key={refKey(s.v6, s.si)} value={refKey(s.v6, s.si)} className="font-mono text-xs">{s.cidr}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <div className="relative min-w-[14rem] flex-1">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search hostname, IP, MAC, DUID" className="h-8 pl-8 text-xs" />
            </div>
          </div>
        )}
      </CardHeader>
      <CardContent className="space-y-2">
        {subnets.length === 0 && <p className="text-sm text-muted-foreground italic">Add a subnet first, then pin hosts to fixed addresses here.</p>}
        {subnets.length > 0 && rows.length === 0 && (
          <EmptyState className="py-10"
            title="No reservations"
            message="Pin hosts to fixed addresses here, or from a live lease under Monitor › DHCP."
          />
        )}
        {rows.length > 0 && visible.length === 0 && (
          <p className="text-sm text-muted-foreground italic">No reservations match.</p>
        )}
        {visible.length > 0 && (
          <div className="grid grid-cols-[minmax(9rem,1.3fr)_1fr_1.4fr_1.2fr_28px] items-center gap-2 px-0.5 text-[11px] text-muted-foreground">
            <span>Subnet</span><span>Hostname</span><span>MAC / DUID</span><span>Address</span><span />
          </div>
        )}
        {visible.map((row) => (
          <div key={`${refKey(row.v6, row.si)}-${row.ri}`} className="grid grid-cols-[minmax(9rem,1.3fr)_1fr_1.4fr_1.2fr_28px] items-center gap-2">
            <Select value={refKey(row.v6, row.si)} onValueChange={(v) => { const t = parseRef(v); if (t) move(row, t) }}>
              <SelectTrigger className="h-8 text-xs font-mono">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {subnets.map((s) => (
                  <SelectItem key={refKey(s.v6, s.si)} value={refKey(s.v6, s.si)} className="font-mono text-xs">{s.cidr}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input value={row.res.hostname ?? ''} onChange={(e) => patch(row, { hostname: e.target.value || undefined })} placeholder="hostname" className="h-8 text-xs" />
            <Input
              value={(row.v6 ? row.res.duid : row.res.hw_address) ?? ''}
              onChange={(e) => patch(row, row.v6 ? { duid: e.target.value || undefined } : { hw_address: e.target.value || undefined })}
              placeholder={row.v6 ? 'DUID' : 'aa:bb:cc:dd:ee:ff'}
              className="h-8 text-xs font-mono"
            />
            <div className="relative">
              <Input value={row.res.ip_address ?? ''} onChange={(e) => patch(row, { ip_address: e.target.value || undefined })} placeholder={row.v6 ? '2001:db8::5' : '10.0.0.5'} className="h-8 text-xs font-mono pr-12" />
              <button
                type="button"
                onClick={() => autoIP(row)}
                title="Suggest a free out-of-pool address"
                className="absolute right-1 top-1/2 -translate-y-1/2 rounded px-1.5 py-0.5 text-[10px] text-muted-foreground hover:bg-accent hover:text-foreground"
              >
                Auto
              </button>
            </div>
            <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => remove(row)}>
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}

function DdnsTab({ cfg, onChange }: {
  cfg: DhcpConfigData
  onChange: (next: DhcpConfigData) => void
}) {
  const d = cfg.ddns ?? {}
  const set = <K extends keyof DhcpDDNS>(k: K, v: DhcpDDNS[K]) => onChange({ ...cfg, ddns: { ...d, [k]: v } })

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Dynamic DNS</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <label className="flex items-center gap-3">
          <Switch checked={!!d.enabled} onCheckedChange={(v) => set('enabled', v || undefined)} />
          <div>
            <div className="text-sm font-medium">Register leases in DNS</div>
            <div className="text-xs text-muted-foreground">Kea updates the local BIND with A/AAAA and PTR records as leases come and go. Requires the DNS server to be enabled.</div>
          </div>
        </label>

        {d.enabled && (
          <div className="space-y-4">
            <div className="space-y-1.5">
              <Label className="text-[11px] text-muted-foreground">Forward domain</Label>
              <Input value={d.domain ?? ''} onChange={(e) => set('domain', e.target.value || undefined)} placeholder="lan.example.com" className="h-8 text-xs font-mono" />
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-1.5">
                <Label className="text-[11px] text-muted-foreground">Record TTL (seconds)</Label>
                <Input type="number" value={d.ttl ?? ''} onChange={(e) => set('ttl', e.target.value ? Number(e.target.value) : undefined)} placeholder="3600" className="h-8 text-xs" />
              </div>
              <label className="flex items-center gap-3 pt-5">
                <Switch checked={d.reverse !== false} onCheckedChange={(v) => set('reverse', v ? undefined : false)} />
                <div className="text-sm">Update reverse (PTR) zones</div>
              </label>
            </div>
            <p className="text-xs text-muted-foreground">The forward domain is appended to client hostnames. The closest existing parent zone is reused: ans.mvinc.fr registers hostname.ans.mvinc.fr in mvinc.fr when that zone exists. Existing reverse zones are also reused. Missing zones are created automatically. The TSIG key that authenticates updates is generated on first apply and stored in the config.</p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

interface IfaceData {
  addresses?: string[]
}

type DhcpTab = 'subnets' | 'reservations' | 'ddns'

export function DhcpConfig({ onActionChange }: { onActionChange?: (a: React.ReactNode) => void }) {
  const { data, isLoading } = useFetch<DhcpConfigData | null>(() => api.apiConfigSectionGet({ section: 'dhcp' }) as Promise<DhcpConfigData | null>)
  const { data: ifaces } = useFetch<Record<string, IfaceData>>(() => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<Record<string, IfaceData>>)
  const [cfg, setCfg] = useState<DhcpConfigData>({})
  const [initialized, setInitialized] = useState(false)
  const [tab, setTab] = useTabState<DhcpTab>('dhcp.config', 'subnets')
  const { isDirty, markDirty, save, saving } = usePageSave('dhcp')

  const ifaceNames = Object.keys(ifaces ?? {})

  const networks = Array.from(new Set(
    Object.values(ifaces ?? {}).flatMap((i) => i?.addresses ?? [])
      .map(cidrNetwork)
      .filter((n): n is string => n !== null),
  ))

  useEffect(() => {
    if (!isLoading && !initialized) {
      setCfg(data ?? {})
      setInitialized(true)
    }
  }, [isLoading, initialized, data])

  const update = (next: DhcpConfigData) => { setCfg(next); markDirty() }
  const set = <K extends keyof DhcpConfigData>(k: K, v: DhcpConfigData[K]) => update({ ...cfg, [k]: v })

  useEffect(() => {
    onActionChange?.(
      <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(cfg)} onCancel={() => setCfg(data ?? {})} />
    )
    return () => onActionChange?.(null)
  }, [isDirty, saving, cfg, data, onActionChange, save])

  if (isLoading) return <Spinner />

  const remote = !!cfg.control_agent?.url

  return (
    <div className="space-y-5">
      <label className="flex items-center gap-3">
        <Switch checked={!!cfg.enabled} onCheckedChange={(v) => set('enabled', v || undefined)} />
        <div>
          <div className="text-sm font-medium">Enable DHCP server</div>
          <div className="text-xs text-muted-foreground">Render and run a local Kea DHCP server from this config.</div>
        </div>
      </label>

      {remote && (
        <p className="rounded-md bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          A remote control agent ({cfg.control_agent?.url}) is configured, so no local Kea is rendered. The subnets below are ignored for a remote server.
        </p>
      )}

      <Tabs
        active={tab}
        onChange={setTab}
        tabs={[
          { key: 'subnets', label: 'Subnets' },
          { key: 'reservations', label: 'Reservations' },
          { key: 'ddns', label: 'Dynamic DNS' },
        ]}
      />

      {tab === 'subnets' && (
        <div className="space-y-5">
          <SubnetList v6={false} subnets={cfg.subnets4 ?? []} networks={networks} ifaceNames={ifaceNames} onChange={(s) => set('subnets4', s.length ? s : undefined)} />
          <SubnetList v6={true} subnets={cfg.subnets6 ?? []} networks={networks} ifaceNames={ifaceNames} onChange={(s) => set('subnets6', s.length ? s : undefined)} />
        </div>
      )}

      {tab === 'reservations' && <ReservationsTab cfg={cfg} onChange={update} />}

      {tab === 'ddns' && <DdnsTab cfg={cfg} onChange={update} />}
    </div>
  )
}
