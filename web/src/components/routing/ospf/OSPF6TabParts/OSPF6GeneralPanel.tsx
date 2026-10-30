import { ExtraDirectives } from '@/components/ExtraDirectives'
import { OSPF6Config } from '@/components/routing/types'
import { ComboRow, EntryRow, NumberInput, PreferencesGroup, SwitchRow, TagInput } from 'cheval-ui'

type OSPF6GeneralPanelShape = {
  ospf6: OSPF6Config; setOSPF6: (v: OSPF6Config) => void; onDirty: () => void
}

export function OSPF6GeneralPanel({ ospf6, setOSPF6, onDirty }: OSPF6GeneralPanelShape) {
  const upd = (patch: Partial<OSPF6Config>) => { setOSPF6({ ...ospf6, ...patch }); onDirty() }
  return (
    <div className="space-y-6">
      <PreferencesGroup>
        <EntryRow title="Router ID" value={ospf6.router_id} onChange={(e) => upd({ router_id: e.target.value })} placeholder="10.0.0.1" className="font-mono" />
        <SwitchRow
          title="Default Information Originate"
          subtitle="Advertise a default route into the OSPFv3 domain"
          checked={!!ospf6.default_information_originate}
          onCheckedChange={(v) => upd({ default_information_originate: v })}
        />
        <ComboRow title="Reference Bandwidth (Mbps)">
          <NumberInput value={ospf6.reference_bandwidth || undefined} onChange={(v) => upd({ reference_bandwidth: v })} placeholder="auto-cost" className="font-mono" />
        </ComboRow>
        <ComboRow title="Administrative Distance">
          <NumberInput value={ospf6.distance || undefined} onChange={(v) => upd({ distance: v })} placeholder="110" className="font-mono" />
        </ComboRow>
        <ComboRow title="Passive Interfaces">
          <TagInput values={ospf6.passive_interfaces} onChange={(v) => upd({ passive_interfaces: v })} placeholder="eth0" mono />
        </ComboRow>
        <ComboRow title="Redistribute">
          <TagInput values={ospf6.redistribute} onChange={(v) => upd({ redistribute: v })} placeholder="connected" />
        </ComboRow>
      </PreferencesGroup>
      <ExtraDirectives label="Additional Directives (router ospf6)" values={ospf6.extra} onChange={(v) => upd({ extra: v })} placeholder="auto-cost reference-bandwidth 100000" />
    </div>
  )
}
