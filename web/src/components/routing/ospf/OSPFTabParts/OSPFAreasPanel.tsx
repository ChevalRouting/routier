import { ExtraDirectives } from '@/components/ExtraDirectives'
import { AreaRow } from '@/components/routing/ospf/OSPFTab'
import { newId } from '@/components/routing/shared'
import { OSPF_TYPES, OSPFArea } from '@/components/routing/types'
import { checkCIDR } from '@/lib/validate'
import { Badge, Button, ComboRow, EmptyState, EntryRow, PreferencesGroup, Sheet, SwitchRow, TagInput } from 'cheval-ui'
import { Plus } from 'lucide-react'
import { EntityRow } from '@/components/ui/EntityRow'
import { NumberRow } from '@/components/ui/NumberRow'
import { SelectRow } from '@/components/ui/SelectRow'
import { plural } from '@/lib/utils'
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
              <Badge variant="outline">{plural(row.networks?.length ?? 0, 'network')}</Badge>
            </EntityRow>
          ))}
        </PreferencesGroup>
      )}

      <Sheet open={openRow !== null} onClose={() => setOpenRowId(null)} title={openRow?.id || 'OSPF Area'}>
        {openRow && (
          <div className="space-y-6">
            <PreferencesGroup>
              <EntryRow title="Area ID" value={openRow.id} onChange={(e) => upd(openRow._rowId, { id: e.target.value })} placeholder="0.0.0.0" className="font-mono" />
              <SelectRow title="Type" value={openRow.type || 'normal'} onChange={(v) => upd(openRow._rowId, { type: v ?? 'normal' })} options={OSPF_TYPES} />
              {openRow.type === 'stub' && (
                <SwitchRow title="Totally Stubby (no-summary)" checked={!!openRow.stub_no_summary} onCheckedChange={(v) => upd(openRow._rowId, { stub_no_summary: v || undefined })} />
              )}
              <SelectRow title="Authentication" value={openRow.auth} onChange={(v) => upd(openRow._rowId, { auth: v ?? '' })} options={['simple', 'md5']} noneLabel="none" />
              <ComboRow title="Networks">
                <TagInput values={openRow.networks ?? []} onChange={(v) => upd(openRow._rowId, { networks: v })} placeholder="10.0.0.0/24" mono validate={checkCIDR} />
              </ComboRow>
              <NumberRow title="Default Cost" value={openRow.default_cost} onChange={(v) => upd(openRow._rowId, { default_cost: v })} placeholder="none" />
              <ComboRow title="Ranges (summarize)">
                <TagInput values={openRow.ranges ?? []} onChange={(v) => upd(openRow._rowId, { ranges: v.length ? v : undefined })} placeholder="10.0.0.0/16" mono validate={checkCIDR} />
              </ComboRow>
            </PreferencesGroup>
            <ExtraDirectives label="Additional Area Directives" values={openRow.extra} onChange={(v) => upd(openRow._rowId, { extra: v })} placeholder="area 0.0.0.0 shortcut enable" />
          </div>
        )}
      </Sheet>
    </div>
  )
}
