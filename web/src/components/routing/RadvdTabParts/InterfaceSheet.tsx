import { numOrUndef, PREFERENCES, PrefixRow, RADVDInterface, RADVDPrefix, RADVDRoute, RouteRow } from '@/components/routing/RadvdTab'
import { VarPickerInput } from '@/components/VarPickerInput'
import { checkIP } from '@/lib/validate'
import { Button, ComboRow, EntryRow, Label, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, SwitchRow, TagInput } from 'cheval-ui'
import { Plus } from 'lucide-react'

type InterfaceSheetShape = {
  name: string
  iface: RADVDInterface
  onChangeName: (n: string) => void
  onChange: (i: RADVDInterface) => void
  onClose: () => void
  onAdd?: () => void
  ifaceNames: string[]
}

export function InterfaceSheet({
  name,
  iface,
  onChangeName,
  onChange,
  onClose,
  onAdd,
  ifaceNames,
}: InterfaceSheetShape) {
  const set = <K extends keyof RADVDInterface>(k: K, v: RADVDInterface[K]) =>
    onChange({ ...iface, [k]: v })

  const prefixes = iface.prefixes ?? []
  const routes = iface.routes ?? []

  const dhcp6Managed =
    !!iface.adv_managed_flag &&
    !!iface.adv_other_config_flag &&
    prefixes.length > 0 &&
    prefixes.every((p) => p.adv_on_link && !p.adv_autonomous)

  const setStatefulDhcp6 = (on: boolean) => {
    if (on) {
      onChange({
        ...iface,
        adv_send_advert: true,
        adv_managed_flag: true,
        adv_other_config_flag: true,
        prefixes: prefixes.map((p) => ({ ...p, adv_on_link: true, adv_autonomous: false })),
      })
    } else {
      onChange({ ...iface, adv_managed_flag: undefined, adv_other_config_flag: undefined })
    }
  }

  const addPrefix = () =>
    set('prefixes', [...prefixes, { prefix: '', adv_on_link: true, adv_autonomous: !iface.adv_managed_flag }])
  const updatePrefix = (i: number, p: RADVDPrefix) =>
    set('prefixes', prefixes.map((x, j) => (j === i ? p : x)))
  const deletePrefix = (i: number) =>
    set('prefixes', prefixes.filter((_, j) => j !== i))

  const addRoute = () =>
    set('routes', [...routes, { prefix: '' }])
  const updateRoute = (i: number, r: RADVDRoute) =>
    set('routes', routes.map((x, j) => (j === i ? r : x)))
  const deleteRoute = (i: number) =>
    set('routes', routes.filter((_, j) => j !== i))

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        <ComboRow title="Interface name">
          <VarPickerInput
            value={name}
            onChange={onChangeName}
            vars={ifaceNames}
            placeholder="interface name"
            prefix=""
            label="Interfaces"
            mono
          />
        </ComboRow>
        <SwitchRow
          title="Send router advertisements"
          checked={!!iface.adv_send_advert}
          onCheckedChange={(v) => set('adv_send_advert', v || undefined)}
        />
      </PreferencesGroup>

      <PreferencesGroup title="Advertisement">
        <EntryRow title="Min interval (s)" value={iface.min_rtr_adv_interval ?? ''} onChange={(e) => set('min_rtr_adv_interval', numOrUndef(e.target.value))} placeholder="200" className="font-mono" />
        <EntryRow title="Max interval (s)" value={iface.max_rtr_adv_interval ?? ''} onChange={(e) => set('max_rtr_adv_interval', numOrUndef(e.target.value))} placeholder="600" className="font-mono" />
        <EntryRow title="Default lifetime (s)" value={iface.adv_default_lifetime ?? ''} onChange={(e) => set('adv_default_lifetime', numOrUndef(e.target.value))} placeholder="1800" className="font-mono" />
        <ComboRow title="Default preference">
          <Select value={iface.adv_default_preference ?? '_none'} onValueChange={(v) => set('adv_default_preference', v === '_none' ? undefined : v)}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="_none">default</SelectItem>
              {PREFERENCES.map((p) => <SelectItem key={p} value={p}>{p}</SelectItem>)}
            </SelectContent>
          </Select>
        </ComboRow>
        <EntryRow title="Link MTU" value={iface.adv_link_mtu ?? ''} onChange={(e) => set('adv_link_mtu', numOrUndef(e.target.value))} placeholder="1500" className="font-mono" />
        <div className="px-4 py-3 space-y-1.5">
          <Label>RA source address</Label>
          <p className="text-xs text-muted-foreground">Link-local addresses radvd sends advertisements from, in order of preference. Use a VRRP virtual link-local address so the RA source fails over with the router.</p>
          <TagInput
            values={iface.adv_ra_src_address ?? []}
            onChange={(v) => set('adv_ra_src_address', v.length ? v : undefined)}
            placeholder="fe80::1"
            validate={checkIP}
            mono
          />
        </div>
        <SwitchRow
          title="Stateful DHCPv6"
          subtitle="Advertise Managed + Other flags and mark prefixes non-autonomous so hosts get addresses from the DHCPv6 server."
          checked={dhcp6Managed}
          onCheckedChange={setStatefulDhcp6}
        />
        <SwitchRow title="Managed (M flag)" checked={!!iface.adv_managed_flag} onCheckedChange={(v) => set('adv_managed_flag', v || undefined)} />
        <SwitchRow title="Other config (O flag)" checked={!!iface.adv_other_config_flag} onCheckedChange={(v) => set('adv_other_config_flag', v || undefined)} />
      </PreferencesGroup>

      <PreferencesGroup
        title="Prefixes"
        description="Address prefixes advertised to clients."
        header={
          <Button type="button" variant="outline" size="sm" onClick={addPrefix} className="gap-1.5">
            <Plus className="h-4 w-4" />Add
          </Button>
        }
      >
        {prefixes.length === 0 && (
          <p className="px-4 py-3 text-xs text-muted-foreground italic">No prefixes, clients will not receive address info.</p>
        )}
        {prefixes.map((p, i) => (
          <div key={i} className="px-4 py-3">
            <PrefixRow prefix={p} onChange={(v) => updatePrefix(i, v)} onDelete={() => deletePrefix(i)} />
          </div>
        ))}
      </PreferencesGroup>

      <PreferencesGroup title="RDNSS" description="Recursive DNS servers advertised to clients.">
        <div className="px-4 py-3">
          <TagInput
            values={iface.rdnss?.servers ?? []}
            onChange={(v) => set('rdnss', v.length ? { ...iface.rdnss, servers: v } : undefined)}
            placeholder="2001:db8::53"
            validate={checkIP}
            mono
          />
        </div>
        <EntryRow
          title="Lifetime (s)"
          value={iface.rdnss?.lifetime ?? ''}
          onChange={(e) => {
            const lifetime = numOrUndef(e.target.value)
            set('rdnss', iface.rdnss?.servers?.length ? { ...iface.rdnss, lifetime } : undefined)
          }}
          placeholder="600"
          className="font-mono"
        />
      </PreferencesGroup>

      <PreferencesGroup
        title="Advertised routes"
        header={
          <Button type="button" variant="outline" size="sm" onClick={addRoute} className="gap-1.5">
            <Plus className="h-4 w-4" />Add
          </Button>
        }
      >
        {routes.length === 0 && (
          <p className="px-4 py-3 text-xs text-muted-foreground italic">No extra routes advertised.</p>
        )}
        {routes.map((r, i) => (
          <div key={i} className="px-4 py-3">
            <RouteRow route={r} onChange={(v) => updateRoute(i, v)} onDelete={() => deleteRoute(i)} />
          </div>
        ))}
      </PreferencesGroup>

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onClose}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add Interface</Button>}
      </div>
    </div>
  )
}
