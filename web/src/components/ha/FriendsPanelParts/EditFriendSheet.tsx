import type { TypesFriendInfo as FriendInfo } from '@/api'
import { api } from '@/lib/client'
import { Button, EntryRow, PreferencesGroup, Sheet, Switch } from 'cheval-ui'
import { useState } from 'react'
import { toast } from 'sonner'

type EditFriendSheetShape = { info: FriendInfo; onClose: () => void; onSaved: () => void }

export function EditFriendSheet({ info, onClose, onSaved }: EditFriendSheetShape) {
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
