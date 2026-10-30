import { ExtraDirectives } from '@/components/ExtraDirectives'
import { IfaceRow, OSPF_AUTH_TYPES } from '@/components/routing/ospf/OSPFTab'
import { newId } from '@/components/routing/shared'
import { OSPF_NETWORK_TYPES, OSPFInterfaceConfig } from '@/components/routing/types'
import { EntityRow } from '@/components/ui/EntityRow'
import { NumberRow } from '@/components/ui/NumberRow'
import { SelectRow } from '@/components/ui/SelectRow'
import { Badge, Button, EmptyState, EntryRow, PreferencesGroup, Sheet, SwitchRow } from 'cheval-ui'
import { Plus } from 'lucide-react'
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

  const openRow = rows.find((row) => row._rowId === openRowId) ?? null

  const addButton = <Button variant="outline" size="sm" onClick={add} className="gap-1.5"><Plus className="h-4 w-4" />Add Interface</Button>

  return (
    <div className="space-y-6">
      {rows.length === 0 ? (
        <EmptyState className="py-10" title="No Interfaces" action={addButton} />
      ) : (
        <PreferencesGroup
          title="Interface Settings"
          header={addButton}
        >
          {rows.map((row) => (
            <EntityRow key={row._rowId} title={row.name} placeholder="New interface" onOpen={() => setOpenRowId(row._rowId)} onRemove={() => remove(row._rowId)} removeLabel="Remove Interface">
              {row.area && <Badge variant="secondary" className="font-mono">area {row.area}</Badge>}
              {row.cost != null && row.cost > 0 && <Badge variant="secondary">cost {row.cost}</Badge>}
              {row.auth_type && row.auth_type !== 'none' && <Badge variant="outline">{row.auth_type}</Badge>}
            </EntityRow>
          ))}
        </PreferencesGroup>
      )}

      <Sheet open={openRow !== null} onClose={() => setOpenRowId(null)} title={openRow?.name || 'OSPF Interface'}>
        {openRow && (
          <div className="space-y-6">
            <PreferencesGroup>
              <SelectRow title="Interface" value={openRow.name} onChange={(v) => upd(openRow._rowId, { name: v ?? '' })} options={ifaceNames} placeholder="Select interface" />
              <SelectRow title="Area" value={openRow.area} onChange={(v) => upd(openRow._rowId, { area: v ?? '' })} options={areaIds} noneLabel="None" />
              <NumberRow title="Cost" value={openRow.cost} onChange={(v) => upd(openRow._rowId, { cost: v })} placeholder="auto" />
              <SelectRow title="Network Type" value={openRow.network_type} onChange={(v) => upd(openRow._rowId, { network_type: v })} options={OSPF_NETWORK_TYPES} noneLabel="default" />
              <NumberRow title="Priority" value={openRow.priority} onChange={(v) => upd(openRow._rowId, { priority: v })} placeholder="1" />
            </PreferencesGroup>
            <PreferencesGroup title="Timers">
              <NumberRow title="Hello Interval (s)" value={openRow.hello_interval} onChange={(v) => upd(openRow._rowId, { hello_interval: v })} placeholder="10" />
              <NumberRow title="Dead Interval (s)" value={openRow.dead_interval} onChange={(v) => upd(openRow._rowId, { dead_interval: v })} placeholder="40" />
              <NumberRow title="Retransmit Interval (s)" value={openRow.retransmit_interval} onChange={(v) => upd(openRow._rowId, { retransmit_interval: v })} placeholder="5" />
              <NumberRow title="Transmit Delay (s)" value={openRow.transmit_delay} onChange={(v) => upd(openRow._rowId, { transmit_delay: v })} placeholder="1" />
            </PreferencesGroup>
            <PreferencesGroup title="Authentication">
              <SelectRow title="Type" value={openRow.auth_type ?? 'none'} onChange={(v) => upd(openRow._rowId, { auth_type: v ?? 'none' })} options={OSPF_AUTH_TYPES} />
              {openRow.auth_type && openRow.auth_type !== 'none' && (
                <EntryRow title="Key" type="password" value={openRow.auth_key ?? ''} onChange={(e) => upd(openRow._rowId, { auth_key: e.target.value })} className="font-mono" />
              )}
              {openRow.auth_type === 'md5' && (
                <NumberRow title="Key ID" value={openRow.auth_key_id} onChange={(v) => upd(openRow._rowId, { auth_key_id: v })} placeholder="1" />
              )}
            </PreferencesGroup>
            <PreferencesGroup>
              <SwitchRow title="MTU Ignore" checked={!!openRow.mtu_ignore} onCheckedChange={(v) => upd(openRow._rowId, { mtu_ignore: v || undefined })} />
              <SwitchRow title="BFD" checked={!!openRow.bfd} onCheckedChange={(v) => upd(openRow._rowId, { bfd: v || undefined })} />
            </PreferencesGroup>
            <ExtraDirectives label="Additional Directives (ip ospf)" values={openRow.extra} onChange={(v) => upd(openRow._rowId, { extra: v })} placeholder="ip ospf authentication null" />
          </div>
        )}
      </Sheet>
    </div>
  )
}
