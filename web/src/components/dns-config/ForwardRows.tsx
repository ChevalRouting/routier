import { DnsForward } from '@/components/dns-config/shared'
import { Button, ComboRow, EntryRow, PreferencesGroup, SwitchRow, TagInput } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'

type ForwardRowsShape = {
  forwards: DnsForward[]
  onChange: (f: DnsForward[]) => void
}

export function ForwardRows({ forwards, onChange }: ForwardRowsShape) {
  const set = (i: number, next: DnsForward) => onChange(forwards.map((f, j) => (j === i ? next : f)))

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-semibold">Forwarding Zones</h2>
        <Button variant="outline" size="sm" onClick={() => onChange([...forwards, { domain: '' }])} className="gap-1.5">
          <Plus className="h-4 w-4" />Add Forwarding Zone
        </Button>
      </div>
      {forwards.map((f, i) => (
        <PreferencesGroup
          key={i}
          title={f.domain || 'New Forwarding Zone'}
          header={(
            <Button variant="ghost" size="icon" onClick={() => onChange(forwards.filter((_, j) => j !== i))} aria-label="Remove Forwarding Zone" title="Remove Forwarding Zone" className="text-muted-foreground hover:text-destructive">
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        >
          <EntryRow title="Domain" value={f.domain} onChange={(e) => set(i, { ...f, domain: e.target.value })} placeholder="corp.example.com" className="font-mono" />
          <ComboRow title="Recursors">
            <TagInput values={f.servers ?? []} onChange={(v) => set(i, { ...f, servers: v.length ? v : undefined })} placeholder="10.0.0.53" mono />
          </ComboRow>
          <SwitchRow title="DNSSEC Signed" checked={!!f.dnssec} onCheckedChange={(v) => set(i, { ...f, dnssec: v || undefined })} />
        </PreferencesGroup>
      ))}
    </div>
  )
}
