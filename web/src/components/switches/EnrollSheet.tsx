import { useMemo, useState } from 'react'
import { toast } from 'sonner'
import { Button, ChoiceRow, ComboRow, EntryRow, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Sheet } from 'cheval-ui'
import type { TypesSwitchCandidate as SwitchCandidate, TypesSwitchModel as SwitchModel } from '@/api'
import { api } from '@/lib/client'
import type { SwitchDefinition } from '@/lib/switches'
import { CredentialFields, credentialDefaults } from './CredentialFields'

const SCHEMES = [
  { value: 'https', label: 'HTTPS' },
  { value: 'http', label: 'HTTP' },
]

const NO_INTERFACE = '-'

type EnrollSheetShape = {
  candidate?: SwitchCandidate
  models: SwitchModel[]
  interfaces: string[]
  taken: string[]
  onEnroll: (name: string, definition: SwitchDefinition) => void
  onClose: () => void
}

function suggestedName(candidate: SwitchCandidate | undefined): string {
  if (!candidate) return ''
  return (candidate.name || candidate.chassis_id).toLowerCase().replace(/[^a-z0-9._-]+/g, '-').replace(/^-+/, '').slice(0, 32)
}

export function EnrollSheet({ candidate, models, interfaces, taken, onEnroll, onClose }: EnrollSheetShape) {
  const [modelId, setModelId] = useState(candidate?.model_id ?? '')
  const model = useMemo(() => models.find((m) => m.id === modelId), [models, modelId])
  const [name, setName] = useState(suggestedName(candidate))
  const [scheme, setScheme] = useState<'https' | 'http'>('https')
  const [host, setHost] = useState(candidate?.mgmt_ip ?? '')
  const [iface, setIface] = useState(candidate && interfaces.includes(candidate.local_iface) ? candidate.local_iface : NO_INTERFACE)
  const [uplink, setUplink] = useState(candidate?.port_id ?? '')
  const [credentials, setCredentials] = useState<Record<string, string>>(() => credentialDefaults(model?.credentials ?? []))
  const [busy, setBusy] = useState(false)

  const nameTaken = taken.includes(name)
  const ready = !!model && name.trim() !== '' && host.trim() !== '' && !nameTaken

  const handleModel = (next: string) => {
    setModelId(next)
    const fields = models.find((m) => m.id === next)?.credentials ?? []
    setCredentials((current) => ({ ...credentialDefaults(fields), ...current }))
  }

  const handleEnroll = async () => {
    setBusy(true)
    try {
      await api.apiSwitchesNameCredentialsPut({ name, TypesSwitchCredentialsRequest: { values: credentials } })
      onEnroll(name, {
        model: modelId,
        host: host.trim(),
        scheme: scheme === 'http' ? 'http' : undefined,
        tls_skip_verify: scheme === 'https' ? true : undefined,
        uplink: uplink.trim() || undefined,
        interface: iface === NO_INTERFACE ? undefined : iface,
      })
      toast.success(`${name} added to the draft, save and apply to start managing it`)
      onClose()
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Sheet open onClose={onClose} title={candidate ? `Manage ${candidate.name || candidate.chassis_id}` : 'Add a switch'}>
      <div className="space-y-5">
        <PreferencesGroup title="Switch" description={candidate?.sys_descr}>
          <ComboRow title="Model">
            <Select value={modelId} onValueChange={handleModel}>
              <SelectTrigger><SelectValue placeholder="Select a model" /></SelectTrigger>
              <SelectContent>
                {models.map((m) => <SelectItem key={m.id} value={m.id}>{m.name}</SelectItem>)}
              </SelectContent>
            </Select>
          </ComboRow>
          <EntryRow title="Name" value={name} onChange={(e) => setName(e.target.value)} error={nameTaken ? 'A switch already uses this name' : null} className="font-mono" />
          <EntryRow title="Management Address" value={host} onChange={(e) => setHost(e.target.value)} className="font-mono" />
          <ChoiceRow title="Protocol" value={scheme} choices={SCHEMES} onChange={(value) => setScheme(value as 'https' | 'http')} help="Use HTTP only when the switch has no HTTPS service, on a trusted network" />
        </PreferencesGroup>
        <PreferencesGroup title="Uplink">
          <ComboRow title="Routier Interface">
            <Select value={iface} onValueChange={setIface}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value={NO_INTERFACE}>Not set</SelectItem>
                {interfaces.map((option) => <SelectItem key={option} value={option}>{option}</SelectItem>)}
              </SelectContent>
            </Select>
          </ComboRow>
          <EntryRow title="Switch Uplink Port" value={uplink} onChange={(e) => setUplink(e.target.value)} help="Port of the switch facing this routier" className="font-mono" />
        </PreferencesGroup>
        {model && (
          <PreferencesGroup title="Credentials" description="Stored now, sealed in the routier database">
            <CredentialFields fields={model.credentials} values={credentials} onChange={(key, value) => setCredentials((current) => ({ ...current, [key]: value }))} />
          </PreferencesGroup>
        )}
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleEnroll} disabled={!ready || busy}>{busy ? 'Saving…' : 'Add to Draft'}</Button>
        </div>
      </div>
    </Sheet>
  )
}
