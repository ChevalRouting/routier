import { Input } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Switch } from 'cheval-ui'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { TagInput } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { AccordionList } from '@/components/ui/AccordionList'
import { checkIPOrCIDR } from '@/lib/validate'
import { VRRPInstance } from '@/components/ha/types'

interface Row {
  uid: string
  v: VRRPInstance
}

function emptyInstance(iface: string): VRRPInstance {
  return { name: '', id: 0, interface: iface, vips: [], priority: 100, password: '' }
}

function VRRPSummary({ v }: { v: VRRPInstance }) {
  return (
    <div className="flex min-w-0 flex-1 items-center gap-3">
      <span className="w-40 shrink-0 truncate text-sm font-medium">{v.name || `VRID ${v.id}`}</span>
      <span className="w-24 shrink-0 font-mono text-xs text-muted-foreground">{v.interface}</span>
      <span className="w-16 shrink-0 text-xs text-muted-foreground">VRID {v.id}</span>
      <span className="w-16 shrink-0 text-xs text-muted-foreground">prio {v.priority}</span>
      <div className="flex min-w-0 flex-1 flex-wrap gap-1">
        {(v.vips ?? []).map((vip) => (
          <span key={vip} className="inline-block rounded bg-muted px-1.5 py-0.5 font-mono text-[11px]">{vip}</span>
        ))}
        {v.friend && <span className="text-[11px] text-muted-foreground">from {v.friend}</span>}
      </div>
    </div>
  )
}

function VRRPBody({ v, ifaceNames, onChange }: {
  v: VRRPInstance
  ifaceNames: string[]
  onChange: (v: VRRPInstance) => void
}) {
  const set = <K extends keyof VRRPInstance>(key: K, val: VRRPInstance[K]) => onChange({ ...v, [key]: val })

  return (
    <>
      {v.friend && (
        <p className="rounded-md bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          Derived from friend {v.friend}. Editing it here may be overwritten on the next sync.
        </p>
      )}

      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label className="text-xs">Name (label for this VRID)</Label>
          <Input value={v.name ?? ''} className="h-8 text-sm"
            onChange={(e) => set('name', e.target.value)} placeholder="vrrp-gateway" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs">Interface (hosts the VIPs)</Label>
          <Select value={v.interface} onValueChange={(val) => set('interface', val)}>
            <SelectTrigger className="h-8 text-xs font-mono"><SelectValue /></SelectTrigger>
            <SelectContent>
              {ifaceNames.map((n) => <SelectItem key={n} value={n} className="font-mono text-xs">{n}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <div className="space-y-1.5">
          <Label className="text-xs">VRRP ID</Label>
          <Input value={v.id || ''} onChange={(e) => set('id', Number(e.target.value) || 0)} placeholder="1" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs">Priority</Label>
          <Input value={v.priority || ''} onChange={(e) => set('priority', Number(e.target.value) || 0)} placeholder="100" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs">Transport (optional)</Label>
          <Select value={v.transport ?? ''} onValueChange={(val) => set('transport', val || undefined)}>
            <SelectTrigger className="h-8 text-xs font-mono"><SelectValue placeholder="same as interface" /></SelectTrigger>
            <SelectContent>
              {ifaceNames.map((n) => <SelectItem key={n} value={n} className="font-mono text-xs">{n}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="space-y-1.5">
        <Label className="text-xs">Virtual IPs</Label>
        <TagInput values={v.vips ?? []} onChange={(val) => set('vips', val)} placeholder="10.0.0.1/24" mono validate={checkIPOrCIDR} />
      </div>

      <div className="space-y-1.5">
        <Label className="text-xs">Password</Label>
        <Input type="password" value={v.password ?? ''}
          onChange={(e) => set('password', e.target.value)} placeholder="secret" />
      </div>

      <div className="flex items-center gap-2">
        <Switch checked={!!v.switchover} onCheckedChange={(val) => set('switchover', val)} />
        <span className="text-sm font-medium">Switchover on BACKUP</span>
        <span className="text-xs text-muted-foreground">(brings down WireGuard when entering BACKUP state)</span>
      </div>
      <div className="flex items-center gap-2">
        <Switch checked={!!v.allow_inbound} onCheckedChange={(val) => set('allow_inbound', val)} />
        <span className="text-sm font-medium">Allow inbound</span>
        <span className="text-xs text-muted-foreground">(opens VRRP protocol on this interface in the firewall)</span>
      </div>
    </>
  )
}

export function VRRPTab({ instances, setInstances, ifaceNames, onDirty }: {
  instances: VRRPInstance[]
  setInstances: (v: VRRPInstance[]) => void
  ifaceNames: string[]
  onDirty: () => void
}) {
  const rows: Row[] = instances.map((v) => ({ uid: `${v.interface}-${v.id}-${v.name ?? ''}`, v }))

  const update = (next: VRRPInstance[]) => { setInstances(next); onDirty() }

  const setInstance = (index: number, v: VRRPInstance) =>
    update(instances.map((it, i) => (i === index ? v : it)))

  const removeInstance = (index: number) => update(instances.filter((_, i) => i !== index))

  const addInstance = () => update([...instances, emptyInstance(ifaceNames[0] ?? '')])

  if (ifaceNames.length === 0) {
    return (
      <EmptyState className="py-12" title="No interfaces"
        message="Add an interface first, then define VRRP failover groups here." />
    )
  }

  return (
    <AccordionList
      items={rows.map((r, i) => ({ ...r, index: i }))}
      getId={(it) => `${it.uid}-${it.index}`}
      description="Virtual router failover groups managed by keepalived."
      addLabel="Add instance"
      onAdd={addInstance}
      onRemove={(it) => removeInstance(it.index)}
      emptyTitle="No VRRP instances"
      emptyMessage="Add an instance to create a failover group on an interface."
      renderSummary={(it) => <VRRPSummary v={it.v} />}
      renderBody={(it) => (
        <VRRPBody
          v={it.v}
          ifaceNames={ifaceNames}
          onChange={(v) => setInstance(it.index, v)}
        />
      )}
    />
  )
}
