import { ChoiceRow, ComboRow, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, VLANPicker, type VLANOption } from 'cheval-ui'
import type { SwitchPortDefinition } from '@/lib/switches'

type PortVlanFieldsShape = {
  title: string
  port: SwitchPortDefinition
  vlans: VLANOption[]
  onChange: (port: SwitchPortDefinition) => void
}

const NO_NATIVE = '0'

const VLAN_MODES = [
  { value: 'access', label: 'Access' },
  { value: 'trunk', label: 'Trunk' },
]

export function PortVlanFields({ title, port, vlans, onChange }: PortVlanFieldsShape) {
  const tagged = port.tagged ?? []

  const handleMode = (mode: string) => {
    if (mode === 'access') onChange({ mode: 'access', vlan: port.vlan ?? port.native ?? vlans[0]?.vid })
    else onChange({ mode: 'trunk', native: port.vlan ?? 0, tagged })
  }

  const handleToggle = (vid: number) => onChange({ ...port, tagged: tagged.includes(vid) ? tagged.filter((id) => id !== vid) : [...tagged, vid] })

  return (
    <PreferencesGroup title={title}>
      <ChoiceRow title="VLAN Mode" value={port.mode} choices={VLAN_MODES} onChange={handleMode} help={port.mode === 'access' ? 'Untagged traffic on one VLAN' : 'Tagged VLANs, with an optional untagged native VLAN'} />
      {port.mode === 'access' && (
        <ComboRow title="VLAN">
          <Select value={port.vlan ? String(port.vlan) : ''} onValueChange={(value) => onChange({ mode: 'access', vlan: Number(value) })}>
            <SelectTrigger><SelectValue placeholder="Pick a VLAN" /></SelectTrigger>
            <SelectContent>
              {vlans.map((option) => <SelectItem key={option.vid} value={String(option.vid)}>{option.vid} {option.name}</SelectItem>)}
            </SelectContent>
          </Select>
        </ComboRow>
      )}
      {port.mode === 'trunk' && (
        <>
          <ComboRow title="Native VLAN">
            <Select value={String(port.native ?? 0)} onValueChange={(value) => onChange({ ...port, native: Number(value) || undefined })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value={NO_NATIVE}>None (tagged only)</SelectItem>
                {vlans.map((option) => <SelectItem key={option.vid} value={String(option.vid)}>{option.vid} {option.name}</SelectItem>)}
              </SelectContent>
            </Select>
          </ComboRow>
          <VLANPicker options={vlans} selected={tagged} onToggle={handleToggle} emptyLabel="Declare VLAN interfaces first" />
        </>
      )}
    </PreferencesGroup>
  )
}
