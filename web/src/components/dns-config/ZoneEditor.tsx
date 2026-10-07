import { RecordRows } from '@/components/dns-config/RecordRows'
import { DnsZone, impliedZoneRecords } from '@/components/dns-config/shared'
import { Button, Input, Label, TagInput } from 'cheval-ui'
import { Trash2 } from 'lucide-react'

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
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-2">
        <h2 className="truncate font-mono text-sm font-medium">{zone.name || 'new zone'}</h2>
        <Button variant="ghost" size="sm" className="gap-1.5 text-muted-foreground" onClick={onRemove}>
          <Trash2 className="h-4 w-4" /> Remove
        </Button>
      </div>

      <div className="grid gap-3 sm:grid-cols-3">
          <div>
            <Label className="text-xs">Zone</Label>
            <Input className="h-8 font-mono text-xs" value={zone.name} onChange={(e) => onChange({ ...zone, name: e.target.value })} />
          </div>
          <div>
            <Label className="text-xs">Default TTL</Label>
            <Input className="h-8 font-mono text-xs" placeholder="3600" value={zone.ttl ?? ''} onChange={(e) => onChange({ ...zone, ttl: e.target.value ? Number(e.target.value) : undefined })} />
          </div>
          <div>
            <Label className="text-xs">SOA e-mail</Label>
            <Input className="h-8 font-mono text-xs" placeholder="hostmaster@example.net" value={soa.email ?? ''} onChange={(e) => onChange({ ...zone, soa: { ...soa, email: e.target.value || undefined } })} />
          </div>
        </div>

        <div>
          <Label className="text-xs">Nameservers</Label>
          <TagInput
            values={zone.nameservers ?? []}
            onChange={(v) => onChange({ ...zone, nameservers: v.length ? v : undefined })}
            placeholder="ns1.example.net."
          />
          {glueMissing && (
            <p className="mt-1 text-xs text-amber-600 dark:text-amber-500">
              A nameserver inside this zone has no A or AAAA record. BIND rejects the zone without one.
            </p>
          )}
        </div>

        <div>
          <Label className="text-xs">Transferred in from</Label>
          <TagInput
            values={zone.primaries ?? []}
            onChange={(v) => onChange({ ...zone, primaries: v.length ? v : undefined })}
            placeholder="primary server IP (leave empty for a primary zone)"
          />
        </div>

        {!secondary && (
          <div>
            <Label className="text-xs">Records</Label>
            {implied.length > 0 && (
              <div className="mb-2 space-y-2">
                {implied.map((r, k) => (
                  <div key={k} className="flex items-center gap-2 opacity-60">
                    <Input disabled className="h-8 w-32 font-mono text-xs" value={r.name} />
                    <Input disabled className="h-8 w-24 font-mono text-xs" value={r.type} />
                    <Input disabled className="h-8 flex-1 font-mono text-xs" value={r.value} />
                    <span className="w-8 shrink-0 text-center text-[10px] uppercase text-muted-foreground">auto</span>
                  </div>
                ))}
                <p className="text-xs text-muted-foreground">
                  Generated from the fields above. Add A or AAAA glue records below for any nameserver inside this zone.
                </p>
              </div>
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
