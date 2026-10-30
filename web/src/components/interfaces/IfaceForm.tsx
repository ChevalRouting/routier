import { Bond, BOND_HASH_POLICIES, BOND_LACP_RATES, BOND_MODES, emptyBond, emptyVxlan, Iface, IFACE_TYPES, IfaceFormProps, matchNic, MemberPicker, nicLabel, nicSelector, NicSelect, selectorLabel, TUNNEL_MODES, Vxlan } from '@/components/interfaces/shared'
import { checkInterfaceAddress, checkIP, checkMulticastIP, checkPort, checkVLANId } from '@/lib/validate'
import { Button, ComboRow, EntryRow, Label, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Switch, TagInput } from 'cheval-ui'
import {
  ChevronDown, ChevronRight
} from 'lucide-react'
import { useState } from 'react'

export function IfaceForm({ name, iface, vrfNames, parents, transports = [], nics, onChange, onNameChange, onAdd, onDone }: IfaceFormProps) {
  const [bridgeOpen, setBridgeOpen] = useState(false)

  const isVirtual = iface.type === 'dummy' || iface.type === 'bridge' || iface.type === 'vxlan' || iface.type === 'bond' || iface.type === 'vxlan-bridge' || iface.type === 'vnet'
  const isDummy = iface.type === 'dummy'
  const isVxlanBridge = iface.type === 'vxlan-bridge'
  const isVnet = iface.type === 'vnet'
  const isVxlan = iface.type === 'vxlan'
  const isBond = iface.type === 'bond'
  const isVlan = iface.type === 'vlan'
  const isTunnel = iface.type === 'tunnel'
  const selectedNic = matchNic(nics, iface, name)
  const parentOptions = parents.filter((p) => p !== name)

  const set = <K extends keyof Iface>(key: K, val: Iface[K]) =>
    onChange({ ...iface, [key]: val })

  const setVxlan = (patch: Partial<Vxlan>) =>
    onChange({ ...iface, vxlan: { ...(iface.vxlan ?? emptyVxlan()), ...patch } })

  const setBond = (patch: Partial<Bond>) =>
    onChange({ ...iface, bond: { ...(iface.bond ?? emptyBond()), ...patch } })

  const hasBridge = !!iface.bridge
  const toggleBridge = () => {
    if (hasBridge) {
      const { bridge: _b, ...rest } = iface
      onChange(rest as Iface)
    } else {
      onChange({ ...iface, bridge: { members: [], stp: false } })
      setBridgeOpen(true)
    }
  }

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        {onNameChange !== undefined && (
          <EntryRow
            title="Name"
            autoFocus
            value={name}
            onChange={(e) => onNameChange(e.target.value)}
            placeholder="eth0"
            className="font-mono"
          />
        )}
        <ComboRow title="Type">
          <Select
            value={iface.type || 'physical'}
            onValueChange={(v) => onChange({
              ...iface,
              type: v === 'physical' ? undefined : v,
              vxlan: v === 'vxlan-bridge' ? { vni: 0, local: iface.vxlan?.local, vtep: iface.vxlan?.vtep, port: iface.vxlan?.port } : v === 'vxlan' ? (iface.vxlan ?? emptyVxlan()) : undefined,
              vnet: v === 'vnet' ? (iface.vnet ?? { vni: 0, vlan: 0 }) : undefined,
              bond: v === 'bond' ? (iface.bond ?? emptyBond()) : undefined,
              vlan: v === 'vlan' ? (iface.vlan ?? { id: 0 }) : undefined,
              bridge: v === 'vxlan-bridge' || v === 'vxlan' || v === 'tunnel' || v === 'bond' || v === 'vlan' ? undefined : iface.bridge,
              mode: v === 'tunnel' ? (iface.mode ?? 'gre') : undefined,
              select: (v === 'vlan') !== (iface.type === 'vlan') ? '' : iface.select,
            })}
          >
            <SelectTrigger className="font-mono">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {IFACE_TYPES.map((t) => (
                <SelectItem key={t.value} value={t.value} className="font-mono">{t.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </ComboRow>
        {!isVirtual && !isTunnel && !isVlan && (
          <ComboRow title="Physical Device">
            <Select value={selectedNic?.name ?? ''} onValueChange={(v) => set('select', nicSelector(nics.find((n) => n.name === v) ?? { name: v }))}>
              <SelectTrigger className="font-mono">
                <SelectValue placeholder="- select a device -" />
              </SelectTrigger>
              <SelectContent>
                {iface.select && !selectedNic && (
                  <SelectItem value={iface.select} className="font-mono">{iface.select}</SelectItem>
                )}
                {nics.map((nic) => (
                  <SelectItem key={nic.name} value={nic.name} className="font-mono">{nicLabel(nic)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </ComboRow>
        )}
        {isVlan && (
          <ComboRow title={isVnet ? 'VXLAN Bridge' : 'Parent Interface'}>
            <NicSelect value={iface.select || ''} nics={[]} extra={parentOptions} placeholder="- select a parent -" onChange={(v) => set('select', v)} />
          </ComboRow>
        )}
        {isVnet && (
          <ComboRow title="VXLAN Bridge">
            <Select value={iface.select || ''} onValueChange={(v) => set('select', v)}>
              <SelectTrigger className="font-mono"><SelectValue placeholder="Select a transport" /></SelectTrigger>
              <SelectContent>
                {transports.map((transport) => <SelectItem key={transport} value={transport}>{transport}</SelectItem>)}
              </SelectContent>
            </Select>
          </ComboRow>
        )}
        <EntryRow
          title={isVxlanBridge ? 'Overlay MTU' : 'MTU (0 = auto)'}
          value={iface.mtu || ''}
          onChange={(e) => set('mtu', Number(e.target.value) || 0)}
          placeholder="0"
          className="font-mono"
        />
        {!isVxlanBridge && <ComboRow title="VRF">
          <Select
            value={iface.vrf || '__none__'}
            onValueChange={(v) => onChange({ ...iface, vrf: v === '__none__' ? undefined : v })}
          >
            <SelectTrigger className="font-mono">
              <SelectValue placeholder="- none -" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">- none -</SelectItem>
              {vrfNames.map((n) => (
                <SelectItem key={n} value={n}>{n}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </ComboRow>}
      </PreferencesGroup>

      {isTunnel && (
        <PreferencesGroup title="Tunnel">
          <ComboRow title="Mode">
            <Select value={iface.mode || 'gre'} onValueChange={(v) => set('mode', v)}>
              <SelectTrigger className="font-mono"><SelectValue /></SelectTrigger>
              <SelectContent>
                {TUNNEL_MODES.map((m) => <SelectItem key={m} value={m} className="font-mono">{m}</SelectItem>)}
              </SelectContent>
            </Select>
          </ComboRow>
          <EntryRow title="TTL (0 = inherit)" value={iface.ttl || ''} onChange={(e) => set('ttl', Number(e.target.value) || 0)} placeholder="0" className="font-mono" />
          <EntryRow title="Local Endpoint" value={iface.local ?? ''} onChange={(e) => set('local', e.target.value)} placeholder="203.0.113.1" className="font-mono" />
          <EntryRow title="Remote Endpoint" value={iface.remote ?? ''} onChange={(e) => set('remote', e.target.value)} placeholder="203.0.113.2" className="font-mono" />
        </PreferencesGroup>
      )}

      {isVxlan && (
        <PreferencesGroup title="VXLAN">
          <div className="flex min-h-[3.25rem] items-center gap-3 px-4 py-2.5">
            <div className="min-w-0 flex-1">
              <div className="text-sm font-medium">External (collect metadata)</div>
              <div className="mt-0.5 text-xs text-muted-foreground">Receive every VNI on the UDP port; the VNI comes from tunnel metadata. Only one per port unless VNI filter is on.</div>
            </div>
            <Switch checked={!!iface.vxlan?.external} onCheckedChange={(v) => setVxlan(v ? { external: true, vni: 0 } : { external: undefined, vnifilter: undefined })} />
          </div>
          {iface.vxlan?.external && (
            <div className="flex min-h-[3.25rem] items-center gap-3 px-4 py-2.5">
              <div className="min-w-0 flex-1">
                <div className="text-sm font-medium">VNI filter</div>
                <div className="mt-0.5 text-xs text-muted-foreground">Let several external devices share the UDP port, each filtering its own VNIs</div>
              </div>
              <Switch checked={!!iface.vxlan?.vnifilter} onCheckedChange={(v) => setVxlan({ vnifilter: v || undefined })} />
            </div>
          )}
          {!iface.vxlan?.external && (
            <EntryRow
              title="VNI"
              inputMode="numeric"
              value={iface.vxlan?.vni || ''}
              onChange={(e) => setVxlan({ vni: Number(e.target.value) || 0 })}
              placeholder="100"
              className="font-mono"
              error={!iface.vxlan?.vni || iface.vxlan.vni < 1 || iface.vxlan.vni > 16777215 ? 'VNI must be 1 to 16777215' : null}
            />
          )}
          <EntryRow title="Local Address" value={iface.vxlan?.local ?? ''} onChange={(e) => setVxlan({ local: e.target.value || undefined })} placeholder="192.0.2.1" className="font-mono" error={checkIP(iface.vxlan?.local ?? '')} />
          <EntryRow title="Remote Address" value={iface.vxlan?.remote ?? ''} onChange={(e) => setVxlan({ remote: e.target.value || undefined })} placeholder="192.0.2.2" className="font-mono" error={iface.vxlan?.group ? 'Remote and group are mutually exclusive' : checkIP(iface.vxlan?.remote ?? '')} />
          <EntryRow title="Multicast Group" value={iface.vxlan?.group ?? ''} onChange={(e) => setVxlan({ group: e.target.value || undefined })} placeholder="239.1.1.1" className="font-mono" error={iface.vxlan?.remote ? 'Remote and group are mutually exclusive' : checkMulticastIP(iface.vxlan?.group ?? '')} />
          <EntryRow title="VTEP Interface" value={iface.vxlan?.vtep ?? ''} onChange={(e) => setVxlan({ vtep: e.target.value || undefined })} placeholder="underlay" className="font-mono" />
          <EntryRow title="UDP Port (0 = 4789)" inputMode="numeric" value={iface.vxlan?.port || ''} onChange={(e) => setVxlan({ port: Number(e.target.value) || undefined })} placeholder="4789" className="font-mono" error={checkPort(String(iface.vxlan?.port || ''))} />
          <div className="flex min-h-[3.25rem] items-center gap-3 px-4 py-2.5">
            <div className="min-w-0 flex-1">
              <div className="text-sm font-medium">MAC learning</div>
              <div className="mt-0.5 text-xs text-muted-foreground">Enabled by default; EVPN fabrics commonly disable it</div>
            </div>
            <Switch checked={iface.vxlan?.learning !== false} onCheckedChange={(v) => setVxlan({ learning: v })} />
          </div>
        </PreferencesGroup>
      )}

      {isVxlanBridge && (
        <PreferencesGroup title="VXLAN Bridge">
          <EntryRow title="Local VTEP Address" value={iface.vxlan?.local ?? ''} onChange={(e) => setVxlan({ local: e.target.value || undefined })} className="font-mono" error={checkIP(iface.vxlan?.local ?? '')} />
          <EntryRow title="Underlay Interface" value={iface.vxlan?.vtep ?? ''} onChange={(e) => setVxlan({ vtep: e.target.value || undefined })} className="font-mono" />
          <EntryRow title="UDP Port" inputMode="numeric" value={iface.vxlan?.port || ''} onChange={(e) => setVxlan({ port: Number(e.target.value) || undefined })} placeholder="4789" className="font-mono" />
        </PreferencesGroup>
      )}

      {isVnet && (
        <PreferencesGroup title="VNet">
          <EntryRow title="VNI" inputMode="numeric" value={iface.vnet?.vni || ''} onChange={(e) => set('vnet', { vlan: iface.vnet?.vlan ?? 0, vni: Number(e.target.value) || 0 })} className="font-mono" />
          <EntryRow title="Transport VLAN" inputMode="numeric" value={iface.vnet?.vlan || ''} onChange={(e) => set('vnet', { vni: iface.vnet?.vni ?? 0, vlan: Number(e.target.value) || 0 })} className="font-mono" error={checkVLANId(String(iface.vnet?.vlan || ''))} />
        </PreferencesGroup>
      )}

      {isBond && (
        <PreferencesGroup title="Bond">
          <div className="px-4 py-3 space-y-1.5">
            <Label className="text-xs">Members</Label>
            <MemberPicker members={iface.bond?.members ?? []} nics={nics} extra={parentOptions} onChange={(v) => setBond({ members: v })} />
          </div>
          <ComboRow title="Mode">
            <Select value={iface.bond?.mode || 'balance-rr'} onValueChange={(v) => setBond({ mode: v })}>
              <SelectTrigger className="font-mono"><SelectValue /></SelectTrigger>
              <SelectContent>
                {BOND_MODES.map((m) => <SelectItem key={m} value={m} className="font-mono">{m}</SelectItem>)}
              </SelectContent>
            </Select>
          </ComboRow>
          <EntryRow title="MII Monitor (ms, 0 = off)" inputMode="numeric" value={iface.bond?.miimon || ''} onChange={(e) => setBond({ miimon: Number(e.target.value) || undefined })} placeholder="100" className="font-mono" />
          {(iface.bond?.mode === 'balance-xor' || iface.bond?.mode === '802.3ad') && (
            <ComboRow title="Transmit Hash Policy">
              <Select value={iface.bond?.xmit_hash_policy || 'layer2'} onValueChange={(v) => setBond({ xmit_hash_policy: v })}>
                <SelectTrigger className="font-mono"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {BOND_HASH_POLICIES.map((p) => <SelectItem key={p} value={p} className="font-mono">{p}</SelectItem>)}
                </SelectContent>
              </Select>
            </ComboRow>
          )}
          {iface.bond?.mode === '802.3ad' && (
            <ComboRow title="LACP Rate">
              <Select value={iface.bond?.lacp_rate || 'slow'} onValueChange={(v) => setBond({ lacp_rate: v })}>
                <SelectTrigger className="font-mono"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {BOND_LACP_RATES.map((r) => <SelectItem key={r} value={r} className="font-mono">{r}</SelectItem>)}
                </SelectContent>
              </Select>
            </ComboRow>
          )}
          {iface.bond?.mode === '802.3ad' && (
            <EntryRow title="Minimum Links" inputMode="numeric" value={iface.bond?.min_links || ''} onChange={(e) => setBond({ min_links: Number(e.target.value) || undefined })} placeholder="0" className="font-mono" />
          )}
          <EntryRow title="Up Delay (ms)" inputMode="numeric" value={iface.bond?.updelay || ''} onChange={(e) => setBond({ updelay: Number(e.target.value) || undefined })} placeholder="0" className="font-mono" />
          <EntryRow title="Down Delay (ms)" inputMode="numeric" value={iface.bond?.downdelay || ''} onChange={(e) => setBond({ downdelay: Number(e.target.value) || undefined })} placeholder="0" className="font-mono" />
          {(iface.bond?.mode === 'active-backup' || iface.bond?.mode === 'balance-tlb' || iface.bond?.mode === 'balance-alb') && (
            <ComboRow title="Primary Member">
              <Select value={iface.bond?.primary || '__none__'} onValueChange={(v) => setBond({ primary: v === '__none__' ? undefined : v })}>
                <SelectTrigger className="font-mono">
                  <SelectValue placeholder="- none -" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__" className="font-mono">- none -</SelectItem>
                  {(iface.bond?.members ?? []).map((m) => (
                    <SelectItem key={m} value={m} className="font-mono">{selectorLabel(nics, m)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </ComboRow>
          )}
        </PreferencesGroup>
      )}

      {isVlan && (
        <PreferencesGroup title="VLAN">
          <EntryRow
            title="VLAN ID"
            inputMode="numeric"
            value={iface.vlan?.id || ''}
            onChange={(e) => onChange({ ...iface, vlan: { id: Number(e.target.value) || 0 } })}
            placeholder="100"
            className="font-mono"
            error={checkVLANId(String(iface.vlan?.id || ''))}
          />
        </PreferencesGroup>
      )}

      {!isVxlanBridge && <PreferencesGroup title="Addresses">
        <div className="px-4 py-3">
          <TagInput values={iface.addresses ?? []} onChange={(v) => set('addresses', v)} placeholder="192.168.1.1/24, dhcp, slaac" mono validate={checkInterfaceAddress} />
        </div>
      </PreferencesGroup>}

      {!isTunnel && !isVxlanBridge && (
      <PreferencesGroup title="DHCP Options">
        <div className="px-4 py-3">
          <TagInput values={iface.dhcp_options ?? []} onChange={(v) => set('dhcp_options', v)} placeholder="option:dns-server,8.8.8.8" />
        </div>
      </PreferencesGroup>
      )}

      {!isDummy && !isTunnel && !isBond && !isVlan && !isVxlan && !isVxlanBridge && (
      <div className="rounded-md bg-muted/30 px-3 py-2">
        <div className="flex items-center gap-2 py-2">
          <Button variant="ghost" size="sm" aria-expanded={bridgeOpen} onClick={() => setBridgeOpen(!bridgeOpen)} className="gap-1.5 font-semibold">
            {bridgeOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            Bridge
          </Button>
          <Button
            type="button"
            variant={hasBridge ? 'destructive' : 'outline'}
            size="sm"
            className="ml-auto h-7 text-xs"
            onClick={toggleBridge}
          >
            {hasBridge ? 'Remove Bridge' : 'Add Bridge'}
          </Button>
        </div>
        {bridgeOpen && !hasBridge && (
          <p className="text-sm text-muted-foreground py-1">No bridge configured</p>
        )}
        {bridgeOpen && hasBridge && (
          <div className="mt-2 space-y-3">
            <div className="space-y-1.5">
              <Label className="text-xs">Members</Label>
              <MemberPicker members={iface.bridge!.members} nics={nics} extra={parentOptions} onChange={(v) => set('bridge', { ...iface.bridge!, members: v })} />
            </div>
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm">Enable STP (Spanning Tree Protocol)</span>
              <Switch checked={iface.bridge!.stp} onCheckedChange={(v) => set('bridge', { ...iface.bridge!, stp: v })} />
            </div>
          </div>
        )}
      </div>
      )}

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Close'}</Button>
        {onAdd && <Button onClick={onAdd}>Add Interface</Button>}
      </div>
    </div>
  )
}
