import { ExtraDirectives } from '@/components/ExtraDirectives'
import { OSPF6Config } from '@/components/routing/types'
import { Input, Label, NumberInput, Separator, Switch, TagInput } from 'cheval-ui'

type OSPF6GeneralPanelShape = {
  ospf6: OSPF6Config; setOSPF6: (v: OSPF6Config) => void; onDirty: () => void
}

export function OSPF6GeneralPanel({ ospf6, setOSPF6, onDirty }: OSPF6GeneralPanelShape) {
  const upd = (patch: Partial<OSPF6Config>) => { setOSPF6({ ...ospf6, ...patch }); onDirty() }
  return (
    <div className="space-y-6 max-w-lg">
      <div className="space-y-1.5">
        <Label htmlFor="ospf6-rid" className="text-xs">Router ID</Label>
        <Input id="ospf6-rid" value={ospf6.router_id} onChange={(e) => upd({ router_id: e.target.value })} placeholder="10.0.0.1" className="font-mono" />
      </div>
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-medium">Default information originate</div>
          <div className="text-xs text-muted-foreground">Advertise a default route into the OSPFv3 domain</div>
        </div>
        <Switch checked={!!ospf6.default_information_originate} onCheckedChange={(v) => upd({ default_information_originate: v })} />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-1.5">
          <Label className="text-xs">Reference bandwidth (Mbps)</Label>
          <NumberInput value={ospf6.reference_bandwidth || undefined} onChange={(v) => upd({ reference_bandwidth: v })} placeholder="auto-cost" className="font-mono" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs">Administrative distance</Label>
          <NumberInput value={ospf6.distance || undefined} onChange={(v) => upd({ distance: v })} placeholder="110" className="font-mono" />
        </div>
      </div>
      <Separator />
      <div className="space-y-1.5">
        <Label className="text-xs font-semibold">Passive interfaces</Label>
        <TagInput values={ospf6.passive_interfaces} onChange={(v) => upd({ passive_interfaces: v })} placeholder="eth0" mono />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-semibold">Redistribute</Label>
        <TagInput values={ospf6.redistribute} onChange={(v) => upd({ redistribute: v })} placeholder="connected" />
      </div>
      <Separator />
      <ExtraDirectives label="Additional router ospf6 directives" values={ospf6.extra} onChange={(v) => upd({ extra: v })} placeholder="auto-cost reference-bandwidth 100000" />
    </div>
  )
}
