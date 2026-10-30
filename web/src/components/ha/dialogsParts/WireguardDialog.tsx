import type { TypesFriendInfo as FriendInfo, TypesFriendInterface as FriendInterface } from '@/api'
import { AddressSelect } from '@/components/ha/dialogs'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Button, ComboRow, EntryRow, PreferencesGroup, Sheet } from 'cheval-ui'
import { useState } from 'react'
import { toast } from 'sonner'

type WireguardDialogShape = { friend: FriendInfo; onClose: () => void; onDone: () => void }

export function WireguardDialog({ friend, onClose, onDone }: WireguardDialogShape) {
  const { data: remote } = useFetch<FriendInterface[]>(() => api.apiFriendsNameInterfacesGet({ name: friend.name }))
  const { data: local } = useFetch<FriendInterface[]>(() => api.apiFriendsInterfacesGet())
  const [subnet, setSubnet] = useState('169.254.50.0/31')
  const [localSel, setLocalSel] = useState('')
  const [friendSel, setFriendSel] = useState('')
  const [busy, setBusy] = useState(false)

  const derive = async () => {
    const [localIface, localAddr] = localSel.split('|')
    const [friendIface, friendAddr] = friendSel.split('|')
    setBusy(true)
    try {
      const res = await api.apiFriendsNameWireguardPost({ name: friend.name, TypesDeriveWireguardRequest: {
        subnet,
        local_endpoint: { interface: localIface, address: localAddr || undefined },
        friend_endpoint: { interface: friendIface, address: friendAddr || undefined },
      } })
      if (res.push_error) toast.warning(`${res.interface} created; counterpart push failed: ${res.push_error}`)
      else toast.success(`Created ${res.interface} and pushed counterpart`)
      onDone()
      onClose()
    } catch (err) {
      toast.error((err as Error).message)
    } finally { setBusy(false) }
  }

  const ready = !!subnet && !!localSel && !!friendSel

  return (
    <Sheet open onClose={onClose} title={`Create WireGuard to ${friend.name}`}>
      <PreferencesGroup description="Fresh keys, no routes installed">
        <EntryRow title="Tunnel Subnet" value={subnet} onChange={(e) => setSubnet(e.target.value)} className="font-mono" placeholder="169.254.50.0/31" />
        <ComboRow title="Local Endpoint">
          <AddressSelect value={localSel} onChange={setLocalSel} ifs={local} placeholder="Select address" />
        </ComboRow>
        <ComboRow title="Friend Endpoint">
          <AddressSelect value={friendSel} onChange={setFriendSel} ifs={remote} placeholder="Select address" />
        </ComboRow>
      </PreferencesGroup>
      <div className="mt-6 flex justify-end gap-2">
        <Button variant="outline" onClick={onClose}>Cancel</Button>
        <Button variant="suggested" onClick={derive} disabled={busy || !ready}>{busy ? 'Deriving…' : 'Derive Tunnel'}</Button>
      </div>
    </Sheet>
  )
}
