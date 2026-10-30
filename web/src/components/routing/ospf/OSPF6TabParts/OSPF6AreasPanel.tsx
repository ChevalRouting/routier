import { ExtraDirectives } from '@/components/ExtraDirectives'
import { OSPF6AreaRow } from '@/components/routing/ospf/OSPF6Tab'
import { newId } from '@/components/routing/shared'
import { OSPF6Area } from '@/components/routing/types'
import { checkCIDR } from '@/lib/validate'
import { Badge, Button, ComboRow, EmptyState, EntryRow, PreferencesGroup, Sheet, SwitchRow, TagInput } from 'cheval-ui'
import { Plus } from 'lucide-react'
import { EntityRow } from '@/components/ui/EntityRow'
import { NumberRow } from '@/components/ui/NumberRow'
import { SelectRow } from '@/components/ui/SelectRow'
import { plural } from '@/lib/utils'
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
  const openRow = rows.find((row) => row._rowId === openRowId) ?? null
  const addButton = <Button variant="outline" size="sm" onClick={add} className="gap-1.5"><Plus className="h-4 w-4" />Add Area</Button>

  return (
    <div className="space-y-6">
      {rows.length === 0 ? (
        <EmptyState className="py-10" title="No Areas" action={addButton} />
      ) : (
        <PreferencesGroup
          title="Areas"
          header={addButton}
        >
          {rows.map((row) => (
            <EntityRow key={row._rowId} title={row.id} placeholder="New area" onOpen={() => setOpenRowId(row._rowId)} onRemove={() => remove(row._rowId)} removeLabel="Remove Area">
              <Badge variant="secondary">{row.type || 'normal'}</Badge>
              <Badge variant="outline">{plural(row.ranges?.length ?? 0, 'range')}</Badge>
            </EntityRow>
          ))}
        </PreferencesGroup>
      )}

      <Sheet open={openRow !== null} onClose={() => setOpenRowId(null)} title={openRow?.id || 'OSPFv3 Area'}>
        {openRow && (
          <div className="space-y-6">
            <PreferencesGroup>
              <EntryRow title="Area ID" value={openRow.id} onChange={(e) => upd(openRow._rowId, { id: e.target.value })} placeholder="0.0.0.0" className="font-mono" />
              <SelectRow title="Type" value={openRow.type || 'normal'} onChange={(v) => upd(openRow._rowId, { type: !v || v === 'normal' ? '' : v })} options={['normal', 'stub', 'nssa']} />
              {openRow.type === 'stub' && (
                <SwitchRow title="Totally Stubby (no-summary)" checked={!!openRow.stub_no_summary} onCheckedChange={(v) => upd(openRow._rowId, { stub_no_summary: v || undefined })} />
              )}
              <ComboRow title="Ranges">
                <TagInput values={openRow.ranges ?? []} onChange={(v) => upd(openRow._rowId, { ranges: v })} placeholder="2001:db8::/32" mono validate={checkCIDR} />
              </ComboRow>
              <NumberRow title="Default Cost" value={openRow.default_cost} onChange={(v) => upd(openRow._rowId, { default_cost: v })} placeholder="none" />
            </PreferencesGroup>
            <ExtraDirectives label="Additional Area Directives" values={openRow.extra} onChange={(v) => upd(openRow._rowId, { extra: v })} placeholder="area 0.0.0.0 range 2001:db8::/32 advertise" />
          </div>
        )}
      </Sheet>
    </div>
  )
}
