import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Sheet } from '@/components/ui/sheet'
import { Badge } from '@/components/ui/badge'
import TagInput from '@/components/TagInput'
import { Plus, Trash2, Radio } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { PreferencesGroup, PreferencesColumns, EntryRow, ComboRow, SwitchRow } from '@/components/Preferences'
import { VarPickerInput } from '@/components/VarPickerInput'
import { checkIP } from '@/lib/validate'
import { EmptyState } from '@/components/EmptyState'

export interface RADVDPrefix {
  prefix: string
  adv_on_link?: boolean
  adv_autonomous?: boolean
  adv_router_addr?: boolean
  adv_valid_lifetime?: string
  adv_preferred_lifetime?: string
}

export interface RADVDRDNSS {
  servers?: string[]
  lifetime?: number
}

export interface RADVDRoute {
  prefix: string
  lifetime?: number
  preference?: string
}

export interface RADVDInterface {
  adv_send_advert?: boolean
  min_rtr_adv_interval?: number
  max_rtr_adv_interval?: number
  adv_managed_flag?: boolean
  adv_other_config_flag?: boolean
  adv_default_lifetime?: number
  adv_default_preference?: string
  adv_link_mtu?: number
  prefixes?: RADVDPrefix[]
  rdnss?: RADVDRDNSS
  routes?: RADVDRoute[]
}

export interface RADVDConfig {
  interfaces?: Record<string, RADVDInterface>
}

const PREFERENCES = ['low', 'medium', 'high']

function numOrUndef(v: string): number | undefined {
  const n = parseInt(v, 10)
  return isNaN(n) || v === '' ? undefined : n
}

function PrefixRow({
  prefix,
  onChange,
  onDelete,
}: {
  prefix: RADVDPrefix
  onChange: (p: RADVDPrefix) => void
  onDelete: () => void
}) {
  const set = <K extends keyof RADVDPrefix>(k: K, v: RADVDPrefix[K]) =>
    onChange({ ...prefix, [k]: v })

  return (
    <div className="border rounded-md p-3 space-y-3 bg-background">
      <div className="flex items-center gap-2">
        <Input
          value={prefix.prefix}
          onChange={(e) => set('prefix', e.target.value)}
          placeholder="2001:db8::/64"
          className="font-mono text-xs h-8 flex-1"
        />
        <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0 hover:text-destructive" onClick={onDelete}>
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-2">
        {(
          [
            ['adv_on_link', 'On-link (L)'],
            ['adv_autonomous', 'Autonomous (A)'],
            ['adv_router_addr', 'Router addr (R)'],
          ] as const
        ).map(([key, label]) => (
          <div key={key} className="flex items-center justify-between gap-2">
            <span className="text-xs text-muted-foreground">{label}</span>
            <Switch checked={!!prefix[key]} onCheckedChange={(v) => set(key, v || undefined)} />
          </div>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <Label className="text-[11px]">Valid lifetime</Label>
          <Input
            value={prefix.adv_valid_lifetime ?? ''}
            onChange={(e) => set('adv_valid_lifetime', e.target.value || undefined)}
            placeholder="86400 or infinity"
            className="font-mono text-xs h-8"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-[11px]">Preferred lifetime</Label>
          <Input
            value={prefix.adv_preferred_lifetime ?? ''}
            onChange={(e) => set('adv_preferred_lifetime', e.target.value || undefined)}
            placeholder="14400 or infinity"
            className="font-mono text-xs h-8"
          />
        </div>
      </div>
    </div>
  )
}

function RouteRow({
  route,
  onChange,
  onDelete,
}: {
  route: RADVDRoute
  onChange: (r: RADVDRoute) => void
  onDelete: () => void
}) {
  return (
    <div className="flex items-center gap-2">
      <Input
        value={route.prefix}
        onChange={(e) => onChange({ ...route, prefix: e.target.value })}
        placeholder="2001:db8::/48"
        className="font-mono text-xs h-8 flex-1"
      />
      <Select value={route.preference ?? '_none'} onValueChange={(v) => onChange({ ...route, preference: v === '_none' ? undefined : v })}>
        <SelectTrigger className="h-8 text-xs w-28">
          <SelectValue placeholder="preference" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="_none">preference</SelectItem>
          {PREFERENCES.map((p) => <SelectItem key={p} value={p}>{p}</SelectItem>)}
        </SelectContent>
      </Select>
      <Input
        value={route.lifetime ?? ''}
        onChange={(e) => onChange({ ...route, lifetime: numOrUndef(e.target.value) })}
        placeholder="lifetime"
        className="font-mono text-xs h-8 w-24"
      />
      <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0 hover:text-destructive" onClick={onDelete}>
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

function InterfaceSheet({
  name,
  iface,
  onChangeName,
  onChange,
  onClose,
  onAdd,
  ifaceNames,
}: {
  name: string
  iface: RADVDInterface
  onChangeName: (n: string) => void
  onChange: (i: RADVDInterface) => void
  onClose: () => void
  onAdd?: () => void
  ifaceNames: string[]
}) {
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

const NEW_KEY = '__new__'

export default function RadvdTab({
  radvd,
  onChange,
  onDirty,
  ifaceNames,
}: {
  radvd: RADVDConfig | null
  onChange: (v: RADVDConfig | null) => void
  onDirty: () => void
  ifaceNames: string[]
}) {
  const [openIface, setOpenIface] = useState<string | null>(null)
  const [pendingName, setPendingName] = useState('')
  const [draftIface, setDraftIface] = useState<RADVDInterface | null>(null)

  const ifaces = radvd?.interfaces ?? {}

  const updateAll = (next: Record<string, RADVDInterface>) => {
    onChange(Object.keys(next).length ? { interfaces: next } : null)
    onDirty()
  }

  const deleteIface = (name: string) => {
    const next = { ...ifaces }
    delete next[name]
    updateAll(next)
    if (openIface === name) setOpenIface(null)
  }

  const handleAddNew = () => {
    setDraftIface({ adv_send_advert: true })
    setPendingName('')
    setOpenIface(NEW_KEY)
  }

  const handleCommitNew = () => {
    const n = pendingName.trim()
    if (!n) { toast.error('Please enter an interface name'); return }
    if (n in ifaces) { toast.error(`Interface "${n}" already exists`); return }
    updateAll({ ...ifaces, [n]: draftIface ?? { adv_send_advert: true } })
    setDraftIface(null)
    setOpenIface(null)
  }

  const handleCloseSheet = () => { setOpenIface(null); setDraftIface(null) }

  const openIfaceData = openIface === NEW_KEY ? draftIface : (openIface ? ifaces[openIface] : null)
  const sheetTitle = openIface === NEW_KEY ? 'Add Interface' : (pendingName || openIface || 'Interface')

  const rows = Object.entries(ifaces)

  return (
    <div className="space-y-4">
      {rows.length === 0 ? (
        <EmptyState
          className="max-w-2xl mx-auto"
          icon={<Radio />}
          title="No RADVD interfaces"
          message="Add an interface to start advertising IPv6 prefixes."
          action={
            <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
              <Plus className="h-3.5 w-3.5" />Add interface
            </Button>
          }
        />
      ) : (
        <PreferencesColumns
          title="Interfaces"
          header={
            <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
              <Plus className="h-4 w-4" />Add
            </Button>
          }
        >
          {rows.map(([name, iface]) => {
            const prefixCount = iface.prefixes?.length ?? 0
            return (
              <div
                key={name}
                onClick={() => { setPendingName(name); setOpenIface(name) }}
                className="flex items-center gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors"
              >
              <span className="font-mono font-semibold text-sm w-32 shrink-0 truncate">{name}</span>
              <div className="flex flex-wrap gap-1 flex-1 min-w-0">
                <Badge variant={iface.adv_send_advert ? 'default' : 'secondary'} className="text-xs">
                  {iface.adv_send_advert ? 'RA on' : 'RA off'}
                </Badge>
                {prefixCount > 0 && (
                  <Badge variant="outline" className="text-xs font-mono">
                    {prefixCount} prefix{prefixCount !== 1 ? 'es' : ''}
                  </Badge>
                )}
                {iface.rdnss?.servers?.length ? (
                  <Badge variant="outline" className="text-xs">RDNSS</Badge>
                ) : null}
                {(iface.routes?.length ?? 0) > 0 && (
                  <Badge variant="outline" className="text-xs">{iface.routes!.length} route{iface.routes!.length !== 1 ? 's' : ''}</Badge>
                )}
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={(e) => { e.stopPropagation(); deleteIface(name) }}
                className="h-7 w-7 p-0 hover:text-destructive shrink-0"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
              </div>
            )
          })}
        </PreferencesColumns>
      )}

      <Sheet open={openIface !== null} onClose={handleCloseSheet} title={sheetTitle}>
        {openIface !== null && openIfaceData !== null && (
          <InterfaceSheet
            name={pendingName}
            iface={openIfaceData!}
            ifaceNames={ifaceNames}
            onChangeName={setPendingName}
            onChange={openIface === NEW_KEY ? setDraftIface : (u) => updateAll({ ...ifaces, [openIface!]: u })}
            onClose={handleCloseSheet}
            onAdd={openIface === NEW_KEY ? handleCommitNew : undefined}
          />
        )}
      </Sheet>
    </div>
  )
}
