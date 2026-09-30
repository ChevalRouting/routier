import { useState, useEffect, useRef, useCallback } from 'react'
import { toast } from 'sonner'
import { useTabState } from 'cheval-ui'
import { api, configLayerRequest } from '@/lib/client'
import type { DhcpStatsResponse as DhcpStats, DhcpLeaseView as DhcpLeaseRow, KeaSubnet } from '@/api'
import { getToken } from '@/lib/utils'
import { Card, CardContent, CardHeader, CardTitle } from 'cheval-ui'
import { Input } from 'cheval-ui'
import { Table, TableHeader, TableBody, TableHead, TableRow, TableCell } from 'cheval-ui'
import { Dialog, AlertDialog } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { Segmented } from 'cheval-ui'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from 'cheval-ui'
import { SectionNav } from 'cheval-ui'
import { StateChip } from 'cheval-ui'
import { Tabs } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { Pagination, usePagination } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Trash2, ChevronsDown, Search, Pin, RefreshCw, RotateCw } from 'lucide-react'

export const DHCP_KEY_STATS = [
  'declined-addresses',
  'pkt4-received', 'pkt4-ack-sent', 'pkt4-offer-sent',
  'pkt6-received', 'pkt6-reply-sent',
  'cumulative-assigned-addresses', 'cumulative-assigned-nas',
]

export function statValue(stats: { name: string; value: number }[] | null | undefined, name: string): number | undefined {
  return stats?.find((s) => s.name === name)?.value
}

export function DhcpStatGrid({ title, stats }: { title: string; stats: { name: string; value: number }[] | null }) {
  if (!stats || stats.length === 0) {
    return (
      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">{title}</CardTitle></CardHeader>
        <CardContent><p className="text-xs text-muted-foreground italic">No statistics (server not running).</p></CardContent>
      </Card>
    )
  }
  const shown = DHCP_KEY_STATS.map((n) => ({ n, v: statValue(stats, n) })).filter((x) => x.v !== undefined)
  return (
    <Card>
      <CardHeader className="pb-2"><CardTitle className="text-sm">{title}</CardTitle></CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          {shown.map(({ n, v }) => (
            <div key={n} className="space-y-0.5">
              <div className="text-lg font-semibold tabular-nums">{v}</div>
              <div className="text-[11px] text-muted-foreground font-mono truncate" title={n}>{n}</div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

type LeaseFamily = 'all' | 'dhcp4' | 'dhcp6'

const NO_ZONE = '__none__'

interface DnsZoneOption {
  name: string
  primaries?: string[]
}

interface DnsServerSection {
  zones?: DnsZoneOption[]
}

function forwardZoneNames(section: DnsServerSection | null): string[] {
  return (section?.zones ?? [])
    .filter((z) => (z.primaries ?? []).length === 0)
    .map((z) => z.name)
    .filter((n) => n && !n.endsWith('in-addr.arpa') && !n.endsWith('ip6.arpa'))
}

function subnetService(cidr: string): string {
  return cidr.includes(':') ? 'dhcp6' : 'dhcp4'
}

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
      ++requestID.current
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
      // leave blank; the user can enter one manually
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
        onConfirm={() => { const ip = clearing; setClearing(null); if (ip) doClear(ip) }}
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

export function DhcpStatusPanel() {
  const [stats, setStats] = useState<DhcpStats | null>(null)
  const [restarting, setRestarting] = useState(false)

  const load = useCallback(() => api.apiDhcpStatsGet().then(setStats).catch(() => {}), [])

  useEffect(() => {
    let alive = true
    const tick = () => api.apiDhcpStatsGet().then((s) => { if (alive) setStats(s) }).catch(() => {})
    tick()
    const id = setInterval(tick, 10_000)
    return () => { alive = false; clearInterval(id) }
  }, [])

  const running = (stats?.services ?? []).some((s) => s.running)

  const restart = async () => {
    setRestarting(true)
    try {
      await configLayerRequest('/api/dhcp/restart', { method: 'POST' })
      toast.success('DHCP server restarted')
      await load()
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setRestarting(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        {(stats?.services ?? []).map((s) => (
          <div key={s.service} className="flex items-center gap-1.5">
            <span className="text-xs font-mono text-muted-foreground">{s.service}</span>
            <StateChip state={s.running ? 'RUNNING' : 'STOPPED'} />
          </div>
        ))}
        <Button variant="outline" size="sm" className="gap-1.5" disabled={restarting || !running} onClick={() => void restart()}>
          <RotateCw className={`h-3.5 w-3.5 ${restarting ? 'animate-spin' : ''}`} /> Restart
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <DhcpStatGrid title="IPv4 statistics" stats={stats?.stats4 ?? null} />
        <DhcpStatGrid title="IPv6 statistics" stats={stats?.stats6 ?? null} />
      </div>

      <DhcpLeases />
    </div>
  )
}

type DhcpLogSource = 'kea-dhcp4' | 'kea-dhcp6' | 'kea-dhcp-ddns'
const DHCP_LOG_TABS: { key: DhcpLogSource; label: string }[] = [
  { key: 'kea-dhcp4', label: 'kea-dhcp4' },
  { key: 'kea-dhcp6', label: 'kea-dhcp6' },
  { key: 'kea-dhcp-ddns', label: 'kea-dhcp-ddns' },
]

export function DhcpLogsPanel() {
  const [source, setSource] = useTabState<DhcpLogSource>('monitor.dhcp.logs', 'kea-dhcp4')
  const [lines, setLines] = useState<string[]>([])
  const [streaming, setStreaming] = useState(false)
  const [autoScroll, setAutoScroll] = useState(true)
  const scrollBoxRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  const stickToBottom = () => {
    const el = scrollBoxRef.current
    if (el) el.scrollTop = el.scrollHeight
  }

  useEffect(() => {
    if (autoScroll) stickToBottom()
  }, [lines, autoScroll])

  const handleScroll = () => {
    const el = scrollBoxRef.current
    if (!el) return
    setAutoScroll(el.scrollHeight - el.scrollTop - el.clientHeight < 32)
  }

  const startStream = useCallback(async (selectedSource: DhcpLogSource) => {
    if (abortRef.current) abortRef.current.abort()
    const controller = new AbortController()
    abortRef.current = controller
    setStreaming(true); setAutoScroll(true)
    try {
      const res = await fetch(`/api/dhcp/leases/stream?source=${encodeURIComponent(selectedSource)}`, {
        headers: { Authorization: `Bearer ${getToken() ?? ''}` },
        signal: controller.signal,
      })
      if (!res.ok || !res.body) throw new Error(`Unable to stream ${selectedSource} logs (${res.status})`)
      const reader = res.body.getReader(); const decoder = new TextDecoder(); let buffer = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done || controller.signal.aborted) break
        buffer += decoder.decode(value, { stream: true })
        const parts = buffer.split('\n\n'); buffer = parts.pop() ?? ''
        for (const part of parts) {
          const line = part.split('\n').filter((value) => value.startsWith('data:')).map((value) => value.slice(5).trimStart()).join('\n')
          if (line) setLines((prev) => { const next = [...prev, line]; return next.length > 2000 ? next.slice(-2000) : next })
        }
      }
    } catch (err: unknown) {
      if (!controller.signal.aborted && err instanceof Error) toast.error(err.message)
    } finally { if (abortRef.current === controller) setStreaming(false) }
  }, [])

  const stopStream = () => { abortRef.current?.abort(); abortRef.current = null; setStreaming(false) }

  useEffect(() => {
    setLines([])
    startStream(source)
    return () => { abortRef.current?.abort() }
  }, [source, startStream])

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Tabs tabs={DHCP_LOG_TABS} active={source} onChange={setSource} variant="pills" />
        <div className="flex items-center gap-2">
          <Button size="sm" variant={streaming ? 'default' : 'outline'} onClick={streaming ? stopStream : () => startStream(source)}
            className={`gap-1.5 ${streaming ? 'bg-green-600 hover:bg-green-700 border-green-600 dark:bg-green-700 dark:hover:bg-green-600' : ''}`}>
            <span className={`h-1.5 w-1.5 rounded-full inline-block ${streaming ? 'bg-white animate-pulse' : 'bg-muted-foreground'}`} />
            Live
          </Button>
          <Button size="sm" variant="ghost" onClick={() => setLines([])} className="gap-1.5 text-muted-foreground">
            <Trash2 className="h-3.5 w-3.5" />Clear
          </Button>
          <span className="text-xs text-muted-foreground">{lines.length} lines</span>
        </div>
      </div>

      <div className="relative">
        <div ref={scrollBoxRef} onScroll={handleScroll} className="bg-zinc-950 text-zinc-100 font-mono text-xs p-3 rounded-md overflow-auto h-[520px]">
          {lines.length === 0
            ? <span className="text-zinc-500">{streaming ? `Waiting for ${source} log output…` : 'Press Live to begin streaming logs.'}</span>
            : lines.map((line, i) => <div key={i}>{line}</div>)
          }
        </div>
        {!autoScroll && lines.length > 0 && (
          <button type="button" onClick={() => { setAutoScroll(true); stickToBottom() }}
            className="absolute bottom-3 right-5 flex items-center gap-1 rounded-full bg-zinc-700 hover:bg-zinc-600 text-zinc-100 text-xs px-2.5 py-1 shadow">
            <ChevronsDown className="h-3 w-3" />Tail
          </button>
        )}
      </div>
    </div>
  )
}

export function DhcpPanel() {
  const [sub, setSub] = useTabState<'status' | 'logs'>('dhcp', 'status')
  return (
    <SectionNav
      items={[{ key: 'status', label: 'Status & Leases' }, { key: 'logs', label: 'Logs' }]}
      active={sub}
      onChange={setSub}
    >
      {sub === 'status' && <DhcpStatusPanel />}
      {sub === 'logs' && <DhcpLogsPanel />}
    </SectionNav>
  )
}

export type MonitorTab = 'system' | 'bgp' | 'traffic' | 'logs' | 'ha-status' | 'neighbors' | 'processes' | 'collection' | 'dhcp'
