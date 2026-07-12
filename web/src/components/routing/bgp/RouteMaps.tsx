import { useState, useEffect, useRef } from 'react'
import { Input } from '@/components/ui/input'
import { NumberInput } from '@/components/ui/number-input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/EmptyState'
import { Sheet } from '@/components/ui/sheet'
import { Plus, Trash2 } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { PreferencesGroup, PreferencesColumns, Row } from '@/components/Preferences'
import { Segmented } from '@/components/ui/segmented'
import { RouteMapEntry } from '../types'
import { KVPair, toRecord, fromRecord, newId } from '../shared'
import { ClauseEditor, MATCH_OPS, SET_OPS } from '../ClauseEditor'

export type RouteMapRow = {
  _id: number
  seq: number
  action: 'permit' | 'deny'
  match: KVPair[]
  set: KVPair[]
  call: string
  on_match: string
  continue: number
}

export function RouteMapEntryCard({
  row, onChange, onDelete, prefixListNames, callTargets,
}: {
  row: RouteMapRow
  onChange: (patch: Partial<RouteMapRow>) => void
  onDelete: () => void
  prefixListNames: string[]
  callTargets: string[]
}) {
  const onMatchMode: 'none' | 'next' | 'goto' =
    row.on_match === 'next' ? 'next' : row.on_match.startsWith('goto') ? 'goto' : 'none'
  const gotoSeq = onMatchMode === 'goto' ? Number(row.on_match.slice(5)) || 0 : 0

  return (
    <PreferencesGroup>
      <div className="flex items-end gap-4 px-4 py-3">
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">Seq</Label>
          <NumberInput
            value={row.seq || undefined}
            onChange={(v) => onChange({ seq: v ?? 0 })}
            className="font-mono h-8 text-xs w-24"
          />
        </div>
        <div className="space-y-1 mr-4">
          <Label className="text-xs text-muted-foreground">Action</Label>
          <Segmented
            value={row.action}
            onChange={(v) => onChange({ action: v })}
            options={[
              { value: 'permit', label: 'permit', activeClass: 'bg-success text-success-foreground' },
              { value: 'deny', label: 'deny', activeClass: 'bg-danger text-danger-foreground' },
            ]}
          />
        </div>
        <div className="flex-1" />
        <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-destructive" onClick={onDelete}>
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>

      <div className="px-4 py-3 space-y-4">
        <div className="space-y-1.5">
          <Label className="text-xs font-semibold text-muted-foreground">Match</Label>
          <ClauseEditor pairs={row.match} onChange={(v) => onChange({ match: v })} catalog={MATCH_OPS} prefixListNames={prefixListNames} verb="match" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs font-semibold text-muted-foreground">Set</Label>
          <ClauseEditor pairs={row.set} onChange={(v) => onChange({ set: v })} catalog={SET_OPS} prefixListNames={prefixListNames} verb="set" />
        </div>
      </div>

      <div className="px-4 py-3 space-y-3">
        <Label className="text-xs font-semibold text-muted-foreground">Flow control</Label>
        <div className="flex flex-wrap items-end gap-4">
          <div className="space-y-1">
            <Label className="text-xs text-muted-foreground">Call route-map</Label>
            <Select value={row.call || '_none'} onValueChange={(v) => onChange({ call: v === '_none' ? '' : v })}>
              <SelectTrigger className="h-8 text-xs font-mono w-44"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="_none">- none -</SelectItem>
                {callTargets.map((n) => <SelectItem key={n} value={n} className="font-mono text-xs">{n}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1 mr-4">
            <Label className="text-xs text-muted-foreground">On match</Label>
            <Segmented
              value={onMatchMode}
              onChange={(m) => onChange({ on_match: m === 'none' ? '' : m === 'next' ? 'next' : `goto ${gotoSeq || row.seq + 10}` })}
              options={[{ value: 'none', label: 'default' }, { value: 'next', label: 'next' }, { value: 'goto', label: 'goto' }]}
            />
          </div>
          {onMatchMode === 'goto' && (
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">Goto seq</Label>
              <NumberInput
                value={gotoSeq || undefined}
                onChange={(v) => onChange({ on_match: `goto ${v ?? 0}` })}
                className="font-mono h-8 text-xs w-24"
              />
            </div>
          )}
          <div className="space-y-1">
            <Label className="text-xs text-muted-foreground">Continue seq</Label>
            <NumberInput
              value={row.continue || undefined}
              onChange={(v) => onChange({ continue: v ?? 0 })}
              placeholder="–"
              className="font-mono h-8 text-xs w-24"
            />
          </div>
        </div>
      </div>
    </PreferencesGroup>
  )
}

export function RouteMapForm({
  rows, onChange, prefixListNames, callTargets, onDone,
}: {
  rows: RouteMapRow[]
  onChange: (v: RouteMapRow[]) => void
  prefixListNames: string[]
  callTargets: string[]
  onDone: () => void
}) {
  const updEntry = (id: number, patch: Partial<RouteMapRow>) =>
    onChange(rows.map((r) => (r._id === id ? { ...r, ...patch } : r)))
  const delEntry = (id: number) => onChange(rows.filter((r) => r._id !== id))
  const addEntry = () => {
    const maxSeq = rows.reduce((m, r) => Math.max(m, r.seq), 0)
    onChange([...rows, { _id: newId(), seq: maxSeq + 10, action: 'permit', match: [], set: [], call: '', on_match: '', continue: 0 }])
  }

  return (
    <div className="space-y-4">
      {rows.length === 0 ? (
        <p className="text-sm text-muted-foreground italic px-1">No entries. Add one to define match/set rules.</p>
      ) : (
        [...rows].sort((a, b) => a.seq - b.seq).map((row) => (
          <RouteMapEntryCard
            key={row._id}
            row={row}
            onChange={(patch) => updEntry(row._id, patch)}
            onDelete={() => delEntry(row._id)}
            prefixListNames={prefixListNames}
            callTargets={callTargets}
          />
        ))
      )}
      <div className="flex items-center justify-between pt-1">
        <Button variant="outline" size="sm" className="gap-1.5" onClick={addEntry}>
          <Plus className="h-3.5 w-3.5" />Add Entry
        </Button>
        <Button variant="outline" onClick={onDone}>Done</Button>
      </div>
    </div>
  )
}

export function BGPRouteMapsPanel({
  routeMaps, onChange, prefixListNames, onDirty,
}: {
  routeMaps: Record<string, RouteMapEntry[]>
  onChange: (v: Record<string, RouteMapEntry[]>) => void
  prefixListNames: string[]
  onDirty: () => void
}) {
  const [local, setLocal] = useState<Record<string, RouteMapRow[]>>(() =>
    Object.fromEntries(
      Object.entries(routeMaps).map(([name, entries]) => [
        name,
        entries.map((e) => ({
          _id: newId(),
          seq: e.seq,
          action: e.action,
          match: fromRecord(e.match),
          set: fromRecord(e.set),
          call: e.call ?? '',
          on_match: e.on_match ?? '',
          continue: e.continue ?? 0,
        })),
      ])
    )
  )
  const [newName, setNewName] = useState('')
  const [openName, setOpenName] = useState<string | null>(null)
  const firstRun = useRef(true)

  useEffect(() => {
    if (firstRun.current) { firstRun.current = false; return }
    onChange(
      Object.fromEntries(
        Object.entries(local).map(([name, rows]) => [
          name,
          rows.map(({ _id: _, match, set, call, on_match, continue: cont, ...rest }) => ({
            ...rest,
            match: toRecord(match),
            set: toRecord(set),
            ...(call ? { call } : {}),
            ...(on_match ? { on_match } : {}),
            ...(cont ? { continue: cont } : {}),
          } as RouteMapEntry)),
        ])
      )
    )
    onDirty()
  }, [local])

  const addMap = () => {
    const name = newName.trim().toUpperCase()
    if (!name || name in local) return
    setLocal((p) => ({ ...p, [name]: [] }))
    setNewName('')
    setOpenName(name)
  }

  const deleteMap = (name: string) =>
    setLocal((p) => { const next = { ...p }; delete next[name]; return next })

  const setRows = (name: string, rows: RouteMapRow[]) =>
    setLocal((p) => ({ ...p, [name]: rows }))

  const names = Object.keys(local)

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Input
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && addMap()}
          placeholder="Route map name (RM-IN)"
          className="font-mono text-sm max-w-xs"
        />
        <Button variant="outline" size="sm" onClick={addMap} disabled={!newName.trim()} className="gap-1.5">
          <Plus className="h-3.5 w-3.5" />Add Map
        </Button>
      </div>

      {names.length === 0 ? (
        <EmptyState
          title="No route maps"
          message="Route maps transform and filter routes exchanged with neighbors."
          action={<Button variant="outline" size="sm" onClick={addMap} className="gap-2"><Plus className="h-4 w-4" />Add map</Button>}
        />
      ) : (
        <PreferencesColumns>
          {names.map((name) => {
            const rows = local[name]
            return (
              <Row
                key={name}
                title={<span className="font-mono">{name}</span>}
                subtitle={`${rows.length} ${rows.length === 1 ? 'entry' : 'entries'}`}
                onClick={() => setOpenName(name)}
              >
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 text-muted-foreground hover:text-destructive"
                  onClick={(e) => { e.stopPropagation(); deleteMap(name) }}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </Row>
            )
          })}
        </PreferencesColumns>
      )}

      <Sheet open={openName !== null} onClose={() => setOpenName(null)} title={openName ? `Route map ${openName}` : 'Route map'} className="max-w-2xl">
        {openName !== null && local[openName] && (
          <RouteMapForm
            rows={local[openName]}
            onChange={(rows) => setRows(openName, rows)}
            prefixListNames={prefixListNames}
            callTargets={names.filter((n) => n !== openName)}
            onDone={() => setOpenName(null)}
          />
        )}
      </Sheet>
    </div>
  )
}

