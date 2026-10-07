import { ExtraDirectives } from '@/components/ExtraDirectives'
import { OSPF6AreaRow } from '@/components/routing/ospf/OSPF6Tab'
import { newId } from '@/components/routing/shared'
import { OSPF6Area } from '@/components/routing/types'
import { checkCIDR } from '@/lib/validate'
import { Badge, Button, Card, Input, Label, NumberInput, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Sheet, Switch, TagInput } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

type OSPF6AreasPanelShape = {
  areas: OSPF6Area[]; onChange: (v: OSPF6Area[]) => void; onDirty: () => void
}

export function OSPF6AreasPanel({ areas, onChange, onDirty }: OSPF6AreasPanelShape) {
  const changeCallbacks = useRef({ onChange, onDirty })
  useEffect(() => { changeCallbacks.current = { onChange, onDirty } }, [onChange, onDirty])
  const [rows, setRows] = useState<OSPF6AreaRow[]>(() => areas.map((a) => ({ ...a, _rowId: newId() })))
  const [openRowId, setOpenRowId] = useState<number | null>(null)
  const firstRun = useRef(true)
  useEffect(() => {
    if (firstRun.current) { firstRun.current = false; return }
    changeCallbacks.current.onChange(rows.map(({ _rowId: _, ...rest }) => rest as OSPF6Area))
    changeCallbacks.current.onDirty()
  }, [rows])
  const add = () => { const id = newId(); setRows((p) => [...p, { _rowId: id, id: '', ranges: [], type: '' }]); setOpenRowId(id) }
  const remove = (id: number) => { setRows((p) => p.filter((r) => r._rowId !== id)); if (openRowId === id) setOpenRowId(null) }
  const upd = (id: number, patch: Partial<OSPF6AreaRow>) =>
    setRows((p) => p.map((r) => (r._rowId === id ? { ...r, ...patch } : r)))
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label className="text-sm font-semibold">Areas</Label>
        <Button variant="outline" size="sm" onClick={add} className="gap-1.5">
          <Plus className="h-3.5 w-3.5" />Add Area
        </Button>
      </div>
      {rows.length === 0 && <p className="text-sm text-muted-foreground italic">No areas configured.</p>}
      {rows.length > 0 && (
        <div className="grid gap-3 md:grid-cols-2">
          {rows.map((row) => (
            <div key={row._rowId}>
              <Card className="flex cursor-pointer items-center gap-2 p-4 transition-colors hover:bg-accent/50" onClick={() => setOpenRowId(row._rowId)}>
                <span className="flex-1 truncate font-mono text-sm font-semibold">{row.id || 'New area'}</span>
                <Badge variant="secondary">{row.type || 'normal'}</Badge>
                <Badge variant="outline">{row.ranges?.length ?? 0} ranges</Badge>
                <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={(event) => { event.stopPropagation(); remove(row._rowId) }}><Trash2 className="h-3.5 w-3.5" /></Button>
              </Card>
              <Sheet open={openRowId === row._rowId} onClose={() => setOpenRowId(null)} title={row.id || 'OSPFv3 area'} className="max-w-2xl">
                <div className="space-y-4">
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-1.5"><Label className="text-xs text-muted-foreground">Area ID</Label>
                <Input value={row.id} onChange={(e) => upd(row._rowId, { id: e.target.value })} placeholder="0.0.0.0"
                  className="font-mono" /></div>
                    <div className="space-y-1.5"><Label className="text-xs text-muted-foreground">Type</Label>
                <Select value={row.type || 'normal'} onValueChange={(v) => upd(row._rowId, { type: v === 'normal' ? '' : v })}>
                  <SelectTrigger className="h-8 text-xs font-mono"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="normal">normal</SelectItem>
                    <SelectItem value="stub">stub</SelectItem>
                    <SelectItem value="nssa">nssa</SelectItem>
                  </SelectContent>
                </Select></div>
                  </div>
                  <div className="space-y-1.5"><Label className="text-xs text-muted-foreground">Ranges (IPv6 CIDRs)</Label>
                <TagInput values={row.ranges ?? []} onChange={(v) => upd(row._rowId, { ranges: v })} placeholder="2001:db8::/32" mono validate={checkCIDR} />
                  </div>
              <div className="grid grid-cols-[130px_1fr] items-center gap-3">
                <div className="space-y-1.5">
                  <Label className="text-[10px] text-muted-foreground">Default cost</Label>
                  <NumberInput value={row.default_cost || undefined} onChange={(v) => upd(row._rowId, { default_cost: v })} placeholder="none" className="font-mono h-8 text-xs" />
                </div>
                {row.type === 'stub' && (
                  <label className="flex items-center gap-2 self-end pb-1.5 text-xs">
                    <Switch checked={!!row.stub_no_summary} onCheckedChange={(v) => upd(row._rowId, { stub_no_summary: v || undefined })} />
                    <span>Totally stubby (no-summary)</span>
                  </label>
                )}
              </div>
              <ExtraDirectives label="Additional area directives" values={row.extra} onChange={(v) => upd(row._rowId, { extra: v })} placeholder="area 0.0.0.0 range 2001:db8::/32 advertise" />
                </div>
              </Sheet>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
