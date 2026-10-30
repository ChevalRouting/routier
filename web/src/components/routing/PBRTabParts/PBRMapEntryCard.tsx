import { NumberRow } from '@/components/ui/NumberRow'
import { SelectRow } from '@/components/ui/SelectRow'
import type { PBRMapEntry } from '@/components/routing/types'
import { checkCIDR, checkIP } from '@/lib/validate'
import { Button, ComboRow, EntryRow, PreferencesGroup, Segmented } from 'cheval-ui'
import { Trash2 } from 'lucide-react'
import { useState } from 'react'

type PBRAction = 'group' | 'nexthop' | 'vrf'

type ActionOption = {
  value: PBRAction
  label: string
}

type PBRMapEntryCardShape = {
  entry: PBRMapEntry
  groupNames: string[]
  vrfNames: string[]
  onChange: (patch: Partial<PBRMapEntry>) => void
  onRemove: () => void
}

const ACTION_OPTIONS: ActionOption[] = [
  { value: 'group', label: 'Group' },
  { value: 'nexthop', label: 'Nexthop' },
  { value: 'vrf', label: 'VRF' },
]

function currentAction(entry: PBRMapEntry, groupNames: string[]): PBRAction {
  if (entry.set_nexthop_group) return 'group'
  if (entry.set_nexthop) return 'nexthop'
  if (entry.set_vrf) return 'vrf'
  return groupNames.length > 0 ? 'group' : 'nexthop'
}

function actionPatch(action: PBRAction, value: string): Partial<PBRMapEntry> {
  return {
    set_nexthop_group: action === 'group' ? value : '',
    set_nexthop: action === 'nexthop' ? value : '',
    set_vrf: action === 'vrf' ? value : '',
  }
}

export function PBRMapEntryCard({ entry, groupNames, vrfNames, onChange, onRemove }: PBRMapEntryCardShape) {
  const [action, setAction] = useState<PBRAction>(() => currentAction(entry, groupNames))

  const handleAction = (next: PBRAction) => {
    setAction(next)
    onChange(actionPatch(next, ''))
  }

  return (
    <PreferencesGroup
      title={`Entry ${entry.seq || ''}`.trim()}
      header={(
        <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-destructive" onClick={onRemove} aria-label="Remove Entry" title="Remove Entry">
          <Trash2 className="h-4 w-4" />
        </Button>
      )}
    >
      <NumberRow title="Seq" value={entry.seq} placeholder="10" onChange={(v) => onChange({ seq: v ?? 0 })} />
      <EntryRow title="Source" value={entry.match_src ?? ''} placeholder="10.0.0.0/8" error={checkCIDR(entry.match_src ?? '')} onChange={(e) => onChange({ match_src: e.target.value })} className="font-mono" />
      <EntryRow title="Destination" value={entry.match_dst ?? ''} placeholder="0.0.0.0/0" error={checkCIDR(entry.match_dst ?? '')} onChange={(e) => onChange({ match_dst: e.target.value })} className="font-mono" />
      <ComboRow title="Action">
        <Segmented value={action} onChange={handleAction} options={ACTION_OPTIONS} />
      </ComboRow>
      {action === 'group' && (
        <SelectRow title="Nexthop Group" value={entry.set_nexthop_group} placeholder="Select a group" options={groupNames} onChange={(v) => onChange(actionPatch('group', v ?? ''))} />
      )}
      {action === 'nexthop' && (
        <EntryRow title="Nexthop" value={entry.set_nexthop ?? ''} placeholder="192.168.1.1" error={checkIP(entry.set_nexthop ?? '')} onChange={(e) => onChange(actionPatch('nexthop', e.target.value))} className="font-mono" />
      )}
      {action === 'vrf' && (
        <SelectRow title="VRF" value={entry.set_vrf} placeholder="Select a VRF" options={['unchanged', ...vrfNames]} onChange={(v) => onChange(actionPatch('vrf', v ?? ''))} />
      )}
    </PreferencesGroup>
  )
}
