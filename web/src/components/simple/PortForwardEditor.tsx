import { KnownDeviceSelect, useKnownDevices } from '@/components/KnownDeviceSelect'
import type { PortForward } from '@/components/simple/types'
import {
  AccordionList, Badge, Button, ComboRow, EntryRow, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from 'cheval-ui'
import { Trash2 } from 'lucide-react'

type PortForwardFieldsShape = {
  forward: PortForward
  devices: ReturnType<typeof useKnownDevices>
  onChange: (patch: Partial<PortForward>) => void
}

interface PortForwardEditorProps {
  forwards: PortForward[]
  onChange: (forwards: PortForward[]) => void
}

export function PortForwardEditor({ forwards, onChange }: PortForwardEditorProps) {
  const devices = useKnownDevices()
  const update = (forward: PortForward, patch: Partial<PortForward>) => {
    onChange(forwards.map((current) => current === forward ? { ...current, ...patch } : current))
  }

  const add = () => onChange([...forwards, newForward(forwards)])
  const remove = (forward: PortForward) => onChange(forwards.filter((current) => current !== forward))

  return (
    <AccordionList
      items={forwards}
      getId={(forward, index) => `${forward.id || 'new'}-${index}`}
      description={`${forwards.length} port ${forwards.length === 1 ? 'forward' : 'forwards'}`}
      addLabel="Add port forward"
      onAdd={add}
      emptyTitle="No port forwards"
      emptyMessage="Forward an incoming port from the internet to a device on your network."
      renderSummary={(forward) => (
        <div className="flex items-center gap-2">
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-medium">{forward.name || 'Port forward'}</div>
            <div className="truncate font-mono text-xs text-muted-foreground">
              {forward.proto.toUpperCase()} {forward.port || '?'} → {forward.to_host || '?'}{forward.to_port ? `:${forward.to_port}` : ''}
            </div>
          </div>
          {!forward.editable && <Badge variant="outline">Advanced</Badge>}
        </div>
      )}
      renderActions={(forward) => forward.editable && (
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive"
          title="Delete port forward"
          onClick={() => remove(forward)}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      )}
      renderBody={(forward) => (
        <PortForwardFields forward={forward} devices={devices} onChange={(patch) => update(forward, patch)} />
      )}
    />
  )
}

function PortForwardFields({ forward, devices, onChange }: PortForwardFieldsShape) {
  const disabled = !forward.editable

  return (
    <div className="space-y-3">
      {forward.issue && <p className="text-sm text-warning">{forward.issue}</p>}
      <PreferencesGroup>
        <EntryRow title="Name" value={forward.name ?? ''} disabled={disabled} placeholder="Web server" onChange={(event) => onChange({ name: event.target.value })} />
        <ComboRow title="Protocol">
          <Select value={forward.proto || 'tcp'} disabled={disabled} onValueChange={(proto) => onChange({ proto })}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="tcp+udp">TCP + UDP</SelectItem>
              <SelectItem value="tcp">TCP</SelectItem>
              <SelectItem value="udp">UDP</SelectItem>
            </SelectContent>
          </Select>
        </ComboRow>
        <EntryRow title="Incoming port" value={forward.port} disabled={disabled} placeholder="443" className="font-mono" onChange={(event) => onChange({ port: event.target.value })} />
        <ComboRow title="Destination device" subtitle="Choose a known DHCP or neighboring device, or enter an address.">
          <KnownDeviceSelect value={forward.to_host} devices={devices} disabled={disabled} onChange={(to_host) => onChange({ to_host })} />
        </ComboRow>
        <EntryRow title="Destination port" value={forward.to_port ?? ''} disabled={disabled} placeholder="Same as incoming" className="font-mono" onChange={(event) => onChange({ to_port: event.target.value || undefined })} />
      </PreferencesGroup>
    </div>
  )
}

function newForward(forwards: PortForward[]): PortForward {
  const used = new Set(forwards.map((forward) => forward.name))
  let suffix = 1
  while (used.has(`Forward ${suffix}`)) suffix++
  return {
    id: '',
    name: `Forward ${suffix}`,
    proto: 'tcp',
    port: '',
    to_host: '',
    to_port: '',
    editable: true,
  }
}
