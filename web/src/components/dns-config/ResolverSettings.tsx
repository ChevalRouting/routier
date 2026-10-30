import { ForwardRows } from '@/components/dns-config/ForwardRows'
import { ListenBuilder } from '@/components/dns-config/ListenBuilder'
import { DnsServerData, DnsSet, IfaceData, listenOptions, MODE_INFERRED, MODES } from '@/components/dns-config/shared'
import { ComboRow, EntryRow, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, SwitchRow, TagInput } from 'cheval-ui'

type ResolverSettingsShape = {
  cfg: DnsServerData
  set: DnsSet
  ifaces: Record<string, IfaceData>
  ifaceNames: string[]
  vrrpIfaces: string[]
}

export function ResolverSettings({ cfg, set, ifaces, ifaceNames, vrrpIfaces }: ResolverSettingsShape) {
  const handleDisableAutoRules = (disabled: boolean) => set('disable_auto_rules', disabled || undefined)
  const listensOnVIP = (cfg.listen ?? []).some((l) => l.startsWith('vips('))

  return (
    <div className="space-y-6">
      <PreferencesGroup>
        <SwitchRow title="DNS Server" subtitle="Runs a local BIND name server" checked={!!cfg.enabled} onCheckedChange={(v) => set('enabled', v || undefined)} />
        <ComboRow title="Mode" subtitle="authoritative answers only for its own zones">
          <Select value={cfg.mode ?? MODE_INFERRED} onValueChange={(v) => set('mode', v === MODE_INFERRED ? undefined : v)}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              {MODES.map((m) => <SelectItem key={m} value={m}>{m === MODE_INFERRED ? 'inferred from config' : m}</SelectItem>)}
            </SelectContent>
          </Select>
        </ComboRow>
        <EntryRow title="Port" value={cfg.port ?? ''} onChange={(e) => set('port', e.target.value ? Number(e.target.value) : undefined)} placeholder="53" className="font-mono" />
        <SwitchRow title="Validate DNSSEC" subtitle="Internal domains are exempt" checked={!!cfg.dnssec} onCheckedChange={(v) => set('dnssec', v || undefined)} />
        <SwitchRow title="Log Queries" subtitle="Needed for the live query stream" checked={!!cfg.log_queries} onCheckedChange={(v) => set('log_queries', v || undefined)} />
      </PreferencesGroup>

      <PreferencesGroup title="Listening">
        <ComboRow
          title="Listen Addresses"
          subtitle={listensOnVIP ? 'Enables ip_nonlocal_bind so BIND can bind the VRRP address while BACKUP' : 'iface() and vips() survive an interface being renumbered'}
        >
          <ListenBuilder value={cfg.listen ?? []} options={listenOptions(ifaces ?? {}, vrrpIfaces)} onChange={(v) => set('listen', v.length ? v : undefined)} />
        </ComboRow>
        <ComboRow title="Allowed Clients" subtitle="Required, also scopes the firewall rule">
          <TagInput values={cfg.allow_from ?? []} onChange={(v) => set('allow_from', v.length ? v : undefined)} placeholder="10.0.0.0/24" mono />
        </ComboRow>
      </PreferencesGroup>

      <PreferencesGroup title="Firewall">
        <SwitchRow title="Automatic Firewall Rules" checked={!cfg.disable_auto_rules} onCheckedChange={(v) => handleDisableAutoRules(!v)} />
        {!cfg.disable_auto_rules && (
          <ComboRow title="Open on Interfaces">
            <TagInput
              values={cfg.allow_inbound ?? []}
              onChange={(v) => set('allow_inbound', v)}
              suggestions={ifaceNames}
              validate={(v) => (ifaceNames.includes(v) ? null : 'Unknown interface')}
              placeholder={ifaceNames[0] ?? 'eth0'}
              mono
            />
          </ComboRow>
        )}
      </PreferencesGroup>

      <PreferencesGroup title="Upstream">
        <ComboRow title="Upstreams">
          <TagInput values={cfg.upstreams ?? []} onChange={(v) => set('upstreams', v.length ? v : undefined)} placeholder="1.1.1.1" mono />
        </ComboRow>
      </PreferencesGroup>

      <ForwardRows forwards={cfg.forward ?? []} onChange={(f) => set('forward', f.length ? f : undefined)} />
    </div>
  )
}
