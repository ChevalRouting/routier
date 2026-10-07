import type { TypesFriendInfo as FriendInfo, TypesFriendStatus as FriendStatus } from '@/api'
import { EditFriendSheet, statusColor } from '@/components/ha/FriendsPanel'
import { api } from '@/lib/client'
import { Badge, Button, Dialog } from 'cheval-ui'
import { Pencil, Send, ShieldAlert, ShieldCheck, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

type FriendCardShape = {
  info: FriendInfo; status?: FriendStatus; onOpen: () => void; onChanged: () => void
}

export function FriendCard({ info, status, onOpen, onChanged }: FriendCardShape) {
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
