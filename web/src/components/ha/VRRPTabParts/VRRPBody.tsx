import { VRRPInstance } from '@/components/ha/types'
import { VRRPPrependGroup } from '@/components/ha/VRRPTabParts/VRRPPrependGroup'
import { checkIPOrCIDR } from '@/lib/validate'
import { ComboRow, EntryRow, PreferencesGroup, SwitchRow, TagInput } from 'cheval-ui'
import { SelectRow } from '@/components/ui/SelectRow'

type VRRPBodyShape = {
  v: VRRPInstance
  ifaceNames: string[]
  onChange: (v: VRRPInstance) => void
}

export function VRRPBody({ v, ifaceNames, onChange }: VRRPBodyShape) {
  const set = <K extends keyof VRRPInstance>(key: K, val: VRRPInstance[K]) => onChange({ ...v, [key]: val })

  return (
    <div className="space-y-6">
      <PreferencesGroup description={v.friend ? `Derived from friend ${v.friend}; edits may be overwritten on the next sync` : undefined}>
        <EntryRow title="Name" value={v.name ?? ''} onChange={(e) => set('name', e.target.value)} placeholder="vrrp-gateway" />
        <SelectRow title="Interface" subtitle="Hosts the virtual IPs" value={v.interface} onChange={(val) => set('interface', val ?? '')} options={ifaceNames} placeholder="Select interface" />
        <EntryRow title="VRRP ID" value={v.id || ''} onChange={(e) => set('id', Number(e.target.value) || 0)} placeholder="1" className="font-mono" />
        <EntryRow title="Priority" value={v.priority || ''} onChange={(e) => set('priority', Number(e.target.value) || 0)} placeholder="100" className="font-mono" />
        <SelectRow title="Transport" value={v.transport} onChange={(val) => set('transport', val)} options={ifaceNames} noneLabel="Same as interface" />
        <ComboRow title="Virtual IPs">
          <TagInput values={v.vips ?? []} onChange={(val) => set('vips', val)} placeholder="10.0.0.1/24" mono validate={checkIPOrCIDR} />
        </ComboRow>
        <EntryRow title="Password" type="password" value={v.password ?? ''} onChange={(e) => set('password', e.target.value)} />
        <SwitchRow title="Allow Inbound" subtitle="Opens the VRRP protocol on this interface in the firewall" checked={!!v.allow_inbound} onCheckedChange={(val) => set('allow_inbound', val)} />
      </PreferencesGroup>

      <VRRPPrependGroup prepend={v.bgp_prepend} onChange={(val) => set('bgp_prepend', val)} />
    </div>
  )
}
