import { DhcpConfigData, DhcpDDNS } from '@/components/dhcp-config/shared'
import { Card, CardContent, CardHeader, CardTitle, Input, Label, Switch } from 'cheval-ui'

type DdnsTabShape = {
  cfg: DhcpConfigData
  onChange: (next: DhcpConfigData) => void
}

export function DdnsTab({ cfg, onChange }: DdnsTabShape) {
  const d = cfg.ddns ?? {}
  const set = <K extends keyof DhcpDDNS>(k: K, v: DhcpDDNS[K]) => onChange({ ...cfg, ddns: { ...d, [k]: v } })

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Dynamic DNS</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <label className="flex items-center gap-3">
          <Switch checked={!!d.enabled} onCheckedChange={(v) => set('enabled', v || undefined)} />
          <div>
            <div className="text-sm font-medium">Register leases in DNS</div>
            <div className="text-xs text-muted-foreground">Kea updates the local BIND with A/AAAA and PTR records as leases come and go. Requires the DNS server to be enabled.</div>
          </div>
        </label>

        {d.enabled && (
          <div className="space-y-4">
            <div className="space-y-1.5">
              <Label className="text-[11px] text-muted-foreground">Forward domain</Label>
              <Input value={d.domain ?? ''} onChange={(e) => set('domain', e.target.value || undefined)} placeholder="lan.example.com" className="h-8 text-xs font-mono" />
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-1.5">
                <Label className="text-[11px] text-muted-foreground">Record TTL (seconds)</Label>
                <Input type="number" value={d.ttl ?? ''} onChange={(e) => set('ttl', e.target.value ? Number(e.target.value) : undefined)} placeholder="3600" className="h-8 text-xs" />
              </div>
              <label className="flex items-center gap-3 pt-5">
                <Switch checked={d.reverse !== false} onCheckedChange={(v) => set('reverse', v ? undefined : false)} />
                <div className="text-sm">Update reverse (PTR) zones</div>
              </label>
            </div>
            <p className="text-xs text-muted-foreground">The forward domain is appended to client hostnames. The closest existing parent zone is reused: ans.mvinc.fr registers hostname.ans.mvinc.fr in mvinc.fr when that zone exists. Existing reverse zones are also reused. Missing zones are created automatically. The TSIG key that authenticates updates is generated on first apply and stored in the config.</p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
