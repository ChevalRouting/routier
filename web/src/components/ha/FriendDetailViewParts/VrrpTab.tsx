import type {
  TypesVRRPInstanceStatus as VRRPInstanceStatus
} from '@/api'
import { Field, VrrpConfigInstance } from '@/components/ha/FriendDetailView'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Badge, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, Input, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Spinner } from 'cheval-ui'
import { useState } from 'react'
import { toast } from 'sonner'

type VrrpTabShape = { friendName: string; vrrpStatus: VRRPInstanceStatus[] }

type HaShape = { vrrp?: VrrpConfigInstance[] }

type FriendVrrpShape = { iface: string; v: VrrpConfigInstance }

type UpdatedShape = { vrrp?: VrrpConfigInstance[] }

export function VrrpTab({ friendName, vrrpStatus }: VrrpTabShape) {
  const { data: cfg, reload } = useFetch(() => api.apiConfigGet() as unknown as Promise<Record<string, unknown>>)
  const ha = (cfg?.ha ?? {}) as HaShape
  const interfaceNames = Object.keys((cfg?.interfaces ?? {}) as Record<string, unknown>)

  const friendVrrp: FriendVrrpShape[] = []
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
    const updated = JSON.parse(JSON.stringify(ha)) as UpdatedShape
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
