import { VRRPInstance } from '@/components/ha/types'
import { checkIPOrCIDR } from '@/lib/validate'
import { Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Switch, TagInput } from 'cheval-ui'

type VRRPBodyShape = {
  v: VRRPInstance
  ifaceNames: string[]
  onChange: (v: VRRPInstance) => void
}

export function VRRPBody({ v, ifaceNames, onChange }: VRRPBodyShape) {
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
