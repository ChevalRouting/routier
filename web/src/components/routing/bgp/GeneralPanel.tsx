import { Input } from 'cheval-ui'
import { NumberInput } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Separator } from 'cheval-ui'
import { Switch } from 'cheval-ui'
import { ExtraDirectives } from '@/components/ExtraDirectives'
import { BGPConfig } from '../types'

export function BGPGeneralPanel({
  bgp, setBGP, onDirty,
}: {
  bgp: BGPConfig
  setBGP: (v: BGPConfig) => void
  onDirty: () => void
}) {
  const upd = (patch: Partial<BGPConfig>) => { setBGP({ ...bgp, ...patch }); onDirty() }
  const boolOpts: {
    key: 'no_ebgp_requires_policy' | 'no_default_ipv4_unicast' | 'no_import_check' | 'no_rib'
    label: string
    desc: string
  }[] = [
    { key: 'no_ebgp_requires_policy', label: 'No eBGP requires policy', desc: 'Allow eBGP sessions without explicit route policies' },
    { key: 'no_default_ipv4_unicast', label: 'No default IPv4 unicast', desc: 'Disable IPv4 unicast as the default address family' },
    { key: 'no_import_check', label: 'No import check', desc: 'Do not check that the nexthop is reachable' },
    { key: 'no_rib', label: 'No RIB (route reflector)', desc: 'Do not install BGP routes into zebra or the kernel FIB; run as a pure route reflector' },
  ]
  return (
    <div className="space-y-6 max-w-lg">
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-1.5">
          <Label className="text-xs">ASN</Label>
          <NumberInput
            value={bgp.asn || undefined}
            onChange={(v) => upd({ asn: v ?? 0 })}
            placeholder="65000"
            className="font-mono"
          />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs">Router ID</Label>
          <Input
            value={bgp.router_id}
            onChange={(e) => upd({ router_id: e.target.value })}
            placeholder="10.0.0.1"
            className="font-mono"
          />
        </div>
      </div>
      <Separator />
      <div className="space-y-3">
        <p className="text-xs font-semibold text-muted-foreground">Options</p>
        {boolOpts.map(({ key, label, desc }) => (
          <div key={key} className="flex items-start justify-between gap-3">
            <div>
              <div className="text-sm font-medium">{label}</div>
              <div className="text-xs text-muted-foreground">{desc}</div>
            </div>
            <Switch checked={!!bgp[key]} onCheckedChange={(v) => upd({ [key]: v })} />
          </div>
        ))}
      </div>
      <Separator />
      <ExtraDirectives label="Additional router bgp directives" values={bgp.extra} onChange={(v) => upd({ extra: v })} placeholder="coalesce-time 1000" />
    </div>
  )
}

