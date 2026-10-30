import { ExtraDirectives } from '@/components/ExtraDirectives'
import { OSPFConfig } from '@/components/routing/types'
import { ComboRow, EntryRow, NumberInput, PreferencesGroup, SwitchRow, TagInput } from 'cheval-ui'

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
    <div className="space-y-6">
      <PreferencesGroup>
        <EntryRow title="Router ID" value={ospf.router_id} onChange={(e) => upd({ router_id: e.target.value })} placeholder="10.0.0.1" className="font-mono" />
        <SwitchRow
          title="Default Information Originate"
          subtitle="Advertise 0.0.0.0/0 into the OSPF domain"
          checked={!!ospf.default_information_originate}
          onCheckedChange={(v) => upd({ default_information_originate: v })}
        />
        <ComboRow title="Reference Bandwidth (Mbps)">
          <NumberInput value={ospf.reference_bandwidth || undefined} onChange={(v) => upd({ reference_bandwidth: v })} placeholder="auto-cost" className="font-mono" />
        </ComboRow>
        <ComboRow title="Administrative Distance">
          <NumberInput value={ospf.distance || undefined} onChange={(v) => upd({ distance: v })} placeholder="110" className="font-mono" />
        </ComboRow>
        <ComboRow title="Passive Interfaces">
          <TagInput values={ospf.passive_interfaces} onChange={(v) => upd({ passive_interfaces: v })} placeholder="eth0" mono />
        </ComboRow>
        <ComboRow title="Redistribute">
          <TagInput values={ospf.redistribute} onChange={(v) => upd({ redistribute: v })} placeholder="connected" />
        </ComboRow>
      </PreferencesGroup>
      <ExtraDirectives label="Additional Directives (router ospf)" values={ospf.extra} onChange={(v) => upd({ extra: v })} placeholder="auto-cost reference-bandwidth 100000" />
    </div>
  )
}
