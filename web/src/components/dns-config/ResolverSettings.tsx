import { ForwardRows } from '@/components/dns-config/ForwardRows'
import { ListenBuilder } from '@/components/dns-config/ListenBuilder'
import { DnsServerData, DnsSet, IfaceData, listenOptions, MODE_INFERRED, MODES } from '@/components/dns-config/shared'
import { Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Switch, TagInput } from 'cheval-ui'

type ResolverSettingsShape = {
  cfg: DnsServerData
  set: DnsSet
  ifaces: Record<string, IfaceData>
  ifaceNames: string[]
  vrrpIfaces: string[]
}

export function ResolverSettings({ cfg, set, ifaces, ifaceNames, vrrpIfaces }: ResolverSettingsShape) {
  const listensOnVIP = (cfg.listen ?? []).some((l) => l.startsWith('vips('))

  return (
    <div className="space-y-5">
      <label className="flex items-center gap-3">
        <Switch checked={!!cfg.enabled} onCheckedChange={(v) => set('enabled', v || undefined)} />
        <div>
          <div className="text-sm font-medium">Enable DNS server</div>
          <div className="text-xs text-muted-foreground">Render and run a local BIND name server from this config.</div>
        </div>
      </label>

      <div className="space-y-5">
          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <Label className="text-xs">Mode</Label>
              <Select value={cfg.mode ?? MODE_INFERRED} onValueChange={(v) => set('mode', v === MODE_INFERRED ? undefined : v)}>
                <SelectTrigger className="h-8 text-xs"><SelectValue placeholder="inferred" /></SelectTrigger>
                <SelectContent>
                  {MODES.map((m) => (
                    <SelectItem key={m} value={m} className="text-xs">{m === MODE_INFERRED ? 'inferred from config' : m}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="mt-1 text-xs text-muted-foreground">
                <code>authoritative</code> answers only for its own zones and refuses everything else.
              </p>
            </div>
            <div>
              <Label className="text-xs">Port</Label>
              <Input className="h-8 font-mono text-xs" placeholder="53" value={cfg.port ?? ''} onChange={(e) => set('port', e.target.value ? Number(e.target.value) : undefined)} />
            </div>
          </div>

          <div>
            <Label className="text-xs">Listen addresses</Label>
            <ListenBuilder
              value={cfg.listen ?? []}
              options={listenOptions(ifaces ?? {}, vrrpIfaces)}
              onChange={(v) => set('listen', v.length ? v : undefined)}
            />
            {listensOnVIP && (
              <p className="mt-1 text-xs text-muted-foreground">
                A VRRP address is in the list, so Routier enables <code>ip_nonlocal_bind</code> so BIND can bind it while this node is BACKUP.
              </p>
            )}
          </div>

          <div>
            <Label className="text-xs">Allowed clients</Label>
            <TagInput
              values={cfg.allow_from ?? []}
              onChange={(v) => set('allow_from', v.length ? v : undefined)}
              placeholder="10.0.0.0/24"
            />
            <p className="mt-1 text-xs text-muted-foreground">Required. Also scopes the firewall rule, so the two cannot disagree.</p>
          </div>

          <div>
            <Label className="text-xs">Open the firewall on</Label>
            <div className="flex flex-wrap gap-2 pt-1">
              {ifaceNames.map((n) => {
                const on = (cfg.allow_inbound ?? []).includes(n)
                return (
                  <button
                    key={n}
                    type="button"
                    onClick={() => set('allow_inbound', on
                      ? (cfg.allow_inbound ?? []).filter((x) => x !== n)
                      : [...(cfg.allow_inbound ?? []), n])}
                    className={`rounded-md border px-2 py-1 font-mono text-xs ${on ? 'border-primary bg-primary/10' : 'border-border'}`}
                  >
                    {n}
                  </button>
                )
              })}
            </div>
          </div>

          <div>
            <Label className="text-xs">Upstreams</Label>
            <TagInput
              values={cfg.upstreams ?? []}
              onChange={(v) => set('upstreams', v.length ? v : undefined)}
              placeholder="1.1.1.1"
            />
          </div>

          <div>
            <Label className="text-xs">Forwarding zones</Label>
            <ForwardRows forwards={cfg.forward ?? []} onChange={(f) => set('forward', f.length ? f : undefined)} />
          </div>

          <div className="flex flex-col gap-3 sm:flex-row sm:gap-8">
            <label className="flex items-center gap-3">
              <Switch checked={!!cfg.dnssec} onCheckedChange={(v) => set('dnssec', v || undefined)} />
              <div>
                <div className="text-sm font-medium">Validate DNSSEC</div>
                <div className="text-xs text-muted-foreground">Internal domains are exempted automatically.</div>
              </div>
            </label>
            <label className="flex items-center gap-3">
              <Switch checked={!!cfg.log_queries} onCheckedChange={(v) => set('log_queries', v || undefined)} />
              <div>
                <div className="text-sm font-medium">Log queries</div>
                <div className="text-xs text-muted-foreground">Needed for the live query stream.</div>
              </div>
            </label>
          </div>
      </div>
    </div>
  )
}
