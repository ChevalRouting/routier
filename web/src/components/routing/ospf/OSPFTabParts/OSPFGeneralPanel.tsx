import { ExtraDirectives } from '@/components/ExtraDirectives'
import { OSPFConfig } from '@/components/routing/types'
import { Input, Label, NumberInput, Separator, Switch, TagInput } from 'cheval-ui'

type OSPFGeneralPanelShape = {
  ospf: OSPFConfig
  setOSPF: (v: OSPFConfig) => void
  onDirty: () => void
}

export function OSPFGeneralPanel({
  ospf, setOSPF, onDirty,
}: OSPFGeneralPanelShape) {
  const upd = (patch: Partial<OSPFConfig>) => { setOSPF({ ...ospf, ...patch }); onDirty() }
  return (
    <div className="space-y-6 max-w-lg">
      <div className="space-y-1.5">
        <Label htmlFor="ospf-rid" className="text-xs">Router ID</Label>
        <Input id="ospf-rid" value={ospf.router_id} onChange={(e) => upd({ router_id: e.target.value })} placeholder="10.0.0.1" className="font-mono" />
      </div>
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-medium">Default information originate</div>
          <div className="text-xs text-muted-foreground">Advertise a default route (0.0.0.0/0) into the OSPF domain</div>
        </div>
        <Switch checked={!!ospf.default_information_originate} onCheckedChange={(v) => upd({ default_information_originate: v })} />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-1.5">
          <Label className="text-xs">Reference bandwidth (Mbps)</Label>
          <NumberInput value={ospf.reference_bandwidth || undefined} onChange={(v) => upd({ reference_bandwidth: v })} placeholder="auto-cost" className="font-mono" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs">Administrative distance</Label>
          <NumberInput value={ospf.distance || undefined} onChange={(v) => upd({ distance: v })} placeholder="110" className="font-mono" />
        </div>
      </div>
      <Separator />
      <div className="space-y-1.5">
        <Label className="text-xs font-semibold">Passive interfaces</Label>
        <TagInput values={ospf.passive_interfaces} onChange={(v) => upd({ passive_interfaces: v })} placeholder="eth0" mono />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-semibold">Redistribute</Label>
        <TagInput values={ospf.redistribute} onChange={(v) => upd({ redistribute: v })} placeholder="connected" />
      </div>
      <Separator />
      <ExtraDirectives label="Additional router ospf directives" values={ospf.extra} onChange={(v) => upd({ extra: v })} placeholder="auto-cost reference-bandwidth 100000" />
    </div>
  )
}
