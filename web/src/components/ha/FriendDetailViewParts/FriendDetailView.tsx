import type {
  TypesFriendInfo as FriendInfo
} from '@/api'
import { ConfigSections } from '@/components/ConfigView'
import { TunnelDialog, WireguardDialog } from '@/components/ha/dialogs'
import { ConntrackTab, Field, Tab, TABS, TunCard, TunInstance, VrrpTab, WgCard, WgInstance } from '@/components/ha/FriendDetailView'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Badge, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, PageHeader, SectionNav, Spinner } from 'cheval-ui'
import { ArrowLeft, Network, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

type FriendDetailViewShape = { name: string; onBack: () => void }

export function FriendDetailView({ name, onBack }: FriendDetailViewShape) {
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

  const handleDone2 = () => { cache.reload(); nft.reload(); cfg.reload(); local.reload(); ha.reload() }

  const handleDone = () => { cache.reload(); nft.reload(); cfg.reload(); local.reload(); ha.reload() }

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
          onDone={handleDone}
        />
      )}

      {tunOpen && (
        <TunnelDialog
          friend={{ name } as FriendInfo}
          onClose={() => setTunOpen(false)}
          onDone={handleDone2}
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
