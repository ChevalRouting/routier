import { useState } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { useDataRefresh } from '@/lib/dataVersion'
import { usePageSave } from '@/lib/usePageSave'
import SaveButton from '@/components/SaveButton'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Sheet } from '@/components/ui/sheet'
import TagInput from '@/components/TagInput'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import {
  Plus, Trash2, ChevronDown, ChevronRight, Network,
} from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { EmptyState } from '@/components/EmptyState'
import { Pagination, usePagination } from '@/components/Pagination'
import { checkCIDR, checkIPOrCIDR } from '@/lib/validate'
import { ReloadButton } from '@/components/ReloadButton'
import { PreferencesGroup, PreferencesColumns, EntryRow, ComboRow } from '@/components/Preferences'
import { Switch } from '@/components/ui/switch'
import { Spinner } from '@/components/Spinner'

interface Vlan {
  id: number
  addresses: string[]
  mtu: number
}

interface Bridge {
  members: string[]
  stp: boolean
}

interface VrrpInstance {
  name?: string
  id: number
  vips: string[]
  priority: number
  password: string
  interface: string
  switchover?: boolean
  allow_inbound?: boolean
}

interface Iface {
  select: string
  type?: string
  vrf?: string
  addresses: string[]
  dhcp_options: string[]
  mtu: number
  vlans?: Record<string, Vlan>
  bridge?: Bridge
  vrrp?: VrrpInstance[]
}

const IFACE_TYPES = [
  { value: 'physical', label: 'Physical (select device)' },
  { value: 'dummy', label: 'Dummy (virtual)' },
  { value: 'bridge', label: 'Bridge (virtual)' },
]

type IfaceMap = Record<string, Iface>

function emptyIface(): Iface {
  return { select: '', addresses: [], dhcp_options: [], mtu: 0 }
}

function emptyVlan(): Vlan {
  return { id: 0, addresses: [], mtu: 0 }
}

function emptyVrrp(): VrrpInstance {
  return { name: '', id: 0, vips: [], priority: 100, password: '', interface: '' }
}

interface SectionHeaderProps {
  title: string
  expanded: boolean
  onToggle: () => void
  action?: React.ReactNode
}

function SectionHeader({ title, expanded, onToggle, action }: SectionHeaderProps) {
  return (
    <div className="flex items-center gap-2 py-2">
      <button type="button" onClick={onToggle} className="flex items-center gap-1 text-sm font-semibold text-foreground hover:text-primary">
        {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        {title}
      </button>
      {action}
    </div>
  )
}

interface IfaceFormProps {
  name: string
  iface: Iface
  vrfNames: string[]
  onChange: (updated: Iface) => void
  onNameChange?: (n: string) => void
  onAdd?: () => void
  onDone: () => void
}

function IfaceForm({ name, iface, vrfNames, onChange, onNameChange, onAdd, onDone }: IfaceFormProps) {
  const [vlansOpen, setVlansOpen] = useState(false)
  const [bridgeOpen, setBridgeOpen] = useState(false)
  const [vrrpOpen, setVrrpOpen] = useState(false)
  const [pendingVlanNames, setPendingVlanNames] = useState<Record<string, string>>({})
  const [addingVlan, setAddingVlan] = useState(false)
  const [newVlanName, setNewVlanName] = useState('')

  const isVirtual = iface.type === 'dummy' || iface.type === 'bridge'
  const isDummy = iface.type === 'dummy'

  const set = <K extends keyof Iface>(key: K, val: Iface[K]) =>
    onChange({ ...iface, [key]: val })

  const setVlan = (vname: string, vlan: Vlan) =>
    onChange({ ...iface, vlans: { ...(iface.vlans ?? {}), [vname]: vlan } })

  const deleteVlan = (vname: string) => {
    const next = { ...(iface.vlans ?? {}) }
    delete next[vname]
    setPendingVlanNames((p) => { const n = { ...p }; delete n[vname]; return n })
    onChange({ ...iface, vlans: Object.keys(next).length ? next : undefined })
  }

  const confirmAddVlan = () => {
    const n = newVlanName.trim()
    if (!n) return
    const vlans = iface.vlans ?? {}
    if (vlans[n]) return
    onChange({ ...iface, vlans: { ...vlans, [n]: emptyVlan() } })
    setNewVlanName('')
    setAddingVlan(false)
  }

  const commitVlanRename = (oldName: string) => {
    const newName = (pendingVlanNames[oldName] ?? oldName).trim()
    setPendingVlanNames((p) => { const n = { ...p }; delete n[oldName]; return n })
    if (!newName || newName === oldName) return
    const vlans = iface.vlans ?? {}
    if (vlans[newName]) return
    const next: Record<string, Vlan> = {}
    for (const [k, v] of Object.entries(vlans)) {
      next[k === oldName ? newName : k] = v
    }
    onChange({ ...iface, vlans: next })
  }

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

  const addVrrp = () => {
    onChange({ ...iface, vrrp: [...(iface.vrrp ?? []), emptyVrrp()] })
    setVrrpOpen(true)
  }

  const setVrrpField = <K extends keyof VrrpInstance>(idx: number, key: K, val: VrrpInstance[K]) => {
    const list = [...(iface.vrrp ?? [])]
    list[idx] = { ...list[idx], [key]: val }
    onChange({ ...iface, vrrp: list })
  }

  const deleteVrrp = (idx: number) => {
    const list = (iface.vrrp ?? []).filter((_, i) => i !== idx)
    onChange({ ...iface, vrrp: list.length ? list : undefined })
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
            onValueChange={(v) => onChange({ ...iface, type: v === 'physical' ? undefined : v })}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {IFACE_TYPES.map((t) => (
                <SelectItem key={t.value} value={t.value}>{t.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </ComboRow>
        {!isVirtual && (
          <EntryRow
            title="Selector"
            value={iface.select}
            onChange={(e) => set('select', e.target.value)}
            placeholder="mac:aa:bb:cc:dd:ee"
            className="font-mono"
          />
        )}
        <EntryRow
          title="MTU (0 = auto)"
          value={iface.mtu || ''}
          onChange={(e) => set('mtu', Number(e.target.value) || 0)}
          placeholder="0"
          className="font-mono"
        />
        <ComboRow title="VRF">
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
        </ComboRow>
      </PreferencesGroup>

      <PreferencesGroup title="Addresses">
        <div className="px-4 py-3">
          <TagInput values={iface.addresses ?? []} onChange={(v) => set('addresses', v)} placeholder="192.168.1.1/24" mono validate={checkCIDR} />
        </div>
      </PreferencesGroup>

      <PreferencesGroup title="DHCP options">
        <div className="px-4 py-3">
          <TagInput values={iface.dhcp_options ?? []} onChange={(v) => set('dhcp_options', v)} placeholder="option:dns-server,8.8.8.8" />
        </div>
      </PreferencesGroup>

      <div className="border rounded-md px-3 py-2">
        <SectionHeader
          title="VLANs"
          expanded={vlansOpen}
          onToggle={() => setVlansOpen(!vlansOpen)}
          action={
            <Button type="button" variant="outline" size="sm" className="ml-auto h-7 gap-1 text-xs"
              onClick={() => { setAddingVlan(true); setVlansOpen(true) }}>
              <Plus className="h-3 w-3" />Add VLAN
            </Button>
          }
        />
        {vlansOpen && (
          <div className="mt-2 space-y-3">
            {addingVlan && (
              <div className="flex items-center gap-2 border rounded p-2 bg-muted/30">
                <Input
                  autoFocus
                  value={newVlanName}
                  onChange={(e) => setNewVlanName(e.target.value)}
                  onKeyDown={(e) => { if (e.key === 'Enter') confirmAddVlan(); if (e.key === 'Escape') { setAddingVlan(false); setNewVlanName('') } }}
                  placeholder="VLAN name (servers)"
                  className="font-mono text-sm h-8 flex-1"
                />
                <Button size="sm" className="h-8" onClick={confirmAddVlan}>Add</Button>
                <Button size="sm" variant="ghost" className="h-8" onClick={() => { setAddingVlan(false); setNewVlanName('') }}>Cancel</Button>
              </div>
            )}
            {Object.keys(iface.vlans ?? {}).length === 0 && !addingVlan && (
              <p className="text-sm text-muted-foreground italic py-1">No VLANs configured.</p>
            )}
            {Object.entries(iface.vlans ?? {}).map(([vname, vlan]) => (
              <div key={vname} className="border rounded p-3 space-y-3 bg-muted/30">
                <div className="flex items-center justify-between gap-2">
                  <Input
                    value={pendingVlanNames[vname] ?? vname}
                    onChange={(e) => setPendingVlanNames((p) => ({ ...p, [vname]: e.target.value }))}
                    onBlur={() => commitVlanRename(vname)}
                    onKeyDown={(e) => { if (e.key === 'Enter') e.currentTarget.blur() }}
                    className="font-mono text-sm font-medium h-7 border-0 shadow-none focus-visible:ring-1 px-1 w-40"
                    placeholder="vlan-name"
                  />
                  <Button variant="ghost" size="sm" onClick={() => deleteVlan(vname)} className="h-7 w-7 p-0 hover:text-destructive shrink-0">
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label className="text-xs">VLAN ID</Label>
                    <Input value={vlan.id || ''} onChange={(e) => setVlan(vname, { ...vlan, id: Number(e.target.value) || 0 })} placeholder="100" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">MTU <span className="text-muted-foreground">(0=auto)</span></Label>
                    <Input value={vlan.mtu || ''} onChange={(e) => setVlan(vname, { ...vlan, mtu: Number(e.target.value) || 0 })} placeholder="0" />
                  </div>
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Addresses</Label>
                  <TagInput values={vlan.addresses} onChange={(v) => setVlan(vname, { ...vlan, addresses: v })} placeholder="10.0.0.1/24" mono validate={checkCIDR} />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {!isDummy && (
      <div className="border rounded-md px-3 py-2">
        <div className="flex items-center gap-2 py-2">
          <button type="button" onClick={() => setBridgeOpen(!bridgeOpen)} className="flex items-center gap-1 text-sm font-semibold text-foreground hover:text-primary">
            {bridgeOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            Bridge
          </button>
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
          <p className="text-sm text-muted-foreground italic py-1">No bridge configured.</p>
        )}
        {bridgeOpen && hasBridge && (
          <div className="mt-2 space-y-3">
            <div className="space-y-1.5">
              <Label className="text-xs">Members</Label>
              <TagInput values={iface.bridge!.members} onChange={(v) => set('bridge', { ...iface.bridge!, members: v })} placeholder="eth0" mono />
            </div>
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm">Enable STP (Spanning Tree Protocol)</span>
              <Switch checked={iface.bridge!.stp} onCheckedChange={(v) => set('bridge', { ...iface.bridge!, stp: v })} />
            </div>
          </div>
        )}
      </div>
      )}

      <div className="border rounded-md px-3 py-2">
        <SectionHeader
          title="VRRP"
          expanded={vrrpOpen}
          onToggle={() => setVrrpOpen(!vrrpOpen)}
          action={
            <Button type="button" variant="outline" size="sm" className="ml-auto h-7 gap-1 text-xs" onClick={addVrrp}>
              <Plus className="h-3 w-3" />Add Instance
            </Button>
          }
        />
        {vrrpOpen && (
          <div className="mt-2 space-y-3">
            {(iface.vrrp ?? []).length === 0 && (
              <p className="text-sm text-muted-foreground italic py-1">No VRRP instances configured.</p>
            )}
            {(iface.vrrp ?? []).map((v, idx) => (
              <div key={idx} className="border rounded p-3 space-y-3 bg-muted/30">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Instance #{idx + 1}</span>
                  <Button variant="ghost" size="sm" onClick={() => deleteVrrp(idx)} className="h-7 w-7 p-0 hover:text-destructive">
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs">Name (optional)</Label>
                  <Input value={v.name ?? ''} className="text-sm"
                    onChange={(e) => setVrrpField(idx, 'name', e.target.value)}
                    placeholder="vrrp-gateway" />
                </div>
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                  <div className="space-y-1">
                    <Label className="text-xs">VRRP ID</Label>
                    <Input value={v.id || ''} onChange={(e) => setVrrpField(idx, 'id', Number(e.target.value) || 0)} placeholder="1" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Priority</Label>
                    <Input value={v.priority || ''} onChange={(e) => setVrrpField(idx, 'priority', Number(e.target.value) || 0)} placeholder="100" />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Interface</Label>
                    <Input value={v.interface} className="font-mono text-sm"
                      onChange={(e) => setVrrpField(idx, 'interface', e.target.value)} placeholder="eth0" />
                  </div>
                  <div className="space-y-1 col-span-full">
                    <Label className="text-xs">Virtual IPs</Label>
                    <TagInput values={v.vips} onChange={(val) => setVrrpField(idx, 'vips', val)} placeholder="10.0.0.1/24" mono validate={checkIPOrCIDR} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Password</Label>
                    <Input type="password" value={v.password}
                      onChange={(e) => setVrrpField(idx, 'password', e.target.value)} placeholder="secret" />
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Switch checked={!!v.switchover} onCheckedChange={(val) => setVrrpField(idx, 'switchover', val)} />
                  <span className="text-sm font-medium">Switchover on BACKUP</span>
                  <span className="text-xs text-muted-foreground">(brings down WireGuard when entering BACKUP state)</span>
                </div>
                <div className="flex items-center gap-2">
                  <Switch checked={!!v.allow_inbound} onCheckedChange={(val) => setVrrpField(idx, 'allow_inbound', val)} />
                  <span className="text-sm font-medium">Allow inbound</span>
                  <span className="text-xs text-muted-foreground">(opens VRRP protocol on this interface in the firewall)</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add Interface</Button>}
      </div>
    </div>
  )
}

interface IfaceRowProps {
  name: string
  iface: Iface
  onEdit: () => void
  onDelete: () => void
}

function IfaceRow({ name, iface, onEdit, onDelete }: IfaceRowProps) {
  const vlanCount = Object.keys(iface.vlans ?? {}).length
  const hasVrrp = (iface.vrrp ?? []).length > 0

  return (
    <div onClick={onEdit} className="flex items-start gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors">
      <span className="font-mono font-semibold text-sm w-28 shrink-0 truncate pt-0.5">{name}</span>
      <div className="flex flex-wrap gap-1 flex-1 min-w-0">
        {(iface.addresses ?? []).length === 0 && (
          <span className="text-xs text-muted-foreground">-</span>
        )}
        {(iface.addresses ?? []).map((addr) => (
          <Badge key={addr} variant="outline" className="text-xs font-mono">{addr}</Badge>
        ))}
        {iface.type && (
          <Badge variant="secondary" className="text-xs font-mono">{iface.type}</Badge>
        )}
        {iface.vrf && (
          <Badge variant="secondary" className="text-xs font-mono">vrf:{iface.vrf}</Badge>
        )}
        {vlanCount > 0 && (
          <Badge variant="outline" className="text-xs">{vlanCount} VLAN{vlanCount > 1 ? 's' : ''}</Badge>
        )}
        {iface.bridge && (
          <Badge variant="outline" className="text-xs">bridge</Badge>
        )}
        {hasVrrp && (
          <Badge variant="outline" className="text-xs">VRRP</Badge>
        )}
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={(e) => { e.stopPropagation(); onDelete() }}
        className="h-7 w-7 p-0 hover:text-destructive shrink-0"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

const NEW_KEY = '__new__'

export default function Interfaces() {
  const { data, isLoading, reload } = useFetch<IfaceMap>(() => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<IfaceMap>)
  const { data: vrfsData } = useFetch<Record<string, { table: number }>>(
    () => api.apiConfigSectionGet({ section: 'vrfs' }) as Promise<Record<string, { table: number }>>
  )
  const vrfNames = Object.keys(vrfsData ?? {})
  const [entries, setEntries] = useState<IfaceMap | null>(null)
  const [newName, setNewName] = useState('')
  const [openSheet, setOpenSheet] = useState<string | null>(null)
  const [formDraft, setFormDraft] = useState<Iface | null>(null)
  const [pendingName, setPendingName] = useState<string | null>(null)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('interfaces')

  useDataRefresh(() => { setEntries(null); reset() })

  const current: IfaceMap = entries ?? (data as IfaceMap | null) ?? {}

  const update = (name: string, iface: Iface) => {
    setEntries({ ...current, [name]: iface })
    markDirty()
  }

  const deleteEntry = (name: string) => {
    const next = { ...current }
    delete next[name]
    setEntries(next)
    if (openSheet === name) setOpenSheet(null)
    markDirty()
  }

  const handleAddNew = () => { setFormDraft(emptyIface()); setNewName(''); setOpenSheet(NEW_KEY) }

  const handleCommitNew = () => {
    const n = newName.trim()
    if (!n) { toast.error('Please enter an interface name'); return }
    if (current[n]) { toast.error(`Interface "${n}" already exists`); return }
    setEntries({ ...current, [n]: formDraft ?? emptyIface() })
    markDirty()
    setOpenSheet(null)
    setFormDraft(null)
  }

  const handleCloseSheet = () => {
    if (openSheet && openSheet !== NEW_KEY && pendingName !== null && pendingName.trim() !== openSheet) {
      const n = pendingName.trim()
      if (n && !current[n]) {
        const next: IfaceMap = {}
        for (const [k, v] of Object.entries(current)) {
          next[k === openSheet ? n : k] = v
        }
        setEntries(next)
        markDirty()
      }
    }
    setOpenSheet(null)
    setFormDraft(null)
    setPendingName(null)
  }

  const names = Object.keys(current)
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(names, 12)

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      <PageHeader title="Interfaces" description="Manage network interface configuration" action={
        <div className="flex items-center gap-2">
          <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(current)} onCancel={() => { setEntries(null); reset() }} />
          <ReloadButton onClick={() => { setEntries(null); reload() }} />
        </div>
      } />

      {names.length === 0 ? (
        <EmptyState
          className="max-w-2xl mx-auto"
          icon={<Network />}
          title="No interfaces"
          message="Define an interface to assign addresses, VLANs and VRRP."
          action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add interface</Button>}
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
          {pageItems.map((name) => (
            <IfaceRow
              key={name}
              name={name}
              iface={current[name]}
              onEdit={() => { setOpenSheet(name); setPendingName(name) }}
              onDelete={() => deleteEntry(name)}
            />
          ))}
        </PreferencesColumns>
      )}

      <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="interfaces" />

      <Sheet
        open={!!openSheet}
        onClose={handleCloseSheet}
        title={openSheet === NEW_KEY ? 'New Interface' : (pendingName ?? openSheet ?? '')}
        className="max-w-2xl"
      >
        {openSheet && (openSheet === NEW_KEY ? formDraft : current[openSheet]) && (
          <IfaceForm
            name={openSheet === NEW_KEY ? newName : (pendingName ?? openSheet ?? '')}
            iface={openSheet === NEW_KEY ? formDraft! : current[openSheet]}
            vrfNames={vrfNames}
            onChange={openSheet === NEW_KEY ? setFormDraft : (u) => update(openSheet, u)}
            onNameChange={openSheet === NEW_KEY ? setNewName : setPendingName}
            onAdd={openSheet === NEW_KEY ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
          />
        )}
      </Sheet>
    </div>
  )
}
