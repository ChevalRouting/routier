import { checkCIDR } from '@/lib/validate'
import { Badge, ComboRow, EmptyState, EntryRow, MenuButton, PreferencesGroup, Sheet, SwitchRow, TagInput, type MenuItem } from 'cheval-ui'
import { Plus } from 'lucide-react'
import { useState } from 'react'
import { EntityRow } from '@/components/ui/EntityRow'
import { NumberRow } from '@/components/ui/NumberRow'
import { plural } from '@/lib/utils'
import { NameSelect } from '../shared'
import { BGPAddressFamily, STANDARD_AFS } from '../types'

type BGPAddressFamiliesPanelShape = {
  afs: Record<string, BGPAddressFamily>
  onChange: (v: Record<string, BGPAddressFamily>) => void
  routeMapNames: string[]
  vrfNames: string[]
  onDirty: () => void
}

export function BGPAddressFamiliesPanel({
  afs, onChange, routeMapNames, vrfNames, onDirty,
}: BGPAddressFamiliesPanelShape) {
  const [openName, setOpenName] = useState<string | null>(null)
  const configured = Object.keys(afs)
  const available = STANDARD_AFS.filter((af) => !configured.includes(af))

  const upd = (name: string, patch: Partial<BGPAddressFamily>) => {
    onChange({ ...afs, [name]: { ...afs[name], ...patch } }); onDirty()
  }
  const remove = (name: string) => {
    const next = { ...afs }; delete next[name]; onChange(next); onDirty()
    if (openName === name) setOpenName(null)
  }
  const add = (name: string) => {
    if (name in afs) return
    onChange({ ...afs, [name]: { networks: [], redistribute: [] } }); setOpenName(name); onDirty()
  }

  const open = openName ? afs[openName] ?? {} : null
  const isEVPN = openName === 'l2vpn-evpn'
  const isUnicast = openName === 'ipv4-unicast' || openName === 'ipv6-unicast'
  const set = (patch: Partial<BGPAddressFamily>) => openName && upd(openName, patch)
  const addItems: MenuItem[][] = [available.map((af) => ({ key: af, label: af, onSelect: () => add(af) }))]

  const addButton = (
    <MenuButton label="Add Address Family" sections={addItems} className="h-8 border border-input px-3">
      <Plus className="h-4 w-4" />Add Address Family
    </MenuButton>
  )

  return (
    <div className="space-y-6">
      {configured.length === 0 ? (
        <EmptyState className="py-10" title="No Address Families" action={addButton} />
      ) : (
        <PreferencesGroup
          title="Address Families"
          header={available.length > 0 && addButton}
        >
          {configured.map((afName) => {
            const af = afs[afName] ?? {}
            return (
              <EntityRow key={afName} title={afName} onOpen={() => setOpenName(afName)} onRemove={() => remove(afName)} removeLabel="Remove Address Family">
                <Badge variant="outline">{plural(af.networks?.length ?? 0, 'network')}</Badge>
                <Badge variant="outline">{af.redistribute?.length ?? 0} redistributed</Badge>
                {af.default_originate && <Badge variant="secondary">default originate</Badge>}
              </EntityRow>
            )
          })}
        </PreferencesGroup>
      )}

      <Sheet open={open !== null} onClose={() => setOpenName(null)} title={openName ?? ''}>
        {open && (
          <div className="space-y-6">
            <PreferencesGroup>
              <ComboRow title="Networks to Advertise">
                <TagInput values={open.networks ?? []} onChange={(v) => set({ networks: v })} placeholder="10.0.0.0/24" mono validate={checkCIDR} />
              </ComboRow>
              <ComboRow title="Redistribute">
                <TagInput values={open.redistribute ?? []} onChange={(v) => set({ redistribute: v })} placeholder="connected" />
              </ComboRow>
              <NumberRow title="Maximum Paths" value={open.maximum_paths} onChange={(v) => set({ maximum_paths: v })} placeholder="1" />
              <SwitchRow title="Default Originate" checked={!!open.default_originate} onCheckedChange={(v) => set({ default_originate: v })} />
              <ComboRow title="Import VRF">
                <TagInput values={open.import_vrf ?? []} onChange={(v) => set({ import_vrf: v })} placeholder={vrfNames[0] ?? 'vrf name'} suggestions={vrfNames} />
              </ComboRow>
            </PreferencesGroup>

            {isUnicast && (
              <PreferencesGroup title="VPN Route Leaking" description="Requires a route distinguisher, matching route targets, and import or export in each VRF">
                <EntryRow title="Route Distinguisher" value={open.rd ?? ''} onChange={(e) => set({ rd: e.target.value || undefined })} placeholder="65000:1" className="font-mono" />
                <ComboRow title="Import Route Targets">
                  <TagInput values={open.rt_vpn_import ?? []} onChange={(v) => set({ rt_vpn_import: v.length ? v : undefined })} placeholder="65000:1" mono />
                </ComboRow>
                <ComboRow title="Export Route Targets">
                  <TagInput values={open.rt_vpn_export ?? []} onChange={(v) => set({ rt_vpn_export: v.length ? v : undefined })} placeholder="65000:1" mono />
                </ComboRow>
                <SwitchRow title="Import VPN" checked={!!open.import_vpn} onCheckedChange={(v) => set({ import_vpn: v || undefined })} />
                <SwitchRow title="Export VPN" checked={!!open.export_vpn} onCheckedChange={(v) => set({ export_vpn: v || undefined })} />
                <SwitchRow title="Auto Label" checked={!!open.label_vpn_export_auto} onCheckedChange={(v) => set({ label_vpn_export_auto: v || undefined })} />
                <ComboRow title="Import Route Map">
                  <NameSelect value={open.route_map_vpn_import} onChange={(v) => set({ route_map_vpn_import: v })} names={routeMapNames} />
                </ComboRow>
                <ComboRow title="Export Route Map">
                  <NameSelect value={open.route_map_vpn_export} onChange={(v) => set({ route_map_vpn_export: v })} names={routeMapNames} />
                </ComboRow>
              </PreferencesGroup>
            )}

            {isEVPN && (
              <PreferencesGroup title="EVPN Advertisements">
                <SwitchRow title="All VNIs" checked={!!open.advertise_all_vni} onCheckedChange={(v) => set({ advertise_all_vni: v })} />
                <SwitchRow title="Default Gateway" checked={!!open.advertise_default_gateway} onCheckedChange={(v) => set({ advertise_default_gateway: v })} />
                <SwitchRow title="SVI IP" checked={!!open.advertise_svi_ip} onCheckedChange={(v) => set({ advertise_svi_ip: v })} />
                <ComboRow title="Advertise Address Families">
                  <TagInput values={open.advertise ?? []} onChange={(v) => set({ advertise: v })} suggestions={['ipv4-unicast', 'ipv6-unicast']} validate={(v) => ['ipv4-unicast', 'ipv6-unicast'].includes(v) ? null : 'Use ipv4-unicast or ipv6-unicast'} mono />
                </ComboRow>
                <ComboRow title="Import Route Targets">
                  <TagInput values={open.route_target_import ?? []} onChange={(v) => set({ route_target_import: v })} placeholder="65001:100" mono />
                </ComboRow>
                <ComboRow title="Export Route Targets">
                  <TagInput values={open.route_target_export ?? []} onChange={(v) => set({ route_target_export: v })} placeholder="65001:100" mono />
                </ComboRow>
              </PreferencesGroup>
            )}
          </div>
        )}
      </Sheet>
    </div>
  )
}
