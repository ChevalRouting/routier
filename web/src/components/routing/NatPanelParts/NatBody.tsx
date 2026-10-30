import type { NatSpec } from '@/api'
import { VarPickerInput } from '@/components/VarPickerInput'
import { ComboRow, EntryRow, PreferencesGroup, Segmented } from 'cheval-ui'

type NatBodyShape = {
  spec: NatSpec
  ifaceVars: string[]
  addrVars: string[]
  onChange: (patch: Partial<NatSpec>) => void
}

const FAMILIES = [{ value: 'ip', label: 'ip' }, { value: 'ip6', label: 'ip6' }]

const PROTOCOLS = [{ value: 'tcp', label: 'tcp' }, { value: 'udp', label: 'udp' }]

export function NatBody({ spec, ifaceVars, addrVars, onChange }: NatBodyShape) {
  const source = spec.kind === 'masquerade' || spec.kind === 'snat'

  return (
    <PreferencesGroup>
      {source && (
        <ComboRow title="Out Interface">
          <VarPickerInput value={spec.out ?? ''} onChange={(v) => onChange({ out: v })} vars={ifaceVars} placeholder="$wan_interfaces" mono />
        </ComboRow>
      )}
      {spec.kind === 'snat' && (
        <EntryRow title="SNAT To" value={spec.to ?? ''} onChange={(e) => onChange({ to: e.target.value })} placeholder="203.0.113.5" className="font-mono" />
      )}
      {source && (
        <ComboRow title="Source">
          <VarPickerInput value={spec.source ?? ''} onChange={(v) => onChange({ source: v })} vars={addrVars} placeholder="$lan_network" mono />
        </ComboRow>
      )}
      {source && spec.source && (
        <ComboRow title="Source Family">
          <Segmented value={spec.family || 'ip'} onChange={(v) => onChange({ family: v })} options={FAMILIES} />
        </ComboRow>
      )}
      {spec.kind === 'dnat' && (
        <>
          <ComboRow title="In Interface">
            <VarPickerInput value={spec.in ?? ''} onChange={(v) => onChange({ in: v })} vars={ifaceVars} placeholder="$wan_interfaces" mono />
          </ComboRow>
          <ComboRow title="Protocol">
            <Segmented value={spec.proto || 'tcp'} onChange={(v) => onChange({ proto: v })} options={PROTOCOLS} />
          </ComboRow>
          <EntryRow title="Destination Port" value={spec.dport ?? ''} onChange={(e) => onChange({ dport: e.target.value })} placeholder="443" className="font-mono" />
          <EntryRow title="Forward To" value={spec.to ?? ''} onChange={(e) => onChange({ to: e.target.value })} placeholder="10.0.0.5:8443" className="font-mono" />
        </>
      )}
      <EntryRow title="Comment" value={spec.comment ?? ''} onChange={(e) => onChange({ comment: e.target.value })} />
    </PreferencesGroup>
  )
}
