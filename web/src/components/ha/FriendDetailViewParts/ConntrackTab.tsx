import type {
  TypesConntrackdStatus as ConntrackdStatus,
  TypesFriendInterface as FriendInterface
} from '@/api'
import { Field } from '@/components/ha/FriendDetailView'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Badge, Button, Card, CardContent, CardHeader, CardTitle, Input, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Spinner } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

type ConntrackTabShape = { friendName: string; cd?: ConntrackdStatus }

type ExShape = { interface?: string; address?: string; port?: number; allow_inbound?: boolean }

export function ConntrackTab({ friendName, cd }: ConntrackTabShape) {
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
