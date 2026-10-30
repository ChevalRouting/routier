import { Button, ComboRow, EntryRow, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { BOND_MODES } from '@/lib/switchPortEdit'

type BondFieldsShape = {
  name: string
  mode: string
  members: string[]
  candidates: string[]
  onName: (name: string) => void
  onMode: (mode: string) => void
  onMembers: (members: string[]) => void
}

export function BondFields({ name, mode, members, candidates, onName, onMode, onMembers }: BondFieldsShape) {
  const handleToggle = (port: string) => onMembers(members.includes(port) ? members.filter((member) => member !== port) : [...members, port])

  return (
    <PreferencesGroup title="Bond" description="Changing the members, the name or the mode recreates the bond on apply">
      <EntryRow title="Name" value={name} onChange={(e) => onName(e.target.value)} className="font-mono" />
      <ComboRow title="Mode">
        <Select value={mode} onValueChange={onMode}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            {BOND_MODES.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}
          </SelectContent>
        </Select>
      </ComboRow>
      <div className="space-y-1.5 px-4 py-3">
        <div className="text-sm">Members</div>
        <div className="flex flex-wrap gap-1.5">
          {candidates.map((port) => (
            <Button key={port} size="sm" variant={members.includes(port) ? 'default' : 'outline'} aria-pressed={members.includes(port)} className="h-7 px-2 font-mono text-xs" onClick={() => handleToggle(port)}>
              {port}
            </Button>
          ))}
        </div>
      </div>
    </PreferencesGroup>
  )
}
