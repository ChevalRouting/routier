import { ZoneTransfers } from '@/components/dns-config/ZoneTransfers'
import { RecordRows } from '@/components/dns-config/RecordRows'
import { DnsZone, impliedZoneRecords } from '@/components/dns-config/shared'
import { Button, TagInput, PreferencesGroup, EntryRow, ComboRow, Row } from 'cheval-ui'

type ZoneEditorShape = {
  zone: DnsZone
  onChange: (z: DnsZone) => void
  onRemove: () => void
  zones?: DnsZone[]
  onZonesChange?: (z: DnsZone[]) => void
}

export function ZoneEditor({ zone, onChange, onRemove, zones, onZonesChange }: ZoneEditorShape) {
  const secondary = (zone.primaries ?? []).length > 0
  const soa = zone.soa ?? {}
  const implied = impliedZoneRecords(zone)
  const glueMissing = (zone.nameservers ?? []).some((ns) => {
    const norm = (s: string) => s.replace(/\.$/, '').toLowerCase()
    if (!norm(ns).endsWith(norm(zone.name))) return false
    return !(zone.records ?? []).some((r) => {
      const owner = r.name === '@' ? norm(zone.name) : norm(r.name).endsWith(norm(zone.name)) ? norm(r.name) : `${norm(r.name)}.${norm(zone.name)}`
      return owner === norm(ns) && (r.type === 'A' || r.type === 'AAAA')
    })
  })

  return (
    <div className="space-y-6">
      <PreferencesGroup
        description={glueMissing ? 'A nameserver inside this zone has no A or AAAA record; BIND rejects the zone without one' : undefined}
        header={<Button variant="ghost" size="sm" className="text-muted-foreground hover:text-destructive" onClick={onRemove}>Remove Zone</Button>}
      >
        <EntryRow title="Zone" value={zone.name} onChange={(e) => onChange({ ...zone, name: e.target.value })} className="font-mono" />
        <EntryRow title="Default TTL (s)" value={zone.ttl ?? ''} onChange={(e) => onChange({ ...zone, ttl: e.target.value ? Number(e.target.value) : undefined })} placeholder="3600" className="font-mono" />
        <EntryRow title="SOA E-Mail" value={soa.email ?? ''} onChange={(e) => onChange({ ...zone, soa: { ...soa, email: e.target.value || undefined } })} placeholder="hostmaster@example.net" className="font-mono" />
        <ComboRow title="Nameservers">
          <TagInput values={zone.nameservers ?? []} onChange={(v) => onChange({ ...zone, nameservers: v.length ? v : undefined })} placeholder="ns1.example.net." mono />
        </ComboRow>
      </PreferencesGroup>

        <ZoneTransfers key={zone.name} zone={zone} onChange={onChange} />

        {!secondary && (
          <div className="space-y-2">
            <h2 className="text-sm font-semibold">Records</h2>
            {implied.length > 0 && (
              <PreferencesGroup description="Nameservers inside this zone need A or AAAA glue records">
                {implied.map((r, k) => (
                  <Row
                    key={k}
                    title={<span className="font-mono">{r.name} {r.type}</span>}
                    subtitle={<span className="font-mono break-all">{r.value}</span>}
                  >
                    <span className="text-xs text-muted-foreground">Automatic</span>
                  </Row>
                ))}
              </PreferencesGroup>
            )}
            <RecordRows
              records={zone.records ?? []}
              onChange={(r) => onChange({ ...zone, records: r.length ? r : undefined })}
              zoneName={zone.name}
              zones={zones}
              onZonesChange={onZonesChange}
            />
          </div>
        )}
    </div>
  )
}
