import { VarPickerInput } from '@/components/VarPickerInput'
import { ACTIONS, ADDR_FAMILIES, CT_STATES, ManagedRule, PROTOCOLS, RuleMatch } from '@/components/firewall/shared'
import { ComboRow, EntryRow, PreferencesGroup, SwitchRow } from 'cheval-ui'
import { SelectRow } from '@/components/ui/SelectRow'

type RuleBodyShape = { rule: ManagedRule; onChange: (r: ManagedRule) => void; vars: string[] }

export function RuleBody({ rule, onChange, vars }: RuleBodyShape) {
  const m = rule.match ?? {}

  const setMatch = (patch: Partial<RuleMatch>) =>
    onChange({ ...rule, match: { ...m, ...patch } })

  const ifaceVars = vars.filter((v) => v.endsWith('_interfaces') || v === 'interfaces' || v === 'tunnels' || v === 'wireguard')
  const addrVars = vars.filter((v) => v.includes('_address') || v.includes('_network'))
  const familyMissing = !!(m.saddr || m.daddr) && !m.addr_family

  return (
    <div className="space-y-6">
      <PreferencesGroup title="Match">
        <SelectRow title="Protocol" value={m.protocol} onChange={(v) => setMatch({ protocol: v })} options={PROTOCOLS} noneLabel="any" />
        <ComboRow title="Input Interface">
          <VarPickerInput value={m.iif ?? ''} onChange={(v) => setMatch({ iif: v || undefined })} vars={ifaceVars} placeholder="$lan_interfaces" mono />
        </ComboRow>
        <ComboRow title="Output Interface">
          <VarPickerInput value={m.oif ?? ''} onChange={(v) => setMatch({ oif: v || undefined })} vars={ifaceVars} placeholder="$wan_interfaces" mono />
        </ComboRow>
        <ComboRow title="Source Address">
          <VarPickerInput value={m.saddr ?? ''} onChange={(v) => setMatch({ saddr: v || undefined })} vars={addrVars} placeholder="$lan_network" mono />
        </ComboRow>
        <ComboRow title="Destination Address">
          <VarPickerInput value={m.daddr ?? ''} onChange={(v) => setMatch({ daddr: v || undefined })} vars={addrVars} placeholder="192.168.0.0/24" mono />
        </ComboRow>
        <SelectRow
          title="Address Family"
          subtitle={familyMissing ? 'Required with a source or destination address' : undefined}
          value={m.addr_family}
          onChange={(v) => setMatch({ addr_family: v })}
          options={ADDR_FAMILIES}
          noneLabel="None"
        />
        <EntryRow title="Source Port" value={m.sport ?? ''} onChange={(e) => setMatch({ sport: e.target.value || undefined })} placeholder="1024-65535" className="font-mono" />
        <EntryRow title="Destination Port" value={m.dport ?? ''} onChange={(e) => setMatch({ dport: e.target.value || undefined })} placeholder="80,443" className="font-mono" />
        <SelectRow title="CT State" value={m.ct_state} onChange={(v) => setMatch({ ct_state: v })} options={CT_STATES} noneLabel="any" />
      </PreferencesGroup>

      <PreferencesGroup title="Action">
        <SelectRow title="Action" value={rule.action} onChange={(v) => onChange({ ...rule, action: v ?? rule.action })} options={ACTIONS} />
        {(rule.action === 'dnat' || rule.action === 'snat') && (
          <EntryRow title="Target Address" value={rule.action_to ?? ''} onChange={(e) => onChange({ ...rule, action_to: e.target.value || undefined })} placeholder="10.0.0.1" className="font-mono" />
        )}
        <EntryRow title="Comment" value={rule.comment ?? ''} onChange={(e) => onChange({ ...rule, comment: e.target.value || undefined })} />
        <SwitchRow title="Enabled" checked={!rule.disabled} onCheckedChange={(v) => onChange({ ...rule, disabled: v ? undefined : true })} />
      </PreferencesGroup>
    </div>
  )
}
