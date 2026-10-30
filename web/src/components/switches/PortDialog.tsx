import { useState } from 'react'
import { Button, ChoiceRow, Dialog, NoticeBanner, PreferencesGroup, type VLANOption } from 'cheval-ui'
import type { TypesSwitchState as SwitchState } from '@/api'
import type { SwitchDefinition, SwitchPortDefinition } from '@/lib/switches'
import { applyEdit, editError, initialEdit, memberCandidates, releasedBonds, type PortEdit, type PortEditContext, type PortRole, type ReleasedBond } from '@/lib/switchPortEdit'
import { BondFields } from './BondFields'
import { PortStatus } from './PortStatus'
import { PortVlanFields } from './PortVlanFields'

type PortDialogShape = {
  ports: string[]
  definition: SwitchDefinition | undefined
  state: SwitchState | undefined
  vlans: VLANOption[]
  pending: string[]
  onChange: (definition: SwitchDefinition) => void
  onClose: () => void
}

type PortEditorShape = {
  ports: string[]
  definition: SwitchDefinition
  state: SwitchState | undefined
  vlans: VLANOption[]
  onChange: (definition: SwitchDefinition) => void
  onCancel: () => void
}

const ROLES = [
  { value: 'port', label: 'Port' },
  { value: 'bond', label: 'Bond' },
  { value: 'unmanaged', label: 'Not Managed' },
]

function releaseNotice({ name, remaining, recreated }: ReleasedBond): string {
  const back = remaining.length > 0 ? `; ${remaining.join(', ')} go back to the bridge with its VLANs` : ''
  return `${name} is ${recreated ? 'recreated' : 'removed'} on apply${back}.`
}

function PortEditor({ ports, definition, state, vlans, onChange, onCancel }: PortEditorShape) {
  const context: PortEditContext = { ports, definition, state }
  const [edit, setEdit] = useState<PortEdit>(() => initialEdit(context))
  const error = editError(edit, context, vlans.length)
  const released = releasedBonds(edit, context)

  const update = (patch: Partial<PortEdit>) => setEdit((current) => ({ ...current, ...patch }))

  const handleSave = () => {
    onChange(applyEdit(edit, context))
    onCancel()
  }

  return (
    <div className="space-y-4">
      <PreferencesGroup>
        <ChoiceRow title="Role" value={edit.role} choices={ROLES} onChange={(role) => update({ role: role as PortRole })} />
      </PreferencesGroup>
      {edit.role === 'bond' && (
        <BondFields
          name={edit.bondName}
          mode={edit.bondMode}
          members={edit.members}
          candidates={memberCandidates(state)}
          onName={(bondName) => update({ bondName })}
          onMode={(bondMode) => update({ bondMode })}
          onMembers={(members) => update({ members })}
        />
      )}
      {edit.role !== 'unmanaged' && (
        <PortVlanFields title={edit.role === 'bond' ? 'Bond VLANs' : 'VLANs'} port={edit.port} vlans={vlans} onChange={(port: SwitchPortDefinition) => update({ port })} />
      )}
      {edit.role === 'unmanaged' && <p className="text-sm text-muted-foreground">The ports leave the configuration and keep their current settings on the switch</p>}
      {released.map((bond) => <NoticeBanner key={bond.name} intent="warning">{releaseNotice(bond)}</NoticeBanner>)}
      <div className="flex items-center justify-end gap-2">
        {error && <span className="mr-auto text-xs text-muted-foreground">{error}</span>}
        <Button variant="outline" onClick={onCancel}>Cancel</Button>
        <Button onClick={handleSave} disabled={!!error}>Update Draft</Button>
      </div>
    </div>
  )
}

export function PortDialog({ ports, definition, state, vlans, pending, onChange, onClose }: PortDialogShape) {
  return (
    <Dialog open onClose={onClose} title={<span className="font-mono">{ports.length === 1 ? ports[0] : `${ports.length} ports`}</span>} className="max-w-2xl">
      <div className="max-h-[70dvh] space-y-4 overflow-y-auto">
        <PortStatus ports={ports} live={state?.ports ?? []} definition={definition} pending={pending} />
        {definition ? (
          <PortEditor ports={ports} definition={definition} state={state} vlans={vlans} onChange={onChange} onCancel={onClose} />
        ) : (
          <p className="text-sm text-muted-foreground">This switch is removed in the draft; its ports cannot be configured</p>
        )}
      </div>
    </Dialog>
  )
}
