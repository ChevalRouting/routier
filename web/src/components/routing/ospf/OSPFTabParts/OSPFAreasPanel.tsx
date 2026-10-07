import { ExtraDirectives } from '@/components/ExtraDirectives'
import { AreaRow } from '@/components/routing/ospf/OSPFTab'
import { newId } from '@/components/routing/shared'
import { OSPF_TYPES, OSPFArea } from '@/components/routing/types'
import { checkCIDR } from '@/lib/validate'
import { Badge, Button, Card, Input, Label, NumberInput, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Sheet, Switch, TagInput } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

type OSPFAreasPanelShape = {
  areas: OSPFArea[]
  onChange: (v: OSPFArea[]) => void
  onDirty: () => void
}

export function OSPFAreasPanel({
  areas, onChange, onDirty,
}: OSPFAreasPanelShape) {
  const changeCallbacks = useRef({ onChange, onDirty })
  useEffect(() => { changeCallbacks.current = { onChange, onDirty } }, [onChange, onDirty])
  const [rows, setRows] = useState<AreaRow[]>(() =>
    areas.map((a) => ({ ...a, _rowId: newId() }))
  )
  const [openRowId, setOpenRowId] = useState<number | null>(null)
  const firstRun = useRef(true)

  useEffect(() => {
    if (firstRun.current) { firstRun.current = false; return }
    changeCallbacks.current.onChange(rows.map(({ _rowId: _, ...rest }) => rest as OSPFArea))
    changeCallbacks.current.onDirty()
  }, [rows])

  const add = () => { const rowId = newId(); setRows((p) => [...p, { _rowId: rowId, id: '', networks: [], type: 'normal', auth: '' }]); setOpenRowId(rowId) }
  const remove = (rowId: number) => { setRows((p) => p.filter((r) => r._rowId !== rowId)); if (openRowId === rowId) setOpenRowId(null) }
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
        <div className="grid gap-3 md:grid-cols-2">
          {rows.map((row) => (
            <div key={row._rowId}>
              <Card className="flex cursor-pointer items-center gap-2 p-4 transition-colors hover:bg-accent/50" onClick={() => setOpenRowId(row._rowId)}>
                <span className="flex-1 truncate font-mono text-sm font-semibold">{row.id || 'New area'}</span>
                <Badge variant="secondary">{row.type || 'normal'}</Badge>
                <Badge variant="outline">{row.networks?.length ?? 0} networks</Badge>
                <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={(event) => { event.stopPropagation(); remove(row._rowId) }}>
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </Card>
              <Sheet open={openRowId === row._rowId} onClose={() => setOpenRowId(null)} title={row.id || 'OSPF area'} className="max-w-2xl">
                <div className="space-y-4">
                  <div className="grid gap-4 sm:grid-cols-3">
                    <div className="space-y-1.5"><Label className="text-xs text-muted-foreground">Area ID</Label>
                <Input value={row.id} onChange={(e) => upd(row._rowId, { id: e.target.value })} placeholder="0.0.0.0" className="font-mono text-xs h-8 border-0 shadow-none focus-visible:ring-1" />
                    </div><div className="space-y-1.5"><Label className="text-xs text-muted-foreground">Type</Label>
                <Select value={row.type || 'normal'} onValueChange={(v) => upd(row._rowId, { type: v })}>
                  <SelectTrigger className="h-8 text-xs font-mono">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {OSPF_TYPES.map((t) => <SelectItem key={t} value={t}>{t}</SelectItem>)}
                  </SelectContent>
                </Select>
                    </div><div className="space-y-1.5"><Label className="text-xs text-muted-foreground">Authentication</Label>
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
                    </div>
                  </div>
                  <div className="space-y-1.5"><Label className="text-xs text-muted-foreground">Networks</Label>
                <TagInput values={row.networks ?? []} onChange={(v) => upd(row._rowId, { networks: v })} placeholder="10.0.0.0/24" mono validate={checkCIDR} />
                  </div>
              <div className="grid grid-cols-[130px_1fr] items-center gap-3">
                <div className="space-y-1.5">
                  <Label className="text-[10px] text-muted-foreground">Default cost</Label>
                  <NumberInput value={row.default_cost || undefined} onChange={(v) => upd(row._rowId, { default_cost: v })} placeholder="none" className="font-mono h-8 text-xs" />
                </div>
                <div className="space-y-1.5">
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
              </Sheet>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
