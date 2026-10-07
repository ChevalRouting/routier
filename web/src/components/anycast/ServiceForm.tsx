import { EndpointEditor } from '@/components/anycast/EndpointEditor'
import { AnycastService, Endpoint, ServiceFormProps, emptyEndpoint } from '@/components/anycast/shared'
import { checkIPOrCIDR } from '@/lib/validate'
import { Button, EntryRow, Label, PreferencesGroup, SwitchRow, TagInput } from 'cheval-ui'
import {
  Plus
} from 'lucide-react'

export function ServiceForm({ service, onChange, onAdd, onDone }: ServiceFormProps) {
  const set = <K extends keyof AnycastService>(key: K, val: AnycastService[K]) =>
    onChange({ ...service, [key]: val })

  const updateEndpoint = (idx: number, ep: Endpoint) => {
    const next = [...(service.endpoints ?? [])]
    next[idx] = ep
    set('endpoints', next)
  }

  const deleteEndpoint = (idx: number) =>
    set('endpoints', (service.endpoints ?? []).filter((_, i) => i !== idx))

  const addEndpoint = () =>
    set('endpoints', [...(service.endpoints ?? []), emptyEndpoint()])

  return (
    <>
      <PreferencesGroup>
        <EntryRow
          title="Service Name"
          autoFocus
          value={service.name}
          onChange={(e) => set('name', e.target.value)}
          placeholder="my-service"
          className="font-mono"
        />
        <SwitchRow
          title="Active"
          checked={service.active ?? false}
          onCheckedChange={(v) => set('active', v)}
        />
      </PreferencesGroup>

      <PreferencesGroup title="Anycast IPs">
        <div className="px-4 py-3">
          <TagInput
            values={service.anycast_ips ?? []}
            onChange={(v) => set('anycast_ips', v)}
            placeholder="192.0.2.1/32"
            validate={checkIPOrCIDR}
            mono
          />
        </div>
      </PreferencesGroup>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-sm font-semibold">
            Endpoints
            <span className="ml-1 text-xs font-normal text-muted-foreground">
              ({(service.endpoints ?? []).length})
            </span>
          </Label>
          <Button type="button" variant="outline" size="sm" onClick={addEndpoint} className="h-7 gap-1 text-xs">
            <Plus className="h-3 w-3" />Add Endpoint
          </Button>
        </div>
        {(service.endpoints ?? []).length === 0 && (
          <p className="text-sm text-muted-foreground italic">No endpoints configured.</p>
        )}
        {(service.endpoints ?? []).map((ep, idx) => (
          <EndpointEditor
            key={idx}
            endpoint={ep}
            onChange={(updated) => updateEndpoint(idx, updated)}
            onDelete={() => deleteEndpoint(idx)}
          />
        ))}
      </div>

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add Service</Button>}
      </div>
    </>
  )
}
