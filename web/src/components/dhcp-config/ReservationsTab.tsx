import { plural } from '@/lib/utils'
import { AutoTextarea } from '@/components/ui/AutoTextarea'
import { DhcpConfigData, KeaReservation, ReservationRef, SubnetRef, familyKey } from '@/components/dhcp-config/shared'
import { api } from '@/lib/client'
import { Button, cn, EmptyState, Input, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { Plus, Search, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

type ReservationsTabShape = {
  cfg: DhcpConfigData
  onChange: (next: DhcpConfigData) => void
}

export function ReservationsTab({ cfg, onChange }: ReservationsTabShape) {
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
    <div className="space-y-3">
        <div className="flex flex-row items-center justify-between">
          <span className="text-sm font-semibold">{plural(rows.length, 'Reservation')}</span>
          <Button variant="outline" size="sm" className="gap-1.5" onClick={add} disabled={subnets.length === 0}>
            <Plus className="h-3.5 w-3.5" />Add Reservation
          </Button>
        </div>
        {rows.length > 0 && (
          <div className="flex flex-wrap items-center gap-2">
            <Select value={filter} onValueChange={setFilter}>
              <SelectTrigger aria-label="Subnet Filter" className="h-8 w-full text-xs sm:w-48">
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
        {subnets.length === 0 && <EmptyState className="py-10" title="No Subnets" />}
        {subnets.length > 0 && rows.length === 0 && (
          <EmptyState className="py-10"
            title="No Reservations"
          />
        )}
        {rows.length > 0 && visible.length === 0 && (
          <p className="text-sm text-muted-foreground">No Results</p>
        )}
        {visible.length > 0 && (
          <div className="hidden grid-cols-[minmax(8rem,0.9fr)_1.3fr_1.4fr_1.4fr_32px] gap-2 px-0.5 text-xs text-muted-foreground sm:grid">
            <span>Subnet</span><span>Hostname</span><span>MAC / DUID</span><span>Address</span><span />
          </div>
        )}
        {visible.map((row) => (
          <div
            key={`${refKey(row.v6, row.si)}-${row.ri}`}
            className="grid grid-cols-[1fr_auto] items-start gap-2 rounded-xl bg-card p-3 [box-shadow:var(--card-shadow)] sm:grid-cols-[minmax(8rem,0.9fr)_1.3fr_1.4fr_1.4fr_32px] sm:rounded-none sm:bg-transparent sm:p-0 sm:[box-shadow:none]"
          >
            <ReservationField label="Subnet" className="order-3 col-span-2 sm:order-none sm:col-span-1">
              <Select value={refKey(row.v6, row.si)} onValueChange={(v) => { const t = parseRef(v); if (t) move(row, t) }}>
                <SelectTrigger aria-label="Subnet" className="h-8 text-xs font-mono">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {subnets.map((s) => (
                    <SelectItem key={refKey(s.v6, s.si)} value={refKey(s.v6, s.si)} className="font-mono text-xs">{s.cidr}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </ReservationField>
            <AutoTextarea
              aria-label="Hostname"
              value={row.res.hostname ?? ''}
              onChange={(v) => patch(row, { hostname: v || undefined })}
              placeholder="Hostname"
              className="order-1 break-words text-xs max-sm:font-medium sm:order-none"
            />
            <ReservationField label={row.v6 ? 'DUID' : 'MAC Address'} className="order-4 col-span-2 sm:order-none sm:col-span-1">
              <AutoTextarea
                aria-label={row.v6 ? 'DUID' : 'MAC Address'}
                value={(row.v6 ? row.res.duid : row.res.hw_address) ?? ''}
                onChange={(v) => patch(row, row.v6 ? { duid: v || undefined } : { hw_address: v || undefined })}
                placeholder={row.v6 ? 'DUID' : 'aa:bb:cc:dd:ee:ff'}
                className="text-xs font-mono"
              />
            </ReservationField>
            <ReservationField label="Address" className="order-5 col-span-2 sm:order-none sm:col-span-1">
              <div className="relative">
                <AutoTextarea
                  aria-label="Address"
                  value={row.res.ip_address ?? ''}
                  onChange={(v) => patch(row, { ip_address: v || undefined })}
                  placeholder={row.v6 ? '2001:db8::5' : '10.0.0.5'}
                  className="text-xs font-mono pr-14"
                />
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => autoIP(row)}
                  title="Suggest Free Address"
                  className="absolute right-1 top-1 h-6 px-1.5"
                >
                  Auto
                </Button>
              </div>
            </ReservationField>
            <Button variant="ghost" size="icon" className="order-2 h-8 w-8 text-muted-foreground hover:text-destructive sm:order-none" onClick={() => remove(row)} aria-label="Remove Reservation" title="Remove Reservation">
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        ))}
    </div>
  )
}

interface ReservationFieldProps {
  label: string
  className?: string
  children: React.ReactNode
}

function ReservationField({ label, className, children }: ReservationFieldProps) {
  return (
    <div className={cn('space-y-1 sm:space-y-0', className)}>
      <span className="block text-xs text-muted-foreground sm:hidden">{label}</span>
      {children}
    </div>
  )
}
