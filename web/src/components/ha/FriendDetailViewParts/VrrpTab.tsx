import type {
  TypesVRRPInstanceStatus as VRRPInstanceStatus
} from '@/api'
import { Field, LoadingState, VrrpConfigInstance } from '@/components/ha/FriendDetailView'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Button, Card, CardContent, CardHeader, CardTitle, ComboRow, EmptyState, EntryRow, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, StateChip, TagInput } from 'cheval-ui'
import { useState } from 'react'
import { toast } from 'sonner'

type VrrpTabShape = { friendName: string; vrrpStatus: VRRPInstanceStatus[] }

type HaShape = { vrrp?: VrrpConfigInstance[] }

type FriendVrrpShape = { iface: string; v: VrrpConfigInstance }

type UpdatedShape = { vrrp?: VrrpConfigInstance[] }

export function VrrpTab({ friendName, vrrpStatus }: VrrpTabShape) {
  const { data: cfg, error: cfgError, reload } = useFetch(() => api.apiConfigGet() as unknown as Promise<Record<string, unknown>>)
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
  const [priority, setPriority] = useState('')
  const [saving, setSaving] = useState(false)

  const statusFor = (id?: number) => vrrpStatus.find((s) => s.id === id)
  const ready = !!localIface && !!friendIface && !!vrid && vips.length > 0

  const addInstance = async () => {
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
      setVrid(''); setVips([]); setPriority('')
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
        <EmptyState className="py-10" title="No VRRP Instances" message="Configure VRRP below to create a failover group with this friend" />
      ) : friendVrrp.map(({ iface: ifn, v }) => {
        const s = statusFor(v.id)
        return (
          <Card key={`${ifn}-${v.id}`}>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between gap-2">
                <CardTitle className="flex items-center gap-2 text-base">
                  {v.name || `${ifn} (VRID ${v.id})`}
                  {s && <StateChip state={s.state} />}
                </CardTitle>
                <Button variant="ghost" size="sm" className="text-destructive" disabled={saving} onClick={() => removeInstance(ifn, v.id)}>Remove</Button>
              </div>
            </CardHeader>
            <CardContent className="grid grid-cols-1 gap-x-6 gap-y-3 text-sm sm:grid-cols-2 md:grid-cols-3">
              <Field label="Interface" value={ifn} mono />
              <Field label="VRID" value={v.id} />
              <Field label="Priority" value={v.priority} />
              <Field label="VIPs" value={(v.vips ?? []).join(', ')} mono />
              <Field label="Master" value={s?.master_ip} mono />
            </CardContent>
          </Card>
        )
      })}

      {!cfg ? <LoadingState error={cfgError} onRetry={() => reload(true)} /> : (
        <PreferencesGroup
          title="Configure VRRP With This Friend"
          header={(
            <Button variant="suggested" onClick={addInstance} disabled={saving || !ready}>
              {saving ? 'Applying…' : 'Configure & Apply on Both Sides'}
            </Button>
          )}
        >
          <ComboRow title="Our Interface">
            <Select value={localIface} onValueChange={setLocalIface}>
              <SelectTrigger><SelectValue placeholder="Select interface" /></SelectTrigger>
              <SelectContent>
                {interfaceNames.map((k) => <SelectItem key={k} value={k} className="font-mono">{k}</SelectItem>)}
              </SelectContent>
            </Select>
          </ComboRow>
          <ComboRow title="Friend Interface">
            <Select value={friendIface} onValueChange={setFriendIface}>
              <SelectTrigger><SelectValue placeholder="Select interface" /></SelectTrigger>
              <SelectContent>
                {(friendIfaces ?? []).map((i) => <SelectItem key={i.name} value={i.name} className="font-mono">{i.name}</SelectItem>)}
              </SelectContent>
            </Select>
          </ComboRow>
          <EntryRow title="VRID" value={vrid} onChange={(e) => setVrid(e.target.value)} className="font-mono" placeholder="51" />
          <EntryRow title="Priority" value={priority} onChange={(e) => setPriority(e.target.value)} className="font-mono" placeholder="150" />
          <ComboRow title="Virtual IPs">
            <TagInput values={vips} onChange={setVips} placeholder="10.0.0.1/24" mono />
          </ComboRow>
        </PreferencesGroup>
      )}
    </div>
  )
}
