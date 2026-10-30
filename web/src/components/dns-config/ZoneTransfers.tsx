import { DnsZone } from '@/components/dns-config/shared'
import { Button, ChoiceRow, ComboRow, EntryRow, PreferencesGroup, TagInput } from 'cheval-ui'
import { useState } from 'react'

type ZoneTransfersShape = {
  zone: DnsZone
  onChange: (zone: DnsZone) => void
}

export function ZoneTransfers({ zone, onChange }: ZoneTransfersShape) {
  const role = (zone.primaries ?? []).length ? 'secondary' : 'primary'
  const [revealed, setRevealed] = useState(false)
  const key = zone.transfer_key
  const secondary = role === 'secondary'

  const handleRole = (value: string) => {
    onChange(value === 'secondary'
      ? { ...zone, records: undefined, secondaries: undefined, primaries: [''], transfer_source: zone.transfer_source ?? zone.notify_source, transfer_source_v6: zone.transfer_source_v6 ?? zone.notify_source_v6, notify_source: undefined, notify_source_v6: undefined }
      : { ...zone, primaries: undefined, notify_source: zone.notify_source ?? zone.transfer_source, notify_source_v6: zone.notify_source_v6 ?? zone.transfer_source_v6, transfer_source: undefined, transfer_source_v6: undefined })
  }
  const handlePeers = (values: string[]) => {
    onChange({ ...zone, [secondary ? 'primaries' : 'secondaries']: values.length ? values : secondary ? [''] : undefined })
  }
  const handleTransferSource = (event: React.ChangeEvent<HTMLInputElement>) => {
    onChange({ ...zone, [secondary ? 'transfer_source' : 'notify_source']: event.target.value.trim() || undefined })
  }
  const handleTransferSourceV6 = (event: React.ChangeEvent<HTMLInputElement>) => {
    onChange({ ...zone, [secondary ? 'transfer_source_v6' : 'notify_source_v6']: event.target.value.trim() || undefined })
  }
  const handleName = (event: React.ChangeEvent<HTMLInputElement>) => {
    onChange({ ...zone, transfer_key: { name: event.target.value, secret: key?.secret ?? '' } })
  }
  const handleSecret = (event: React.ChangeEvent<HTMLInputElement>) => {
    onChange({ ...zone, transfer_key: { name: key?.name ?? '', secret: event.target.value.trim() } })
  }
  const handleGenerate = () => {
    const bytes = crypto.getRandomValues(new Uint8Array(32))
    const secret = btoa(String.fromCharCode(...bytes))
    onChange({ ...zone, transfer_key: { name: key?.name || `transfer.${zone.name}`, secret } })
  }
  const handleRemove = () => onChange({ ...zone, transfer_key: undefined })
  const handleReveal = () => setRevealed(!revealed)

  return (
    <PreferencesGroup className="[&>div:last-child>div]:grid-cols-1 md:[&>div:last-child>div]:grid-cols-[minmax(0,1fr)_20rem]" title="Zone Transfers" description="Same HMAC-SHA256 key name and secret on the primary and secondary; NS records do not authorize transfers">
      <ChoiceRow title="Zone Role" value={role} choices={[{ value: 'primary', label: 'Primary' }, { value: 'secondary', label: 'Secondary' }]} onChange={handleRole} help="Switching to secondary removes staged local records. The primary supplies the zone contents." />
      <ComboRow title={secondary ? 'Primary Servers' : 'Secondary Servers'} subtitle={secondary ? 'Addresses to pull this zone from' : 'Addresses allowed to pull this zone and receive change notifications'}>
        <TagInput values={((secondary ? zone.primaries : zone.secondaries) ?? []).filter(Boolean)} onChange={handlePeers} placeholder="192.0.2.53" />
      </ComboRow>
      <EntryRow title={secondary ? 'Transfer Source (IPv4)' : 'Notify Source (IPv4)'} value={(secondary ? zone.transfer_source : zone.notify_source) ?? ''} onChange={handleTransferSource} placeholder="Automatic" help={secondary ? 'Local address for zone pulls and SOA refreshes. Leave empty or use * for automatic selection.' : 'Local address for notifications to secondary servers. Leave empty or use * for automatic selection.'} />
      <EntryRow title={secondary ? 'Transfer Source (IPv6)' : 'Notify Source (IPv6)'} value={(secondary ? zone.transfer_source_v6 : zone.notify_source_v6) ?? ''} onChange={handleTransferSourceV6} placeholder="Automatic" help={secondary ? 'Local IPv6 address for zone pulls and SOA refreshes.' : 'Local IPv6 address for notifications to secondary servers.'} />
      <EntryRow title="TSIG Key Name" value={key?.name ?? ''} onChange={handleName} placeholder="transfer.example.net" />
      <EntryRow title="Import TSIG Secret" type={revealed ? 'text' : 'password'} value={key?.secret ?? ''} onChange={handleSecret} placeholder="Base64 shared secret" />
      <ComboRow title="TSIG Key" subtitle="Without a key, transfers use IP authorization">
        <div className="flex flex-wrap gap-2">
          <Button size="sm" onClick={handleGenerate}>Generate Key</Button>
          {key && <Button size="sm" variant="outline" onClick={handleReveal}>{revealed ? 'Hide Secret' : 'Show Secret'}</Button>}
          {key && <Button size="sm" variant="outline" onClick={handleRemove}>Remove Key</Button>}
        </div>
      </ComboRow>
    </PreferencesGroup>
  )
}
