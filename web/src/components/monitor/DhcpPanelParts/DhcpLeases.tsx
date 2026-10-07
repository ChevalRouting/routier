import type { DhcpLeaseView as DhcpLeaseRow, KeaSubnet } from '@/api'
import { DnsServerSection, LeaseFamily, NO_ZONE, forwardZoneNames, subnetService } from '@/components/monitor/DhcpPanel'
import { api } from '@/lib/client'
import { AlertDialog, Badge, Button, Card, CardContent, CardHeader, CardTitle, Dialog, EmptyState, Input, Label, Pagination, Segmented, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, usePagination } from 'cheval-ui'
import { Pin, RefreshCw, Search, Trash2 } from 'lucide-react'
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
          <Button variant="outline" size="sm" disabled={loading} onClick={() => load(q)} className="gap-1.5">
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
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
          <div className="relative w-64 max-w-full">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search IP, MAC, DUID, hostname" className="h-8 pl-8 text-xs" />
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {loadError && <p role="alert" className="mb-3 text-sm text-destructive">Unable to refresh leases: {loadError}{rows.length > 0 ? ' Showing the last successful result.' : ''}</p>}
        {filtered.length === 0 && (loadError || loading) ? (
          loading && !loadError ? <p className="py-10 text-sm text-muted-foreground">Loading leases…</p> : null
        ) : filtered.length === 0 ? (
          q || family !== 'all' || subnetKey !== 'all' ? (
            <p className="text-sm text-muted-foreground italic">No leases match.</p>
          ) : (
            <EmptyState className="py-10" title="No active leases" message="Leases show up here as clients obtain addresses." />
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
                      <Badge variant={l.reserved ? 'default' : 'secondary'} className="text-[10px]">
                        {l.reserved ? 'Reservation' : 'Dynamic'}
                      </Badge>
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-right">
                      {!l.reserved && (
                        <Button
                          variant="ghost" size="sm" className="gap-1.5" disabled={busy}
                          title="Create a reservation for this host and clear the lease"
                          onClick={() => openReserve(l)}
                        >
                          <Pin className="h-3.5 w-3.5" />Reserve
                        </Button>
                      )}
                      <Button
                        variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive"
                        title="Clear lease" disabled={busy}
                        onClick={() => setClearing(ip)}
                      >
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
        title={`Clear lease ${clearing ?? ''}?`}
        confirmLabel="Clear"
        destructive
        onCancel={() => setClearing(null)}
        onConfirm={handleConfirm}
      />

      <Dialog
        open={reserving !== null}
        onClose={() => setReserving(null)}
        title="Reserve host"
        description="Reserves this address for the host, applies it, and clears the current lease so the client picks up its new address."
        footer={
          <>
            <Button variant="outline" onClick={() => setReserving(null)} disabled={busy}>Cancel</Button>
            <Button onClick={doReserve} disabled={busy || resIP.trim() === '' || (resZone !== NO_ZONE && resDnsName.trim() === '')}>Reserve</Button>
          </>
        }
      >
        {reserving && (
          <div className="space-y-4">
            <div className="grid grid-cols-[7rem_1fr] items-baseline gap-x-3 gap-y-1.5 text-sm">
              <span className="text-muted-foreground">Current IP</span>
              <span className="font-mono break-all">{reserving.ip_address}</span>
              <span className="text-muted-foreground">{reserving.service === 'dhcp6' ? 'DUID' : 'MAC'}</span>
              <span className="font-mono break-all">{reserving.service === 'dhcp6' ? reserving.duid : (reserving.hw_address || reserving.duid || '-')}</span>
            </div>
            <div className="space-y-1.5">
              <Label>Hostname</Label>
              <Input value={resHostname} onChange={(e) => setResHostname(e.target.value)} placeholder="hostname (optional)" />
            </div>
            <div className="space-y-1.5">
              <Label>Reservation IP</Label>
              <Input value={resIP} onChange={(e) => setResIP(e.target.value)} placeholder="pre-filled with a free address" className="font-mono" />
              <p className="text-[11px] text-muted-foreground">Pre-filled with a free out-of-pool address; change it if you like.</p>
            </div>
            {zones.length > 0 && (
              <div className="space-y-1.5 rounded-md bg-muted/30 p-3">
                <Label>DNS record (optional)</Label>
                <div className="flex gap-2">
                  <Input
                    value={resDnsName}
                    onChange={(e) => setResDnsName(e.target.value)}
                    placeholder="name"
                    className="font-mono"
                    disabled={resZone === NO_ZONE}
                  />
                  <Select value={resZone} onValueChange={setResZone}>
                    <SelectTrigger className="w-52 font-mono text-xs"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value={NO_ZONE} className="text-xs">No DNS record</SelectItem>
                      {zones.map((z) => (
                        <SelectItem key={z} value={z} className="font-mono text-xs">{z}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <p className="text-[11px] text-muted-foreground">
                  {resZone === NO_ZONE
                    ? 'Pick a zone to also create an A/AAAA record (and a PTR when a reverse zone exists).'
                    : `Creates ${resDnsName.trim() || 'name'}.${resZone} pointing at the reserved address.`}
                </p>
              </div>
            )}
          </div>
        )}
      </Dialog>
    </Card>
  )
}
