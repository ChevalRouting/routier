import { useState, useEffect, type ReactNode } from 'react'
import { toast } from 'sonner'
import { ArrowLeft, RefreshCw, Network, ChevronDown, ChevronRight } from 'lucide-react'
import { api } from '@/lib/client'
import type {
  TypesFriendInfo as FriendInfo, TypesConntrackdStatus as ConntrackdStatus,
  TypesFriendInterface as FriendInterface, TypesVRRPInstanceStatus as VRRPInstanceStatus,
} from '@/api'
import { useFetch } from '@/lib/useFetch'
import { Card, CardContent, CardHeader, CardTitle } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { Input } from 'cheval-ui'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from 'cheval-ui'
import { SectionNav } from 'cheval-ui'
import { PageHeader } from 'cheval-ui'
import { Spinner } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { ConfigSections, ValueNode } from '@/components/ConfigView'
import { WireguardDialog, TunnelDialog } from '@/components/ha/dialogs'

function Field({ label, value, mono }: { label: string; value?: ReactNode; mono?: boolean }) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={`truncate ${mono ? 'font-mono text-xs' : ''}`}>{value ?? '-'}</div>
    </div>
  )
}

type Tab = 'overview' | 'wireguard' | 'tunnel' | 'conntrack' | 'vrrp'
const TABS: { key: Tab; label: string }[] = [
  { key: 'overview',  label: 'Overview' },
  { key: 'wireguard', label: 'WireGuard' },
  { key: 'tunnel',    label: 'Tunnel' },
  { key: 'conntrack', label: 'Conntrack' },
  { key: 'vrrp',      label: 'VRRP' },
]

interface WgPeer { name?: string; public_key?: string; endpoint?: string; allowed_ips?: string[]; keepalive?: number }
interface WgInstance {
  friend?: string
  listen_port?: number
  addresses?: string[]
  table?: string
  mtu?: number
  peers?: WgPeer[]
}

function WgCard({ name, wg }: { name: string; wg: WgInstance }) {
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

interface TunInstance {
  friend?: string
  mode?: string
  local?: string
  remote?: string
  ttl?: number
  addresses?: string[]
  mtu?: number
}

function TunCard({ name, tun }: { name: string; tun: TunInstance }) {
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

function ConntrackTab({ friendName, cd }: { friendName: string; cd?: ConntrackdStatus }) {
  const { data: cfg } = useFetch(() => api.apiConfigGet() as unknown as Promise<Record<string, unknown>>)
  const { data: selfIfaces } = useFetch(() => api.apiFriendsInterfacesGet())
  const { data: friendIfaces } = useFetch(() => api.apiFriendsNameInterfacesGet({ name: friendName }))
  const [localSel, setLocalSel] = useState('')
  const [friendSel, setFriendSel] = useState('')
  const [port, setPort] = useState('')
  const [allowInbound, setAllowInbound] = useState(false)
  const [saving, setSaving] = useState(false)
  const [init, setInit] = useState(false)

  useEffect(() => {
    if (!cfg || init) return
    const ex = (cfg.conntrackd ?? {}) as { interface?: string; address?: string; port?: number; allow_inbound?: boolean }
    if (ex.interface && ex.address) setLocalSel(`${ex.interface}|${ex.address}`)
    setPort(ex.port ? String(ex.port) : '')
    setAllowInbound(!!ex.allow_inbound)
    setInit(true)
  }, [cfg, init])

  const opts = (ifs: FriendInterface[] | null) =>
    (ifs ?? []).flatMap((i) => (i.addresses ?? []).map((a) => ({ iface: i.name, addr: a })))

  const ipSelect = (value: string, set: (v: string) => void, ifs: FriendInterface[] | null) => (
    <Select value={value} onValueChange={set}>
      <SelectTrigger className="h-9"><SelectValue placeholder="select IP" /></SelectTrigger>
      <SelectContent>
        {opts(ifs).map((o) => (
          <SelectItem key={`${o.iface}|${o.addr}`} value={`${o.iface}|${o.addr}`} className="font-mono">{o.addr} ({o.iface})</SelectItem>
        ))}
      </SelectContent>
    </Select>
  )

  const save = async () => {
    const [li, la] = localSel.split('|')
    const [fi, fa] = friendSel.split('|')
    if (!li || !la) { toast.error('Select our sync IP'); return }
    if (!fi || !fa) { toast.error("Select the friend's sync IP"); return }
    setSaving(true)
    try {
      await api.apiFriendsNameConntrackPost({ name: friendName, TypesConfigureConntrackRequest: {
        local_interface: li, local_address: la,
        friend_interface: fi, friend_address: fa,
        ...(port ? { port: Number(port) } : {}),
        allow_inbound: allowInbound,
      } })
      toast.success('conntrackd configured and applied on both sides')
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader><CardTitle>Conntrackd status</CardTitle></CardHeader>
        <CardContent className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm md:grid-cols-3">
          <Field label="Running" value={<Badge variant={cd?.running ? 'success' : 'destructive'}>{cd?.running ? 'yes' : 'no'}</Badge>} />
          <Field label="Tracked entries" value={cd?.entries} />
          {cd?.by_proto && Object.entries(cd.by_proto).map(([p, n]) => <Field key={p} label={`${p} entries`} value={n} />)}
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Configure conntrackd with this friend</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          {!cfg ? <Spinner /> : (
            <>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted-foreground">Our sync IP</label>
                  {ipSelect(localSel, setLocalSel, selfIfaces)}
                </div>
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted-foreground">Friend sync IP</label>
                  {ipSelect(friendSel, setFriendSel, friendIfaces ?? null)}
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted-foreground">Port (optional)</label>
                  <Input value={port} onChange={(e) => setPort(e.target.value)} className="font-mono text-sm" placeholder="3780" />
                </div>
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted-foreground">Allow inbound</label>
                  <div>
                    <button type="button" onClick={() => setAllowInbound((v) => !v)}>
                      <Badge variant={allowInbound ? 'success' : 'outline'} className="cursor-pointer">{allowInbound ? 'yes' : 'no'}</Badge>
                    </button>
                  </div>
                </div>
              </div>
              <Button onClick={save} disabled={saving}>{saving ? 'Applying…' : 'Configure & apply on both sides'}</Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

interface VrrpConfigInstance { name?: string; friend?: string; id?: number; interface?: string; vips?: string[]; priority?: number }

function VrrpTab({ friendName, vrrpStatus }: { friendName: string; vrrpStatus: VRRPInstanceStatus[] }) {
  const { data: cfg, reload } = useFetch(() => api.apiConfigGet() as unknown as Promise<Record<string, unknown>>)
  const ha = (cfg?.ha ?? {}) as { vrrp?: VrrpConfigInstance[] }
  const interfaceNames = Object.keys((cfg?.interfaces ?? {}) as Record<string, unknown>)

  const friendVrrp: { iface: string; v: VrrpConfigInstance }[] = []
  for (const v of ha.vrrp ?? []) {
    if (v.friend === friendName) friendVrrp.push({ iface: v.interface ?? '', v })
  }

  const { data: friendIfaces } = useFetch(() => api.apiFriendsNameInterfacesGet({ name: friendName }))

  const [localIface, setLocalIface] = useState('')
  const [friendIface, setFriendIface] = useState('')
  const [vrid, setVrid] = useState('')
  const [vips, setVips] = useState<string[]>([])
  const [vipInput, setVipInput] = useState('')
  const [priority, setPriority] = useState('')
  const [saving, setSaving] = useState(false)

  const statusFor = (id?: number) => vrrpStatus.find((s) => s.id === id)

  const addVip = () => {
    const v = vipInput.trim()
    if (v && !vips.includes(v)) setVips((p) => [...p, v])
    setVipInput('')
  }

  const addInstance = async () => {
    if (!localIface || !friendIface || !vrid) { toast.error('Our interface, friend interface and VRID are required'); return }
    if (vips.length === 0) { toast.error('At least one VIP is required'); return }
    setSaving(true)
    try {
      await api.apiFriendsNameVrrpPost({ name: friendName, TypesConfigureVRRPRequest: {
        local_interface: localIface,
        friend_interface: friendIface,
        vrid: Number(vrid),
        vips,
        ...(priority ? { priority: Number(priority) } : {}),
      } })
      toast.success('VRRP configured and applied on both sides')
      setVrid(''); setVips([]); setVipInput(''); setPriority('')
      reload()
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setSaving(false)
    }
  }

  const removeInstance = async (_ifn: string, id?: number) => {
    const updated = JSON.parse(JSON.stringify(ha)) as { vrrp?: VrrpConfigInstance[] }
    updated.vrrp = (updated.vrrp ?? []).filter((v) => !(v.id === id && v.friend === friendName))
    if (updated.vrrp.length === 0) delete updated.vrrp
    setSaving(true)
    try {
      await api.apiConfigSectionPut({ section: 'ha', body: updated })
      toast.success('VRRP instance removed. Apply to take effect')
      reload()
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-4">
      {friendVrrp.length === 0 ? (
        <EmptyState className="py-10" title="No VRRP instances" message="Configure VRRP below to create a failover group with this friend." />
      ) : friendVrrp.map(({ iface: ifn, v }) => {
        const s = statusFor(v.id)
        return (
          <Card key={`${ifn}-${v.id}`}>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between gap-2">
                <CardTitle className="flex items-center gap-2 text-base">
                  {v.name || `${ifn} (VRID ${v.id})`}
                  {s && <Badge variant={s.state === 'MASTER' ? 'success' : s.state === 'BACKUP' ? 'secondary' : 'destructive'}>{s.state}</Badge>}
                </CardTitle>
                <Button variant="ghost" size="sm" className="text-destructive" disabled={saving} onClick={() => removeInstance(ifn, v.id)}>Remove</Button>
              </div>
            </CardHeader>
            <CardContent className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm md:grid-cols-3">
              <Field label="Interface" value={ifn} mono />
              <Field label="VRID" value={v.id} />
              <Field label="Priority" value={v.priority} />
              <Field label="VIPs" value={(v.vips ?? []).join(', ')} mono />
              <Field label="Master" value={s?.master_ip} mono />
            </CardContent>
          </Card>
        )
      })}

      <Card>
        <CardHeader><CardTitle>Configure VRRP with this friend</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          {!cfg ? <Spinner /> : (
            <>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted-foreground">Our interface</label>
                  <Select value={localIface} onValueChange={setLocalIface}>
                    <SelectTrigger className="h-9"><SelectValue placeholder="select interface" /></SelectTrigger>
                    <SelectContent>
                      {interfaceNames.map((k) => <SelectItem key={k} value={k} className="font-mono">{k}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted-foreground">Friend interface</label>
                  <Select value={friendIface} onValueChange={setFriendIface}>
                    <SelectTrigger className="h-9"><SelectValue placeholder="select interface" /></SelectTrigger>
                    <SelectContent>
                      {(friendIfaces ?? []).map((i) => <SelectItem key={i.name} value={i.name} className="font-mono">{i.name}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted-foreground">VRID</label>
                  <Input value={vrid} onChange={(e) => setVrid(e.target.value)} className="font-mono text-sm" placeholder="51" />
                </div>
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-muted-foreground">Priority (optional)</label>
                  <Input value={priority} onChange={(e) => setPriority(e.target.value)} className="font-mono text-sm" placeholder="150" />
                </div>
              </div>
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">Virtual IPs</label>
                <div className="flex flex-wrap items-center gap-2">
                  {vips.map((v) => (
                    <button key={v} type="button" onClick={() => setVips((p) => p.filter((x) => x !== v))}>
                      <Badge variant="default" className="cursor-pointer gap-1 font-mono">{v} ×</Badge>
                    </button>
                  ))}
                  <Input
                    value={vipInput}
                    onChange={(e) => setVipInput(e.target.value)}
                    onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addVip() } }}
                    onBlur={addVip}
                    className="h-8 w-44 font-mono text-sm"
                    placeholder="10.0.0.1/24 ⏎"
                  />
                </div>
              </div>
              <Button onClick={addInstance} disabled={saving}>{saving ? 'Applying…' : 'Configure & apply on both sides'}</Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export function FriendDetailView({ name, onBack }: { name: string; onBack: () => void }) {
  const [tab, setTab] = useState<Tab>('overview')
  const [checking, setChecking] = useState(false)
  const [pairing, setPairing] = useState(false)
  const [wgOpen, setWgOpen] = useState(false)
  const [tunOpen, setTunOpen] = useState(false)

  const cache = useFetch(() => api.apiFriendsNameCacheGet({ name }))
  const nft = useFetch(() => api.apiFriendsNameNftablesGet({ name }))
  const cfg = useFetch(() => api.apiFriendsNameConfigGet({ name }))
  const local = useFetch(() => api.apiConfigGet() as unknown as Promise<Record<string, unknown>>)
  const ha = useFetch(() => api.apiHaStatusGet())
  const friends = useFetch(() => api.apiFriendsGet())

  const info = friends.data?.find((f) => f.name === name)
  const paired = info?.paired
  const st = cache.data?.status
  const loading =
    (cache.isLoading && !cache.data) ||
    (nft.isLoading && !nft.data) ||
    (cfg.isLoading && !cfg.data)

  function reloadAll() {
    cache.reload(); nft.reload(); cfg.reload(); local.reload(); ha.reload(); friends.reload()
  }

  async function check() {
    setChecking(true)
    try {
      const res = await api.apiFriendsNameCheckPost({ name })
      if (res.reachable) toast.success(`${name} reachable (${res.rtt_ms ?? 0} ms)`)
      else toast.error(res.last_error || `${name} unreachable`)
      reloadAll()
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setChecking(false)
    }
  }

  async function pair() {
    setPairing(true)
    try {
      const res = await api.apiFriendsNamePairPost({ name, TypesVerifyFriendRequest: { fingerprint: info?.fingerprint } })
      if (res.paired) toast.success(`Paired with ${name}: encryption established`)
      else toast.error(`Pairing with ${name} did not complete`)
      reloadAll()
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setPairing(false)
    }
  }

  const cd = ha.data?.conntrackd

  return (
    <div className="space-y-6">
      <PageHeader
        title={`Friend: ${name}`}
        description={st?.hostname}
        action={
          <div className="flex items-center gap-2">
            <Button variant="ghost" onClick={onBack} className="gap-1.5">
              <ArrowLeft className="h-4 w-4" />Back
            </Button>
            {paired === false && (
              <Button variant="outline" onClick={pair} disabled={pairing}>
                <Network className={`mr-2 h-4 w-4 ${pairing ? 'animate-pulse' : ''}`} />
                {pairing ? 'Pairing…' : 'Verify & pair'}
              </Button>
            )}
            <Button onClick={check} disabled={checking}>
              <RefreshCw className={`mr-2 h-4 w-4 ${checking ? 'animate-spin' : ''}`} />
              Check now
            </Button>
          </div>
        }
      />

      {wgOpen && (
        <WireguardDialog
          friend={{ name } as FriendInfo}
          onClose={() => setWgOpen(false)}
          onDone={() => { cache.reload(); nft.reload(); cfg.reload(); local.reload(); ha.reload() }}
        />
      )}

      {tunOpen && (
        <TunnelDialog
          friend={{ name } as FriendInfo}
          onClose={() => setTunOpen(false)}
          onDone={() => { cache.reload(); nft.reload(); cfg.reload(); local.reload(); ha.reload() }}
        />
      )}

      {loading ? (
        <div className="flex justify-center py-16"><Spinner /></div>
      ) : (
        <SectionNav items={TABS} active={tab} onChange={setTab}>
          {tab === 'overview' && (
            <div className="space-y-6">
              <div className="grid gap-6 lg:grid-cols-2">
                <Card className="h-full">
                  <CardHeader><CardTitle>Status</CardTitle></CardHeader>
                  <CardContent className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm md:grid-cols-3">
                    <Field label="Reachable" value={
                      <Badge variant={st?.reachable ? 'success' : 'destructive'}>{st?.reachable ? 'yes' : 'no'}</Badge>
                    } />
                    <Field label="Routier version" value={st?.version} />
                    <Field label="OS" value={st?.os} />
                    <Field label="RTT" value={st?.rtt_ms != null ? `${st.rtt_ms} ms` : undefined} />
                    <Field label="Last seen" value={st?.last_seen} />
                    <Field label="Identity match" value={st ? (st.identity_match ? 'yes' : 'no') : undefined} />
                    <Field label="Encryption" value={
                      paired == null ? undefined : <Badge variant={paired ? 'success' : 'destructive'}>{paired ? 'paired' : 'not paired'}</Badge>
                    } />
                    <Field label="Conntrackd" value={st ? (st.conntrackd_running ? 'running' : 'no') : undefined} />
                    <Field label="Fingerprint" value={st?.fingerprint} mono />
                    {st?.last_error && <Field label="Last error" value={<span className="text-destructive">{st.last_error}</span>} />}
                  </CardContent>
                </Card>

                <Card className="h-full">
                  <CardHeader><CardTitle>Cached state</CardTitle></CardHeader>
                  <CardContent className="space-y-2 text-sm">
                    <div className="text-muted-foreground">
                      {cache.data
                        ? `${(cache.data.interfaces ?? []).length} cached interface(s) · config ${cache.data.config ? 'cached' : 'not cached'}`
                        : '-'}
                    </div>
                    {(cache.data?.interfaces ?? []).map((i) => (
                      <div key={i.name} className="font-mono text-xs">
                        <span className="text-muted-foreground">{i.name}:</span> {(i.addresses ?? []).join(', ')}
                      </div>
                    ))}
                  </CardContent>
                </Card>
              </div>

              <Card>
                <CardHeader><CardTitle>Imported nftables variables</CardTitle></CardHeader>
                <CardContent className="text-sm">
                  {nft.data && nft.data.length > 0 ? (
                    <div className="max-h-80 space-y-1.5 overflow-auto font-mono text-xs">
                      {nft.data.map((v) => (
                        <div key={v.name}><span className="text-primary">${v.name}</span> = {v.value}</div>
                      ))}
                    </div>
                  ) : (
                    <div className="text-muted-foreground">none</div>
                  )}
                </CardContent>
              </Card>

              <Card>
                <CardHeader><CardTitle>Config browser</CardTitle></CardHeader>
                <CardContent>
                  {cfg.data ? (
                    <ConfigSections data={cfg.data as unknown as Record<string, unknown>} defaultOpen={false} />
                  ) : cfg.error ? (
                    <div className="space-y-2">
                      <div className="text-sm text-destructive">{cfg.error}</div>
                      {paired === false && (
                        <div className="text-sm text-muted-foreground">
                          This friend isn't paired yet, so its config can't be decrypted. Validate its
                          fingerprint and pair to establish encryption.{' '}
                          <Button variant="outline" size="sm" className="ml-1" onClick={pair} disabled={pairing}>
                            {pairing ? 'Pairing…' : 'Verify & pair'}
                          </Button>
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="text-sm text-muted-foreground">empty</div>
                  )}
                </CardContent>
              </Card>
            </div>
          )}

          {tab === 'wireguard' && (() => {
            const wg = (local.data?.wireguard ?? {}) as Record<string, WgInstance>
            const common = Object.entries(wg).filter(([, v]) => v?.friend === name)
            return (
              <div className="space-y-4">
                <Button variant="outline" onClick={() => setWgOpen(true)} className="gap-2">
                  <Network className="h-4 w-4" />Derive tunnel
                </Button>
                {!local.data ? (
                  <Spinner />
                ) : common.length === 0 ? (
                  <EmptyState className="py-8" title="No WireGuard tunnels" message="Derive a tunnel from the friend card to link the two routers." />
                ) : (
                  common.map(([n, w]) => <WgCard key={n} name={n} wg={w} />)
                )}
              </div>
            )
          })()}

          {tab === 'tunnel' && (() => {
            const tuns = (local.data?.tunnels ?? {}) as Record<string, TunInstance>
            const common = Object.entries(tuns).filter(([, v]) => v?.friend === name)
            return (
              <div className="space-y-4">
                <Button variant="outline" onClick={() => setTunOpen(true)} className="gap-2">
                  <Network className="h-4 w-4" />Derive tunnel
                </Button>
                {!local.data ? (
                  <Spinner />
                ) : common.length === 0 ? (
                  <EmptyState className="py-8" title="No tunnels" message="Derive a GRE or IPIP tunnel from the friend card." />
                ) : (
                  common.map(([n, t]) => <TunCard key={n} name={n} tun={t} />)
                )}
              </div>
            )
          })()}

          {tab === 'conntrack' && <ConntrackTab friendName={name} cd={cd} />}

          {tab === 'vrrp' && <VrrpTab friendName={name} vrrpStatus={ha.data?.vrrp ?? []} />}
        </SectionNav>
      )}
    </div>
  )
}
