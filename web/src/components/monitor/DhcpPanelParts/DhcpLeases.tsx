import type { DhcpLeaseView as DhcpLeaseRow, KeaSubnet } from '@/api'
import { DnsServerSection, LeaseFamily, NO_ZONE, forwardZoneNames, subnetService } from '@/components/monitor/DhcpPanel'
import { api } from '@/lib/client'
import { AlertDialog, Badge, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, Input, Pagination, Segmented, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Sheet, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, usePagination, ReloadButton, PreferencesGroup, EntryRow, ComboRow, Row } from 'cheval-ui'
import { Pin, Search, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

export function DhcpLeases() {
  const [q, setQ] = useState('')
  const [rows, setRows] = useState<DhcpLeaseRow[]>([])
  const [loadError, setLoadError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const requestID = useRef(0)
  const [subnets, setSubnets] = useState<KeaSubnet[]>([])
  const [family, setFamily] = useState<LeaseFamily>('all')
  const [subnetKey, setSubnetKey] = useState('all')
  const [busy, setBusy] = useState(false)
  const [clearing, setClearing] = useState<string | null>(null)
  const [reserving, setReserving] = useState<DhcpLeaseRow | null>(null)
  const [resIP, setResIP] = useState('')
  const [resHostname, setResHostname] = useState('')
  const [zones, setZones] = useState<string[]>([])
  const [resZone, setResZone] = useState(NO_ZONE)
  const [resDnsName, setResDnsName] = useState('')

  const load = useCallback(async (query: string) => {
    const id = ++requestID.current
    setLoading(true)
    try {
      const result = await api.apiDhcpLeasesGet({ q: query || undefined })
      if (id !== requestID.current) return
      setRows(result ?? [])
      setLoadError(null)
    } catch (error) {
      if (id !== requestID.current) return
      setLoadError(error instanceof Error ? error.message : 'Unable to load DHCP leases.')
    } finally {
      if (id === requestID.current) setLoading(false)
    }
  }, [])

  useEffect(() => {
    const requests = requestID
    let stopped = false
    let timer: ReturnType<typeof setTimeout>
    const refresh = async () => {
      await load(q)
      if (!stopped) timer = setTimeout(refresh, 5000)
    }
    timer = setTimeout(refresh, 250)
    return () => {
      stopped = true
      clearTimeout(timer)
      ++requests.current
    }
  }, [q, load])

  useEffect(() => {
    api.apiDhcpSubnetsGet().then((s) => setSubnets(s ?? [])).catch(() => setSubnets([]))
  }, [])

  useEffect(() => {
    api.apiConfigSectionGet({ section: 'dns_server' })
      .then((s) => setZones(forwardZoneNames(s as DnsServerSection | null)))
      .catch(() => setZones([]))
  }, [])

  const changeFamily = (f: LeaseFamily) => {
    setFamily(f)
    if (f !== 'all' && subnetKey !== 'all' && !subnetKey.startsWith(`${f}:`)) setSubnetKey('all')
  }

  const doClear = async (ip: string) => {
    setBusy(true)
    try {
      await api.apiDhcpLeasesClearPost({ DhcpTargetRequest: { target: ip } })
      toast.success(`Cleared lease ${ip}`)
      load(q)
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const openReserve = async (l: DhcpLeaseRow) => {
    setReserving(l)
    setResIP('')
    setResHostname(l.hostname ?? '')
    setResZone(NO_ZONE)
    setResDnsName(l.hostname ?? '')
    try {
      const { status } = await api.apiDhcpFreeIpGet({ target: l.ip_address })
      setResIP(status ?? '')
    } catch {
    }
  }

  const doReserve = async () => {
    if (!reserving) return
    const withZone = resZone !== NO_ZONE
    const dnsName = resDnsName.trim()
    setBusy(true)
    try {
      const { status } = await api.apiDhcpReservationsFromLeasePost({
        DhcpReserveFromLeaseRequest: {
          lease_ip: reserving.ip_address,
          ip: resIP.trim(),
          hostname: resHostname.trim(),
          dns_zone: withZone ? resZone : undefined,
          dns_name: withZone ? dnsName : undefined,
        },
      })
      toast.success(
        withZone
          ? `Reserved ${status} and added ${dnsName}.${resZone}. The client will pick up its new address shortly.`
          : `Reserved ${status}. The client will pick up its new address shortly.`,
      )
      setReserving(null)
      load(q)
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const subnetOptions = subnets.filter((s) => family === 'all' || subnetService(s.subnet) === family)
  const filtered = rows.filter((l) =>
    (family === 'all' || l.service === family) &&
    (subnetKey === 'all' || `${l.service}:${l.subnet_id}` === subnetKey),
  )
  const paged = usePagination(filtered, 15)

  const handleConfirm = () => { const ip = clearing; setClearing(null); if (ip) doClear(ip) }

  return (
    <Card>
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3 pb-2">
        <CardTitle className="text-sm">Active leases ({filtered.length})</CardTitle>
        <div className="flex flex-wrap items-center gap-2">
          <ReloadButton onClick={() => load(q)} loading={loading} />
          <Segmented
            value={family}
            onChange={changeFamily}
            className="text-xs"
            options={[
              { value: 'all', label: 'All' },
              { value: 'dhcp4', label: 'IPv4' },
              { value: 'dhcp6', label: 'IPv6' },
            ]}
          />
          <Select value={subnetKey} onValueChange={setSubnetKey}>
            <SelectTrigger className="h-8 w-44 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all" className="text-xs">All subnets</SelectItem>
              {subnetOptions.map((s) => {
                const key = `${subnetService(s.subnet)}:${s.id}`
                return <SelectItem key={key} value={key} className="font-mono text-xs">{s.subnet}</SelectItem>
              })}
            </SelectContent>
          </Select>
          <div className="relative w-72 max-w-full">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search IP, MAC, DUID, hostname" className="h-8 pl-8 text-xs" />
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {loadError && <p role="alert" className="mb-3 text-sm text-danger">Unable to refresh leases: {loadError}{rows.length > 0 ? ' Showing the last successful result.' : ''}</p>}
        {filtered.length === 0 && (loadError || loading) ? (
          loading && !loadError ? <p className="py-10 text-sm text-muted-foreground">Loading leases…</p> : null
        ) : filtered.length === 0 ? (
          q || family !== 'all' || subnetKey !== 'all' ? (
            <p className="text-sm text-muted-foreground">No Results</p>
          ) : (
            <EmptyState className="py-10" title="No Active Leases" />
          )
        ) : (
          <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>IP</TableHead>
                <TableHead>MAC / DUID</TableHead>
                <TableHead>Hostname</TableHead>
                <TableHead>Type</TableHead>
                <TableHead className="w-[1%]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {paged.pageItems.map((l) => {
                const ip = l.ip_address
                return (
                  <TableRow key={`${l.service}-${ip}`}>
                    <TableCell className="font-mono whitespace-nowrap">{ip}</TableCell>
                    <TableCell className="font-mono text-xs whitespace-nowrap">{l.service === 'dhcp6' ? l.duid : (l.hw_address || l.duid || '')}</TableCell>
                    <TableCell>{l.hostname || <span className="text-muted-foreground">-</span>}</TableCell>
                    <TableCell>
                      <Badge variant={l.reserved ? 'default' : 'secondary'} className="text-xs">
                        {l.reserved ? 'Reservation' : 'Dynamic'}
                      </Badge>
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-right">
                      {!l.reserved && (
                        <Button
                          variant="ghost" size="sm" className="gap-1.5" disabled={busy}
                          title="Create a Reservation for This Host and Clear the Lease"
                          onClick={() => openReserve(l)}
                        >
                          <Pin className="h-3.5 w-3.5" />Reserve…
                        </Button>
                      )}
                      <Button
                        variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive"
                        disabled={busy}
                        onClick={() => setClearing(ip)}
                        aria-label="Clear Lease" title="Clear Lease">
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
          </div>
        )}
        <Pagination page={paged.page} totalPages={paged.totalPages} total={paged.total} pageSize={paged.pageSize} onPage={paged.setPage} unit="leases" className="mt-3" />
      </CardContent>
      <AlertDialog
        open={clearing !== null}
        busy={busy}
        title={`Clear Lease ${clearing ?? ''}?`}
        confirmLabel="Clear"
        destructive
        onCancel={() => setClearing(null)}
        onConfirm={handleConfirm}
      />

      <Sheet
        open={reserving !== null}
        onClose={() => setReserving(null)}
        title="Reserve Host"
      >
        {reserving && (
          <div className="space-y-6">
            <PreferencesGroup description="Applies the reservation and clears the current lease so the client renews">
              <Row title="Current IP"><span className="break-all font-mono text-sm">{reserving.ip_address}</span></Row>
              <Row title={reserving.service === 'dhcp6' ? 'DUID' : 'MAC'}>
                <span className="break-all font-mono text-sm">{reserving.service === 'dhcp6' ? reserving.duid : (reserving.hw_address || reserving.duid || '-')}</span>
              </Row>
              <EntryRow title="Hostname" value={resHostname} onChange={(e) => setResHostname(e.target.value)} />
              <EntryRow title="Reservation IP" value={resIP} onChange={(e) => setResIP(e.target.value)} className="font-mono" />
            </PreferencesGroup>
            {zones.length > 0 && (
              <PreferencesGroup
                title="DNS Record"
                description={resZone === NO_ZONE
                  ? 'Pick a zone to also create an A or AAAA record, and a PTR when a reverse zone exists'
                  : `Creates ${resDnsName.trim() || 'name'}.${resZone} pointing at the reserved address`}
              >
                <ComboRow title="Zone">
                  <Select value={resZone} onValueChange={setResZone}>
                    <SelectTrigger className="font-mono"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value={NO_ZONE}>No DNS record</SelectItem>
                      {zones.map((z) => <SelectItem key={z} value={z} className="font-mono">{z}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </ComboRow>
                <EntryRow title="Name" value={resDnsName} onChange={(e) => setResDnsName(e.target.value)} className="font-mono" disabled={resZone === NO_ZONE} />
              </PreferencesGroup>
            )}
          </div>
        )}
        <div className="mt-6 flex justify-end gap-2">
          <Button variant="outline" onClick={() => setReserving(null)} disabled={busy}>Cancel</Button>
          <Button onClick={doReserve} disabled={busy || resIP.trim() === '' || (resZone !== NO_ZONE && resDnsName.trim() === '')} variant="suggested">Reserve</Button>
        </div>
      </Sheet>
    </Card>
  )
}
