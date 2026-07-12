import { useState, useEffect, useRef } from 'react'
import { useTabState } from '@/lib/useTabState'
import { SectionNav } from '@/components/ui/section-nav'
import { checkCIDR } from '@/lib/validate'
import { Input } from '@/components/ui/input'
import { NumberInput } from '@/components/ui/number-input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import TagInput from '@/components/TagInput'
import { Plus, Trash2, ChevronDown, ChevronRight } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ExtraDirectives } from '@/components/ExtraDirectives'
import { OSPFArea, OSPFInterfaceConfig, OSPFConfig, OSPF_TYPES, OSPF_NETWORK_TYPES, OSPFSubTab } from '../types'
import { newId } from '../shared'

export interface AreaRow extends OSPFArea { _rowId: number }
export interface IfaceRow extends OSPFInterfaceConfig { _rowId: number; name: string }

export const OSPF_AUTH_TYPES = ['none', 'simple', 'md5']

export function OSPFInterfacesPanel({
  interfaces, onChange, ifaceNames, areaIds, onDirty,
}: {
  interfaces: Record<string, OSPFInterfaceConfig>
  onChange: (v: Record<string, OSPFInterfaceConfig>) => void
  ifaceNames: string[]
  areaIds: string[]
  onDirty: () => void
}) {
  const [rows, setRows] = useState<IfaceRow[]>(() =>
    Object.entries(interfaces).map(([name, cfg]) => ({ ...cfg, name, _rowId: newId() }))
  )
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const firstRun = useRef(true)

  useEffect(() => {
    if (firstRun.current) { firstRun.current = false; return }
    const result: Record<string, OSPFInterfaceConfig> = {}
    for (const { _rowId: _, name, ...cfg } of rows) {
      if (name) result[name] = cfg
    }
    onChange(result)
    onDirty()
  }, [rows])

  const add = () => {
    const rowId = newId()
    setRows((p) => [...p, { _rowId: rowId, name: '', auth_type: 'none' }])
    setExpanded((p) => new Set([...p, rowId]))
  }
  const remove = (rowId: number) => setRows((p) => p.filter((r) => r._rowId !== rowId))
  const toggle = (rowId: number) =>
    setExpanded((p) => { const n = new Set(p); if (n.has(rowId)) { n.delete(rowId) } else { n.add(rowId) } return n })
  const upd = (rowId: number, patch: Partial<IfaceRow>) =>
    setRows((p) => p.map((r) => (r._rowId === rowId ? { ...r, ...patch } : r)))

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label className="text-sm font-semibold">Interface settings</Label>
        <Button variant="outline" size="sm" onClick={add} className="gap-1.5">
          <Plus className="h-3.5 w-3.5" />Add Interface
        </Button>
      </div>

      {rows.length === 0 && (
        <p className="text-sm text-muted-foreground italic">No per-interface OSPF settings.</p>
      )}

      <div className="space-y-2">
        {rows.map((row) => {
          const open = expanded.has(row._rowId)
          return (
            <div key={row._rowId} className="border rounded-md overflow-hidden">
              <div className="flex items-center gap-2 px-3 py-2 bg-muted/30">
                <button type="button" className="text-muted-foreground hover:text-foreground" onClick={() => toggle(row._rowId)}>
                  {open ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                </button>
                <Select value={row.name || '__none__'} onValueChange={(v) => upd(row._rowId, { name: v === '__none__' ? '' : v })}>
                  <SelectTrigger className="h-8 text-xs font-mono w-36 border-0 shadow-none focus-visible:ring-1">
                    <SelectValue placeholder="Interface…" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__none__">- select -</SelectItem>
                    {ifaceNames.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}
                  </SelectContent>
                </Select>
                {row.area && <Badge variant="secondary" className="text-xs font-mono">area {row.area}</Badge>}
                {row.cost != null && row.cost > 0 && <Badge variant="secondary" className="text-xs">cost {row.cost}</Badge>}
                {row.auth_type && row.auth_type !== 'none' && <Badge variant="outline" className="text-xs">{row.auth_type}</Badge>}
                <div className="flex-1" />
                <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => remove(row._rowId)}>
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>

              {open && (
                <div className="px-4 pb-4 pt-3 space-y-4 border-t">
                  <div className="space-y-1.5 max-w-xs">
                    <Label className="text-xs text-muted-foreground">Area</Label>
                    <Select value={row.area || '__none__'} onValueChange={(v) => upd(row._rowId, { area: v === '__none__' ? '' : v })}>
                      <SelectTrigger className="h-8 text-xs font-mono">
                        <SelectValue placeholder="- none -" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="__none__">- none -</SelectItem>
                        {areaIds.map((a) => <SelectItem key={a} value={a}>{a}</SelectItem>)}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="grid grid-cols-3 gap-4">
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">Cost</Label>
                      <NumberInput value={row.cost || undefined} onChange={(v) => upd(row._rowId, { cost: v })} placeholder="auto" className="font-mono" />
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">Hello interval (s)</Label>
                      <NumberInput value={row.hello_interval || undefined} onChange={(v) => upd(row._rowId, { hello_interval: v })} placeholder="10" className="font-mono" />
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">Dead interval (s)</Label>
                      <NumberInput value={row.dead_interval || undefined} onChange={(v) => upd(row._rowId, { dead_interval: v })} placeholder="40" className="font-mono" />
                    </div>
                  </div>
                  <div className="space-y-1.5 max-w-xs">
                    <Label className="text-xs text-muted-foreground">Authentication</Label>
                    <Select value={row.auth_type ?? 'none'} onValueChange={(v) => upd(row._rowId, { auth_type: v })}>
                      <SelectTrigger className="h-8 text-xs font-mono">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {OSPF_AUTH_TYPES.map((t) => <SelectItem key={t} value={t}>{t}</SelectItem>)}
                      </SelectContent>
                    </Select>
                  </div>
                  {row.auth_type && row.auth_type !== 'none' && (
                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-1.5">
                        <Label className="text-xs text-muted-foreground">Auth key</Label>
                        <Input type="password" value={row.auth_key ?? ''} onChange={(e) => upd(row._rowId, { auth_key: e.target.value })} placeholder="password" className="font-mono" />
                      </div>
                      {row.auth_type === 'md5' && (
                        <div className="space-y-1.5">
                          <Label className="text-xs text-muted-foreground">Key ID</Label>
                          <NumberInput value={row.auth_key_id || undefined} onChange={(v) => upd(row._rowId, { auth_key_id: v })} placeholder="1" className="font-mono" />
                        </div>
                      )}
                    </div>
                  )}
                  <div className="grid grid-cols-3 gap-4">
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">Network type</Label>
                      <Select value={row.network_type || '__default__'} onValueChange={(v) => upd(row._rowId, { network_type: v === '__default__' ? undefined : v })}>
                        <SelectTrigger className="h-8 text-xs font-mono"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="__default__">default</SelectItem>
                          {OSPF_NETWORK_TYPES.map((t) => <SelectItem key={t} value={t}>{t}</SelectItem>)}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">Priority</Label>
                      <NumberInput value={row.priority || undefined} onChange={(v) => upd(row._rowId, { priority: v })} placeholder="1" className="font-mono" />
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">Retransmit interval (s)</Label>
                      <NumberInput value={row.retransmit_interval || undefined} onChange={(v) => upd(row._rowId, { retransmit_interval: v })} placeholder="5" className="font-mono" />
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">Transmit delay (s)</Label>
                      <NumberInput value={row.transmit_delay || undefined} onChange={(v) => upd(row._rowId, { transmit_delay: v })} placeholder="1" className="font-mono" />
                    </div>
                  </div>
                  <div className="flex items-center gap-6">
                    <div className="flex items-center gap-2"><Switch checked={!!row.mtu_ignore} onCheckedChange={(v) => upd(row._rowId, { mtu_ignore: v || undefined })} /><span className="text-sm">MTU ignore</span></div>
                    <div className="flex items-center gap-2"><Switch checked={!!row.bfd} onCheckedChange={(v) => upd(row._rowId, { bfd: v || undefined })} /><span className="text-sm">BFD</span></div>
                  </div>
                  <ExtraDirectives label="Additional ip ospf directives" values={row.extra} onChange={(v) => upd(row._rowId, { extra: v })} placeholder="ip ospf authentication null" />
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

export function OSPFGeneralPanel({
  ospf, setOSPF, onDirty,
}: {
  ospf: OSPFConfig
  setOSPF: (v: OSPFConfig) => void
  onDirty: () => void
}) {
  const upd = (patch: Partial<OSPFConfig>) => { setOSPF({ ...ospf, ...patch }); onDirty() }
  return (
    <div className="space-y-6 max-w-lg">
      <div className="space-y-1.5">
        <Label htmlFor="ospf-rid" className="text-xs">Router ID</Label>
        <Input id="ospf-rid" value={ospf.router_id} onChange={(e) => upd({ router_id: e.target.value })} placeholder="10.0.0.1" className="font-mono" />
      </div>
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-medium">Default information originate</div>
          <div className="text-xs text-muted-foreground">Advertise a default route (0.0.0.0/0) into the OSPF domain</div>
        </div>
        <Switch checked={!!ospf.default_information_originate} onCheckedChange={(v) => upd({ default_information_originate: v })} />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-1.5">
          <Label className="text-xs">Reference bandwidth (Mbps)</Label>
          <NumberInput value={ospf.reference_bandwidth || undefined} onChange={(v) => upd({ reference_bandwidth: v })} placeholder="auto-cost" className="font-mono" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs">Administrative distance</Label>
          <NumberInput value={ospf.distance || undefined} onChange={(v) => upd({ distance: v })} placeholder="110" className="font-mono" />
        </div>
      </div>
      <Separator />
      <div className="space-y-1.5">
        <Label className="text-xs font-semibold">Passive interfaces</Label>
        <TagInput values={ospf.passive_interfaces} onChange={(v) => upd({ passive_interfaces: v })} placeholder="eth0" mono />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-semibold">Redistribute</Label>
        <TagInput values={ospf.redistribute} onChange={(v) => upd({ redistribute: v })} placeholder="connected" />
      </div>
      <Separator />
      <ExtraDirectives label="Additional router ospf directives" values={ospf.extra} onChange={(v) => upd({ extra: v })} placeholder="auto-cost reference-bandwidth 100000" />
    </div>
  )
}

export function OSPFAreasPanel({
  areas, onChange, onDirty,
}: {
  areas: OSPFArea[]
  onChange: (v: OSPFArea[]) => void
  onDirty: () => void
}) {
  const [rows, setRows] = useState<AreaRow[]>(() =>
    areas.map((a) => ({ ...a, _rowId: newId() }))
  )
  const firstRun = useRef(true)

  useEffect(() => {
    if (firstRun.current) { firstRun.current = false; return }
    onChange(rows.map(({ _rowId: _, ...rest }) => rest as OSPFArea))
    onDirty()
  }, [rows])

  const add = () => setRows((p) => [...p, { _rowId: newId(), id: '', networks: [], type: 'normal', auth: '' }])
  const remove = (rowId: number) => setRows((p) => p.filter((r) => r._rowId !== rowId))
  const upd = (rowId: number, patch: Partial<AreaRow>) =>
    setRows((p) => p.map((r) => (r._rowId === rowId ? { ...r, ...patch } : r)))

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label className="text-sm font-semibold">Areas</Label>
        <Button variant="outline" size="sm" onClick={add} className="gap-1.5">
          <Plus className="h-3.5 w-3.5" />Add Area
        </Button>
      </div>
      {rows.length === 0 && (
        <p className="text-sm text-muted-foreground italic">No areas configured.</p>
      )}
      {rows.length > 0 && (
        <div className="rounded-xl border border-border overflow-hidden divide-y">
          <div className="grid grid-cols-[110px_110px_110px_1fr_36px] bg-muted/50 px-3 py-2 text-xs font-medium text-muted-foreground gap-3">
            <span>Area ID</span><span>Type</span><span>Auth</span><span>Networks</span><span />
          </div>
          {rows.map((row) => (
            <div key={row._rowId} className="px-3 py-2 space-y-2">
              <div className="grid grid-cols-[110px_110px_110px_1fr_36px] items-start gap-3">
                <Input value={row.id} onChange={(e) => upd(row._rowId, { id: e.target.value })} placeholder="0.0.0.0" className="font-mono text-xs h-8 border-0 shadow-none focus-visible:ring-1" />
                <Select value={row.type || 'normal'} onValueChange={(v) => upd(row._rowId, { type: v })}>
                  <SelectTrigger className="h-8 text-xs font-mono">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {OSPF_TYPES.map((t) => <SelectItem key={t} value={t}>{t}</SelectItem>)}
                  </SelectContent>
                </Select>
                <Select value={row.auth || 'none'} onValueChange={(v) => upd(row._rowId, { auth: v === 'none' ? '' : v })}>
                  <SelectTrigger className="h-8 text-xs font-mono">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">none</SelectItem>
                    <SelectItem value="simple">simple</SelectItem>
                    <SelectItem value="md5">md5</SelectItem>
                  </SelectContent>
                </Select>
                <TagInput values={row.networks} onChange={(v) => upd(row._rowId, { networks: v })} placeholder="10.0.0.0/24" mono validate={checkCIDR} />
                <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive mt-0.5" onClick={() => remove(row._rowId)}>
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
              <div className="grid grid-cols-[130px_1fr] items-center gap-3 pl-1">
                <div className="space-y-1">
                  <Label className="text-[10px] text-muted-foreground">Default cost</Label>
                  <NumberInput value={row.default_cost || undefined} onChange={(v) => upd(row._rowId, { default_cost: v })} placeholder="none" className="font-mono h-8 text-xs" />
                </div>
                <div className="space-y-1">
                  <Label className="text-[10px] text-muted-foreground">Ranges (summarize)</Label>
                  <TagInput values={row.ranges ?? []} onChange={(v) => upd(row._rowId, { ranges: v.length ? v : undefined })} placeholder="10.0.0.0/16" mono validate={checkCIDR} />
                </div>
              </div>
              {row.type === 'stub' && (
                <label className="flex items-center gap-2 pl-1 text-xs">
                  <Switch checked={!!row.stub_no_summary} onCheckedChange={(v) => upd(row._rowId, { stub_no_summary: v || undefined })} />
                  <span>Totally stubby (no-summary)</span>
                </label>
              )}
              <ExtraDirectives label="Additional area directives" values={row.extra} onChange={(v) => upd(row._rowId, { extra: v })} placeholder="area 0.0.0.0 shortcut enable" />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export function OSPFTab({
  ospf, setOSPF, enabled, setEnabled, onDirty, ifaceNames,
}: {
  ospf: OSPFConfig
  setOSPF: (v: OSPFConfig) => void
  enabled: boolean
  setEnabled: (v: boolean) => void
  onDirty: () => void
  ifaceNames: string[]
}) {
  const [subTab, setSubTab] = useTabState<OSPFSubTab>('routing.ospf', 'general')

  const subTabs: { key: OSPFSubTab; label: string }[] = [
    { key: 'general', label: 'General' },
    { key: 'areas', label: ospf.areas?.length ? `Areas (${ospf.areas.length})` : 'Areas' },
    { key: 'interfaces', label: Object.keys(ospf.interfaces ?? {}).length ? `Interfaces (${Object.keys(ospf.interfaces ?? {}).length})` : 'Interfaces' },
  ]

  return (
    <div className="space-y-5">
      <p className="text-sm text-muted-foreground">
        OSPF (Open Shortest Path First) is a link-state IGP that distributes IPv4 routes within your
        network by flooding link-state information across areas.
      </p>
      <div className="flex items-center justify-between gap-2">
        <Label htmlFor="ospf-enable" className="cursor-pointer">Enable OSPF</Label>
        <Switch id="ospf-enable" checked={enabled} onCheckedChange={(v) => { setEnabled(v); onDirty() }} />
      </div>

      {!enabled ? (
        <p className="text-sm text-muted-foreground italic">OSPF is disabled. Enable it above to configure.</p>
      ) : (
        <SectionNav items={subTabs} active={subTab} onChange={setSubTab}>
          {subTab === 'general' && (
            <OSPFGeneralPanel ospf={ospf} setOSPF={setOSPF} onDirty={onDirty} />
          )}
          {subTab === 'areas' && (
            <OSPFAreasPanel
              areas={ospf.areas ?? []}
              onChange={(areas) => setOSPF({ ...ospf, areas })}
              onDirty={onDirty}
            />
          )}
          {subTab === 'interfaces' && (
            <OSPFInterfacesPanel
              interfaces={ospf.interfaces ?? {}}
              onChange={(interfaces) => setOSPF({ ...ospf, interfaces })}
              ifaceNames={ifaceNames}
              areaIds={(ospf.areas ?? []).map((a) => a.id).filter(Boolean)}
              onDirty={onDirty}
            />
          )}
        </SectionNav>
      )}
    </div>
  )
}

