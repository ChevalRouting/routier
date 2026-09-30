import { useState, useEffect } from 'react'
import { api } from '@/lib/client'
import type { TypesFriendInfo as FriendInfo, TypesFriendStatus as FriendStatus } from '@/api'
import { useFetch } from '@/lib/useFetch'
import { Button } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { Switch } from 'cheval-ui'
import { Sheet } from 'cheval-ui'
import { Dialog } from 'cheval-ui'
import { PreferencesGroup, EntryRow } from 'cheval-ui'
import { PageHeader } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { Spinner } from 'cheval-ui'
import { Plus, Trash2, Send, ShieldAlert, ShieldCheck, Pencil, RefreshCw, Handshake } from 'lucide-react'
import { toast } from 'sonner'
import { FriendDetailView } from '@/components/ha/FriendDetailView'

function statusColor(st?: FriendStatus): string {
  if (!st) return 'bg-muted-foreground/40'
  if (!st.reachable) return 'bg-danger'
  return 'bg-success'
}

function AddFriendSheet({ onClose, onAdded }: { onClose: () => void; onAdded: () => void }) {
  const [url, setUrl] = useState('')
  const [token, setToken] = useState('')
  const [tlsSkip, setTlsSkip] = useState(true)
  const [name, setName] = useState('')
  const [added, setAdded] = useState<FriendInfo | null>(null)
  const [busy, setBusy] = useState(false)

  const doAdd = async () => {
    if (!name || !url || !token) { toast.error('Name, URL and token are required'); return }
    setBusy(true)
    try {
      const info = await api.apiFriendsPost({ TypesAddFriendRequest: { name, url, token, tls_skip_verify: tlsSkip } })
      onAdded()
      if (info.fingerprint && !info.paired) {
        setAdded(info)
      } else {
        toast.success(info.fingerprint ? `Added friend ${name}` : `Added ${name} (unreachable, verify later)`)
        onClose()
      }
    } catch (err) {
      toast.error((err as Error).message)
    } finally { setBusy(false) }
  }

  const doVerify = async () => {
    if (!added) return
    setBusy(true)
    try {
      await api.apiFriendsNamePairPost({ name: added.name, TypesVerifyFriendRequest: { fingerprint: added.fingerprint } })
      toast.success(`Paired with ${added.name}: encryption established`)
      onAdded()
      onClose()
    } catch (err) {
      toast.error((err as Error).message)
    } finally { setBusy(false) }
  }

  if (added) {
    return (
      <div className="space-y-5">
        <div className="rounded-md bg-muted/30 p-3 space-y-2 text-sm">
          <div className="flex justify-between"><span className="text-muted-foreground">Friend</span><span className="font-mono">{added.name}</span></div>
          {added.hostname && <div className="flex justify-between"><span className="text-muted-foreground">Hostname</span><span className="font-mono">{added.hostname}</span></div>}
          <div className="flex justify-between gap-4"><span className="text-muted-foreground">Fingerprint</span><span className="font-mono text-xs break-all text-right">{added.fingerprint}</span></div>
          <p className="text-xs text-muted-foreground">
            Confirm this fingerprint matches the remote node out-of-band, then pair to establish encryption.
            The friend stays added either way; you can verify later from its detail page.
          </p>
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={onClose}>Verify later</Button>
          <Button onClick={doVerify} disabled={busy}>{busy ? 'Pairing…' : 'Verify & pair'}</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        <EntryRow title="Friend name" value={name} onChange={(e) => setName(e.target.value)}
          placeholder="edge-b" className="font-mono" autoFocus />
        <EntryRow title="URL" value={url} onChange={(e) => setUrl(e.target.value)}
          placeholder="https://10.0.0.2:8080" className="font-mono" />
        <EntryRow title="Shared token" type="password" value={token} onChange={(e) => setToken(e.target.value)}
          placeholder="bearer token shared by both nodes" className="font-mono" />
        <div className="flex items-center justify-between px-4 py-2.5">
          <span className="text-sm">Skip TLS verification</span>
          <Switch checked={tlsSkip} onCheckedChange={setTlsSkip} />
        </div>
      </PreferencesGroup>

      <p className="text-xs text-muted-foreground px-1">
        Adding always succeeds. If the friend is reachable we fetch its fingerprint so you can validate it
        and pair; otherwise it is saved unverified and you can pair later once it is up.
      </p>
      <div className="flex justify-end gap-2">
        <Button variant="outline" onClick={onClose}>Cancel</Button>
        <Button onClick={doAdd} disabled={busy}>{busy ? 'Adding…' : 'Add friend'}</Button>
      </div>
    </div>
  )
}

function EditFriendSheet({ info, onClose, onSaved }: { info: FriendInfo; onClose: () => void; onSaved: () => void }) {
  const [url, setUrl] = useState(info.url)
  const [token, setToken] = useState('')
  const [enabled, setEnabled] = useState(info.enabled)
  const [manage, setManage] = useState(!!info.manage)
  const [busy, setBusy] = useState(false)

  const save = async () => {
    setBusy(true)
    try {
      await api.apiFriendsNamePut({ name: info.name, TypesUpdateFriendRequest: { url, enabled, manage, ...(token ? { token } : {}) } })
      toast.success(`Updated ${info.name}`)
      onSaved()
      onClose()
    } catch (err) {
      toast.error((err as Error).message)
    } finally { setBusy(false) }
  }

  return (
    <Sheet open onClose={onClose} title={`Edit ${info.name}`}>
      <div className="space-y-5">
        <PreferencesGroup>
          <EntryRow title="URL" value={url} onChange={(e) => setUrl(e.target.value)} className="font-mono" />
          <EntryRow title="Token (leave blank to keep)" type="password" value={token}
            onChange={(e) => setToken(e.target.value)} placeholder="unchanged" className="font-mono" />
          <div className="flex items-center justify-between px-4 py-2.5">
            <span className="text-sm">Enabled</span>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
          </div>
          <div className="flex items-center justify-between px-4 py-2.5">
            <div className="flex flex-col">
              <span className="text-sm">Allow this friend to manage this node</span>
              <span className="text-xs text-muted-foreground">Grants its token full control of this node from its UI</span>
            </div>
            <Switch checked={manage} onCheckedChange={setManage} />
          </div>
        </PreferencesGroup>
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={save} disabled={busy}>{busy ? 'Saving…' : 'Save'}</Button>
        </div>
      </div>
    </Sheet>
  )
}

function FriendCard({ info, status, onOpen, onChanged }: {
  info: FriendInfo; status?: FriendStatus; onOpen: () => void; onChanged: () => void
}) {
  const [confirmDelete, setConfirmDelete] = useState<string[] | null>(null)
  const [editOpen, setEditOpen] = useState(false)
  const [busy, setBusy] = useState(false)

  const doSync = async () => {
    setBusy(true)
    try {
      const res = await api.apiFriendsNameSyncPost({ name: info.name })
      const failed = res.filter((r) => r.error)
      if (failed.length) failed.forEach((r) => toast.error(`${r.name}: ${r.error}`))
      else toast.success(`Synced ${info.name}`)
    } catch (err) {
      toast.error((err as Error).message)
    } finally { setBusy(false); onChanged() }
  }

  const doPair = async () => {
    setBusy(true)
    try {
      await api.apiFriendsNamePairPost({ name: info.name, TypesVerifyFriendRequest: { fingerprint: info.fingerprint } })
      toast.success(`Paired with ${info.name}: encryption established`)
      onChanged()
    } catch (err) {
      toast.error((err as Error).message)
    } finally { setBusy(false) }
  }

  const startDelete = async () => {
    setBusy(true)
    try {
      const res = await api.apiFriendsNameDelete({ name: info.name })
      if (res.deleted) { toast.success(`Removed ${info.name}`); onChanged() }
      else setConfirmDelete((res.sections ?? []).map((s) => `${s.section} ${s.key}`))
    } catch (err) {
      toast.error((err as Error).message)
      onChanged()
    } finally { setBusy(false) }
  }

  const confirmDeleteNow = async () => {
    setBusy(true)
    try {
      await api.apiFriendsNameDelete({ name: info.name, confirm: true })
      toast.success(`Removed ${info.name}`)
      setConfirmDelete(null)
      onChanged()
    } catch (err) {
      toast.error((err as Error).message)
      setConfirmDelete(null)
      onChanged()
    } finally { setBusy(false) }
  }

  return (
    <div className="rounded-lg bg-card p-4 space-y-3 shadow-[var(--card-shadow)]">
      <div className="flex items-center gap-2">
        <span className={`h-2.5 w-2.5 rounded-full shrink-0 ${statusColor(status)}`} />
        <button type="button" onClick={onOpen} className="font-semibold hover:underline">{info.name}</button>
        {info.hostname && info.hostname !== info.name && (
          <span className="text-xs text-muted-foreground font-mono">{info.hostname}</span>
        )}
        <div className="ml-auto flex items-center gap-1">
          {info.ha && <Badge variant="secondary" className="text-[10px]">HA</Badge>}
          {info.paired
            ? <Badge variant="outline" className="text-[10px]">paired</Badge>
            : <Badge variant="warning" className="text-[10px]">unverified</Badge>}
          {!info.enabled && <Badge variant="outline" className="text-[10px]">disabled</Badge>}
        </div>
      </div>

      <div className="text-xs text-muted-foreground font-mono break-all">{info.url}</div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
        {status?.reachable
          ? <span className="text-success">reachable{status.rtt_ms ? ` · ${status.rtt_ms}ms` : ''}</span>
          : <span className="text-danger">{status?.last_error || 'unreachable'}</span>}
        {status?.version && <span className="text-muted-foreground">v{status.version}</span>}
        {status && (status.identity_match
          ? <span className="inline-flex items-center gap-1 text-success"><ShieldCheck className="h-3 w-3" />verified</span>
          : <span className="inline-flex items-center gap-1 text-danger"><ShieldAlert className="h-3 w-3" />identity mismatch</span>)}
        {status?.reachable && !status.encryption && (
          <span className="inline-flex items-center gap-1 text-warning"><ShieldAlert className="h-3 w-3" />no encryption</span>
        )}
        {status?.conntrackd_running && <span className="text-info">conntrackd</span>}
      </div>

      {status?.vrrp && status.vrrp.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {status.vrrp.map((v) => (
            <Badge key={v.instance} variant="outline" className="text-[10px] font-mono">
              {v.instance}: {v.state}
            </Badge>
          ))}
        </div>
      )}

      {info.fingerprint && <div className="text-[10px] text-muted-foreground font-mono break-all">{info.fingerprint}</div>}

      <div className="flex flex-wrap gap-2 pt-1">
        {info.ha && (
          <Button variant="outline" size="sm" className="gap-1.5 h-7 text-xs" onClick={doSync} disabled={busy}>
            <Send className="h-3.5 w-3.5" />Sync
          </Button>
        )}
        {!info.paired && (
          <Button variant="outline" size="sm" className="gap-1.5 h-7 text-xs" onClick={doPair} disabled={busy}>
            <ShieldCheck className="h-3.5 w-3.5" />Verify &amp; pair
          </Button>
        )}
        <Button variant="outline" size="sm" className="gap-1.5 h-7 text-xs" onClick={() => setEditOpen(true)}>
          <Pencil className="h-3.5 w-3.5" />Edit
        </Button>
        <Button variant="ghost" size="sm" className="gap-1.5 h-7 text-xs ml-auto hover:text-destructive"
          onClick={startDelete} disabled={busy}>
          <Trash2 className="h-3.5 w-3.5" />Remove
        </Button>
      </div>

      {editOpen && <EditFriendSheet info={info} onClose={() => setEditOpen(false)} onSaved={onChanged} />}

      {confirmDelete !== null && (
        <Dialog open onClose={() => setConfirmDelete(null)} title={`Remove ${info.name}?`}
          description="Removing this friend will also delete the config sections derived from it."
          footer={<><Button variant="outline" onClick={() => setConfirmDelete(null)}>Cancel</Button><Button variant="destructive" onClick={confirmDeleteNow} disabled={busy}>Remove all</Button></>}>
          <ul className="text-sm font-mono space-y-1.5">
            {confirmDelete.map((s) => <li key={s}>· {s}</li>)}
          </ul>
        </Dialog>
      )}
    </div>
  )
}

export function FriendsPanel() {
  const { data: friends, isLoading, reload } = useFetch<FriendInfo[]>(() => api.apiFriendsGet())
  const { data: statuses, reload: reloadStatus } = useFetch<FriendStatus[]>(() => api.apiFriendsStatusGet())
  const [adding, setAdding] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [selected, setSelected] = useState<string | null>(null)
  const [lastReload, setLastReload] = useState(() => Date.now())

  useEffect(() => {
    const tick = () => { reloadStatus(); setLastReload(Date.now()) }
    const id = setInterval(tick, 10000)
    return () => clearInterval(id)
  }, [reloadStatus])

  const statusByName = new Map((statuses ?? []).map((s) => [s.name, s]))
  const reloadAll = () => { reload(); reloadStatus(); setLastReload(Date.now()) }

  const syncAll = async () => {
    setSyncing(true)
    try {
      const res = await api.apiFriendsSyncPost()
      const failed = res.filter((r) => r.error)
      if (failed.length) failed.forEach((r) => toast.error(`${r.name}: ${r.error}`))
      else toast.success('HA cluster synced')
    } catch (err) {
      toast.error((err as Error).message)
    } finally { setSyncing(false); reloadAll() }
  }

  if (selected) {
    return <FriendDetailView name={selected} onBack={() => { setSelected(null); reloadAll() }} />
  }

  if (isLoading) return <Spinner />

  const list = friends ?? []
  const hasHA = list.some((f) => f.ha)

  return (
    <div className="space-y-6">
      <PageHeader
        title="Friends"
        description="Mutually-authenticated peer routers for liveness, HA and WireGuard"
        action={
          <div className="flex items-center gap-2">
            <span className="hidden text-xs text-muted-foreground sm:inline">Updated {new Date(lastReload).toLocaleTimeString()}</span>
            <Button variant="outline" size="sm" className="gap-1.5" onClick={reloadAll}>
              <RefreshCw className="h-3.5 w-3.5" />Refresh
            </Button>
            {hasHA && (
              <Button variant="outline" size="sm" className="gap-1.5" onClick={syncAll} disabled={syncing}>
                <Send className="h-3.5 w-3.5" />{syncing ? 'Syncing…' : 'Sync HA'}
              </Button>
            )}
            {list.length > 0 && (
              <Button size="sm" className="gap-1.5" onClick={() => setAdding(true)}>
                <Plus className="h-3.5 w-3.5" />Add friend
              </Button>
            )}
          </div>
        }
      />

      {list.length === 0 ? (
        <EmptyState className="max-w-2xl mx-auto" icon={<Handshake />} title="No friends"
          message="Add a peer router to monitor it, form an HA cluster, or derive WireGuard tunnels."
          action={<Button size="sm" className="gap-1.5" onClick={() => setAdding(true)}><Plus className="h-3.5 w-3.5" />Add first friend</Button>} />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {list.map((f) => (
            <FriendCard key={f.name} info={f} status={statusByName.get(f.name)} onOpen={() => setSelected(f.name)} onChanged={reloadAll} />
          ))}
        </div>
      )}

      {adding && (
        <Sheet open onClose={() => setAdding(false)} title="Add friend">
          <AddFriendSheet onClose={() => setAdding(false)} onAdded={reloadAll} />
        </Sheet>
      )}
    </div>
  )
}
