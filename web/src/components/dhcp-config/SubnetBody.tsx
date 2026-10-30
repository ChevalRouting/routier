import { VarPickerInput } from '@/components/VarPickerInput'
import { InterfaceSelect, KeaSubnet, PoolRows } from '@/components/dhcp-config/shared'
import { ComboRow, EntryRow, PreferencesGroup, TagInput } from 'cheval-ui'

type SubnetBodyShape = {
  v6: boolean
  subnet: KeaSubnet
  networks: string[]
  ifaceNames: string[]
  onChange: (s: KeaSubnet) => void
}

export function SubnetBody({ v6, subnet, networks, ifaceNames, onChange }: SubnetBodyShape) {
  const set = <K extends keyof KeaSubnet>(k: K, val: KeaSubnet[K]) => onChange({ ...subnet, [k]: val })

  return (
    <div className="space-y-6">
      <PreferencesGroup>
        <ComboRow title="Network">
          <VarPickerInput
            value={subnet.subnet}
            onChange={(v) => set('subnet', v)}
            vars={networks}
            prefix=""
            label="Configured Networks"
            placeholder={v6 ? '2001:db8::/64' : '10.0.0.0/24'}
            mono
            wrapperClassName="flex-1"
          />
        </ComboRow>
        <ComboRow title="Interface">
          <InterfaceSelect value={subnet.interface} options={ifaceNames} onChange={(v) => set('interface', v)} />
        </ComboRow>
        {!v6 && (
          <EntryRow title="Gateway" value={subnet.gateway ?? ''} onChange={(e) => set('gateway', e.target.value || undefined)} placeholder="10.0.0.1" className="font-mono" />
        )}
        <ComboRow title="DNS Servers">
          <TagInput values={subnet.dns ?? []} onChange={(v) => set('dns', v.length ? v : undefined)} placeholder={v6 ? '2001:db8::53' : '1.1.1.1'} mono />
        </ComboRow>
        <ComboRow title="Reserved for Manual Use" subtitle="Excluded from suggested addresses">
          <TagInput values={subnet.exclusions ?? []} onChange={(v) => set('exclusions', v.length ? v : undefined)} placeholder={v6 ? '2001:db8::1-2001:db8::ff' : '10.0.0.1-10.0.0.20'} mono />
        </ComboRow>
      </PreferencesGroup>
      <PoolRows v6={v6} pools={subnet.pools ?? []} onChange={(v) => set('pools', v.length ? v : undefined)} />
    </div>
  )
}
