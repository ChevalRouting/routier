import { ExtraDirectives } from '@/components/ExtraDirectives'
import { IfaceRow, OSPF_AUTH_TYPES } from '@/components/routing/ospf/OSPFTab'
import { newId } from '@/components/routing/shared'
import { OSPF_NETWORK_TYPES, OSPFInterfaceConfig } from '@/components/routing/types'
import { Badge, Button, Card, Input, Label, NumberInput, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Sheet, Switch } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

type OSPFInterfacesPanelShape = {
  interfaces: Record<string, OSPFInterfaceConfig>
  onChange: (v: Record<string, OSPFInterfaceConfig>) => void
  ifaceNames: string[]
  areaIds: string[]
  onDirty: () => void
}

export function OSPFInterfacesPanel({
  interfaces, onChange, ifaceNames, areaIds, onDirty,
}: OSPFInterfacesPanelShape) {
  const changeCallbacks = useRef({ onChange, onDirty })
  useEffect(() => { changeCallbacks.current = { onChange, onDirty } }, [onChange, onDirty])
  const [rows, setRows] = useState<IfaceRow[]>(() =>
    Object.entries(interfaces).map(([name, cfg]) => ({ ...cfg, name, _rowId: newId() }))
  )
  const [openRowId, setOpenRowId] = useState<number | null>(null)
  const firstRun = useRef(true)

  useEffect(() => {
    if (firstRun.current) { firstRun.current = false; return }
    const result: Record<string, OSPFInterfaceConfig> = {}
    for (const { _rowId: _, name, ...cfg } of rows) {
      if (name) result[name] = cfg
    }
    changeCallbacks.current.onChange(result)
    changeCallbacks.current.onDirty()
  }, [rows])

  const add = () => {
    const rowId = newId()
    setRows((p) => [...p, { _rowId: rowId, name: '', auth_type: 'none' }])
    setOpenRowId(rowId)
  }
  const remove = (rowId: number) => { setRows((p) => p.filter((r) => r._rowId !== rowId)); if (openRowId === rowId) setOpenRowId(null) }
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
          return (
            <div key={row._rowId}>
              <Card className="flex cursor-pointer items-center gap-2 p-4 transition-colors hover:bg-accent/50" onClick={() => setOpenRowId(row._rowId)}>
                <span className="w-36 truncate font-mono text-sm font-semibold">{row.name || 'New interface'}</span>
                {row.area && <Badge variant="secondary" className="text-xs font-mono">area {row.area}</Badge>}
                {row.cost != null && row.cost > 0 && <Badge variant="secondary" className="text-xs">cost {row.cost}</Badge>}
                {row.auth_type && row.auth_type !== 'none' && <Badge variant="outline" className="text-xs">{row.auth_type}</Badge>}
                <div className="flex-1" />
                <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={(event) => { event.stopPropagation(); remove(row._rowId) }}>
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </Card>

              <Sheet open={openRowId === row._rowId} onClose={() => setOpenRowId(null)} title={row.name || 'OSPF interface'} className="max-w-2xl">
                <div className="space-y-4">
                  <div className="space-y-1.5 max-w-xs">
                    <Label className="text-xs text-muted-foreground">Interface</Label>
                    <Select value={row.name || '__none__'} onValueChange={(v) => upd(row._rowId, { name: v === '__none__' ? '' : v })}>
                      <SelectTrigger className="font-mono"><SelectValue placeholder="Interface…" /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="__none__">- select -</SelectItem>
                        {ifaceNames.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}
                      </SelectContent>
                    </Select>
                  </div>
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
              </Sheet>
            </div>
          )
        })}
      </div>
    </div>
  )
}
