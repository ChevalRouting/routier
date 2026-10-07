import { DnsForward } from '@/components/dns-config/shared'
import { Button, Input, Label, Switch, TagInput } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'

type ForwardRowsShape = {
  forwards: DnsForward[]
  onChange: (f: DnsForward[]) => void
}

export function ForwardRows({ forwards, onChange }: ForwardRowsShape) {
  const set = (i: number, next: DnsForward) => onChange(forwards.map((f, j) => (j === i ? next : f)))

  return (
    <div className="space-y-2">
      {forwards.map((f, i) => (
        <div key={i} className="space-y-2 rounded-md bg-muted/30 p-2">
          <div className="flex items-center gap-2">
            <Input
              className="h-8 flex-1 font-mono text-xs"
              placeholder="domain"
              value={f.domain}
              onChange={(e) => set(i, { ...f, domain: e.target.value })}
            />
            <label className="flex items-center gap-1 text-xs text-muted-foreground">
              <Switch checked={!!f.dnssec} onCheckedChange={(v) => set(i, { ...f, dnssec: v || undefined })} />
              signed
            </label>
            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => onChange(forwards.filter((_, j) => j !== i))}>
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
          <div>
            <Label className="text-xs">Recursors</Label>
            <TagInput
              values={f.servers ?? []}
              onChange={(v) => set(i, { ...f, servers: v.length ? v : undefined })}
              placeholder="server IP"
            />
          </div>
        </div>
      ))}
      <Button variant="outline" size="sm" onClick={() => onChange([...forwards, { domain: '' }])}>
        <Plus className="mr-1 h-3 w-3" /> Forward a domain
      </Button>
    </div>
  )
}
