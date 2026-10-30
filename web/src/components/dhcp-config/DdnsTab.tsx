import { DhcpConfigData, DhcpDDNS } from '@/components/dhcp-config/shared'
import { EntryRow, PreferencesGroup, SwitchRow } from 'cheval-ui'
import { NumberRow } from '@/components/ui/NumberRow'

type DdnsTabShape = {
  cfg: DhcpConfigData
  onChange: (next: DhcpConfigData) => void
}

export function DdnsTab({ cfg, onChange }: DdnsTabShape) {
  const d = cfg.ddns ?? {}
  const set = <K extends keyof DhcpDDNS>(k: K, v: DhcpDDNS[K]) => onChange({ ...cfg, ddns: { ...d, [k]: v } })

  return (
    <PreferencesGroup
      title="Dynamic DNS"
      description={d.enabled ? 'Records go into the closest existing parent zone; missing zones are created. The TSIG key is generated on first apply.' : undefined}
    >
      <SwitchRow
        title="Register Leases in DNS"
        subtitle="Updates A, AAAA and PTR records in the local BIND; requires the DNS server"
        checked={!!d.enabled}
        onCheckedChange={(v) => set('enabled', v || undefined)}
      />
      {d.enabled && (
        <>
          <EntryRow title="Forward Domain" value={d.domain ?? ''} onChange={(e) => set('domain', e.target.value || undefined)} placeholder="lan.example.com" className="font-mono" />
          <NumberRow title="Record TTL (s)" value={d.ttl} onChange={(v) => set('ttl', v)} placeholder="3600" />
          <SwitchRow title="Update Reverse Zones" checked={d.reverse !== false} onCheckedChange={(v) => set('reverse', v ? undefined : false)} />
        </>
      )}
    </PreferencesGroup>
  )
}
