import type { TypesFriendInfo as FriendInfo, TypesFriendInterface as FriendInterface } from '@/api'
import { AddressSelect } from '@/components/ha/dialogs'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Button, Dialog, Input } from 'cheval-ui'
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
    if (!subnet || !localSel || !friendSel) { toast.error('Subnet and both endpoint IPs are required'); return }
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

  return (
    <Dialog open onClose={onClose} title={`Create WireGuard to ${friend.name}`}
      description="Link-only tunnel with fresh keys; installs no routes."
      footer={<><Button variant="outline" onClick={onClose}>Cancel</Button><Button onClick={derive} disabled={busy}>{busy ? 'Deriving…' : 'Derive tunnel'}</Button></>}>
      <div className="space-y-4">
        <div className="space-y-1.5">
          <label className="text-xs font-medium text-muted-foreground">Tunnel subnet</label>
          <Input value={subnet} onChange={(e) => setSubnet(e.target.value)} className="font-mono text-sm" placeholder="169.254.50.0/31" />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Local endpoint IP</label>
            <AddressSelect value={localSel} onChange={setLocalSel} ifs={local} placeholder="select IP" />
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Friend endpoint IP</label>
            <AddressSelect value={friendSel} onChange={setFriendSel} ifs={remote} placeholder="select IP" />
          </div>
        </div>
      </div>
    </Dialog>
  )
}
