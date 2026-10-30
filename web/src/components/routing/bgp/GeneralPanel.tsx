import { ExtraDirectives } from '@/components/ExtraDirectives'
import { ComboRow, EntryRow, NumberInput, PreferencesGroup, SwitchRow } from 'cheval-ui'
import { BGPConfig } from '../types'

type BGPGeneralPanelShape = {
  bgp: BGPConfig
  setBGP: (v: BGPConfig) => void
  onDirty: () => void
}

type BoolOptsShape = {
    key: 'no_ebgp_requires_policy' | 'no_default_ipv4_unicast' | 'no_import_check' | 'no_rib'
    label: string
    desc: string
  }

export function BGPGeneralPanel({
  bgp, setBGP, onDirty,
}: BGPGeneralPanelShape) {
  const upd = (patch: Partial<BGPConfig>) => { setBGP({ ...bgp, ...patch }); onDirty() }
  const boolOpts: BoolOptsShape[] = [
    { key: 'no_ebgp_requires_policy', label: 'No eBGP Requires Policy', desc: 'Allow eBGP sessions without explicit route policies' },
    { key: 'no_default_ipv4_unicast', label: 'No Default IPv4 Unicast', desc: 'Disable IPv4 unicast as the default address family' },
    { key: 'no_import_check', label: 'No Import Check', desc: 'Do not check that the nexthop is reachable' },
    { key: 'no_rib', label: 'No RIB (route reflector)', desc: 'Do not install BGP routes into zebra or the kernel FIB' },
  ]
  return (
    <div className="space-y-6">
      <PreferencesGroup>
        <ComboRow title="ASN">
          <NumberInput value={bgp.asn || undefined} onChange={(v) => upd({ asn: v ?? 0 })} placeholder="65000" className="font-mono" />
        </ComboRow>
        <EntryRow title="Router ID" value={bgp.router_id} onChange={(e) => upd({ router_id: e.target.value })} placeholder="10.0.0.1" className="font-mono" />
      </PreferencesGroup>
      <PreferencesGroup title="Options">
        {boolOpts.map(({ key, label, desc }) => (
          <SwitchRow key={key} title={label} subtitle={desc} checked={!!bgp[key]} onCheckedChange={(v) => upd({ [key]: v })} />
        ))}
      </PreferencesGroup>
      <ExtraDirectives label="Additional Directives (router bgp)" values={bgp.extra} onChange={(v) => upd({ extra: v })} placeholder="coalesce-time 1000" />
    </div>
  )
}
