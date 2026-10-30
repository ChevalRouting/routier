import { useState } from 'react'
import { toast } from 'sonner'
import { Button, ChoiceRow, EntryRow, IntegerEntryRow, PreferencesGroup, Sheet, SwitchRow } from 'cheval-ui'
import type { TypesSwitchModel as SwitchModel } from '@/api'
import { api } from '@/lib/client'
import type { SwitchDefinition } from '@/lib/switches'
import { CredentialFields, credentialDefaults } from './CredentialFields'

const SCHEMES = [
  { value: 'https', label: 'HTTPS' },
  { value: 'http', label: 'HTTP' },
]

type SwitchSettingsSheetShape = {
  name: string
  definition: SwitchDefinition
  model: SwitchModel | undefined
  onChange: (definition: SwitchDefinition) => void
  onCredentialsSaved: () => void
  onClose: () => void
}

export function SwitchSettingsSheet({ name, definition, model, onChange, onCredentialsSaved, onClose }: SwitchSettingsSheetShape) {
  const [host, setHost] = useState(definition.host)
  const [scheme, setScheme] = useState<'https' | 'http'>(definition.scheme ?? 'https')
  const defaultPort = scheme === 'http' ? 80 : 443
  const [port, setPort] = useState(definition.port ?? defaultPort)
  const [skipVerify, setSkipVerify] = useState(!!definition.tls_skip_verify)
  const [iface, setIface] = useState(definition.interface ?? '')
  const [uplink, setUplink] = useState(definition.uplink ?? '')
  const [credentials, setCredentials] = useState<Record<string, string>>(() => credentialDefaults(model?.credentials ?? []))
  const [credentialsEdited, setCredentialsEdited] = useState(false)
  const [busy, setBusy] = useState(false)

  const handleScheme = (next: 'https' | 'http') => {
    if (port === defaultPort) setPort(next === 'http' ? 80 : 443)
    setScheme(next)
  }

  const handleCredential = (key: string, value: string) => {
    setCredentials((current) => ({ ...current, [key]: value }))
    setCredentialsEdited(true)
  }

  const storeCredentials = async () => {
    setBusy(true)
    try {
      await api.apiSwitchesNameCredentialsPut({ name, TypesSwitchCredentialsRequest: { values: credentials } })
      toast.success(`Credentials of ${name} stored`)
      onCredentialsSaved()
      return true
    } catch (err) {
      toast.error((err as Error).message)
      return false
    } finally {
      setBusy(false)
    }
  }

  const handleSave = async () => {
    if (credentialsEdited && !(await storeCredentials())) return
    onChange({
      ...definition, host: host.trim(), scheme: scheme === 'http' ? 'http' : undefined, port: port === defaultPort ? undefined : port,
      tls_skip_verify: scheme === 'https' && skipVerify ? true : undefined,
      interface: iface.trim() || undefined, uplink: uplink.trim() || undefined,
    })
    onClose()
  }

  return (
    <Sheet open onClose={onClose} title={`${name} settings`}>
      <div className="space-y-5">
        <PreferencesGroup title="Connection">
          <EntryRow title="Management Address" value={host} onChange={(e) => setHost(e.target.value)} className="font-mono" />
          <ChoiceRow title="Protocol" value={scheme} choices={SCHEMES} onChange={(value) => handleScheme(value as 'https' | 'http')} help="Plain HTTP sends the credentials unencrypted; keep it to trusted management networks" />
          <IntegerEntryRow title="Port" value={port} min={1} max={65535} onValueChange={setPort} />
          <SwitchRow title="Skip TLS Verification" subtitle="Accept a self-signed certificate" checked={scheme === 'https' && skipVerify} onCheckedChange={setSkipVerify} disabled={scheme === 'http'} />
        </PreferencesGroup>
        <PreferencesGroup title="Uplink">
          <EntryRow title="Routier Interface" value={iface} onChange={(e) => setIface(e.target.value)} className="font-mono" />
          <EntryRow title="Switch Uplink Port" value={uplink} onChange={(e) => setUplink(e.target.value)} className="font-mono" />
        </PreferencesGroup>
        {model && model.credentials.length > 0 && (
          <PreferencesGroup title="Credentials" description="Sealed in the Routier database, never written to the configuration; empty keeps the current ones">
            <CredentialFields fields={model.credentials} values={credentials} onChange={handleCredential} />
          </PreferencesGroup>
        )}
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleSave} disabled={!host.trim() || busy}>{busy ? 'Saving…' : 'Update Draft'}</Button>
        </div>
      </div>
    </Sheet>
  )
}
