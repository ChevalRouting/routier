import type { TypesFriendInfo as FriendInfo } from '@/api'
import { api } from '@/lib/client'
import { Button, EntryRow, PreferencesGroup, Switch } from 'cheval-ui'
import { useState } from 'react'
import { toast } from 'sonner'

type AddFriendSheetShape = { onClose: () => void; onAdded: () => void }

export function AddFriendSheet({ onClose, onAdded }: AddFriendSheetShape) {
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
