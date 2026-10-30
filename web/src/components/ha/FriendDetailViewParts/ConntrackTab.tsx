import type {
  TypesConntrackdStatus as ConntrackdStatus,
  TypesFriendInterface as FriendInterface
} from '@/api'
import { Field, LoadingState } from '@/components/ha/FriendDetailView'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Button, Card, CardContent, CardHeader, CardTitle, ComboRow, EntryRow, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, StatusChip, SwitchRow } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

type ConntrackTabShape = { friendName: string; cd?: ConntrackdStatus }

type ExShape = { interface?: string; address?: string; port?: number; allow_inbound?: boolean }

export function ConntrackTab({ friendName, cd }: ConntrackTabShape) {
  const { data: cfg, error: cfgError, reload: reloadCfg } = useFetch(() => api.apiConfigGet() as unknown as Promise<Record<string, unknown>>)
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
    const ex = (cfg.conntrackd ?? {}) as ExShape
    if (ex.interface && ex.address) setLocalSel(`${ex.interface}|${ex.address}`)
    setPort(ex.port ? String(ex.port) : '')
    setAllowInbound(!!ex.allow_inbound)
    setInit(true)
  }, [cfg, init])

  const opts = (ifs: FriendInterface[] | null) =>
    (ifs ?? []).flatMap((i) => (i.addresses ?? []).map((a) => ({ iface: i.name, addr: a })))

  const ipSelect = (value: string, set: (v: string) => void, ifs: FriendInterface[] | null) => (
    <Select value={value} onValueChange={set}>
      <SelectTrigger><SelectValue placeholder="Select address" /></SelectTrigger>
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
      const res = await api.apiFriendsNameConntrackPost({ name: friendName, TypesConfigureConntrackRequest: {
        local_interface: li, local_address: la,
        friend_interface: fi, friend_address: fa,
        ...(port ? { port: Number(port) } : {}),
        allow_inbound: allowInbound,
      } })
      if (res.warning) {
        toast.warning(`conntrackd applied with warnings: ${res.warning}`)
      } else {
        toast.success('conntrackd configured and applied on both sides')
      }
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader><CardTitle>Conntrackd Status</CardTitle></CardHeader>
        <CardContent className="grid grid-cols-1 gap-x-6 gap-y-3 text-sm sm:grid-cols-2 md:grid-cols-3">
          <Field label="Running" value={<StatusChip intent={cd?.running ? 'success' : 'danger'} label={cd?.running ? 'Running' : 'Stopped'} />} />
          <Field label="Tracked Entries" value={cd?.entries} />
          {cd?.by_proto && Object.entries(cd.by_proto).map(([p, n]) => <Field key={p} label={`${p} entries`} value={n} />)}
        </CardContent>
      </Card>

      {!cfg ? <LoadingState error={cfgError} onRetry={() => reloadCfg(true)} /> : (
        <PreferencesGroup
          title="Configure conntrackd With This Friend"
          header={(
            <Button variant="suggested" onClick={save} disabled={saving || !localSel || !friendSel}>
              {saving ? 'Applying…' : 'Configure & Apply on Both Sides'}
            </Button>
          )}
        >
          <ComboRow title="Our Sync Address">{ipSelect(localSel, setLocalSel, selfIfaces)}</ComboRow>
          <ComboRow title="Friend Sync Address">{ipSelect(friendSel, setFriendSel, friendIfaces ?? null)}</ComboRow>
          <EntryRow title="Port" value={port} onChange={(e) => setPort(e.target.value)} className="font-mono" placeholder="3780" />
          <SwitchRow title="Allow Inbound" checked={allowInbound} onCheckedChange={setAllowInbound} />
        </PreferencesGroup>
      )}
    </div>
  )
}
