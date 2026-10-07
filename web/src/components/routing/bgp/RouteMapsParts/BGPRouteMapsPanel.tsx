import { RouteMapForm, RouteMapRow } from '@/components/routing/bgp/RouteMaps'
import { fromRecord, newId, toRecord } from '@/components/routing/shared'
import { RouteMapEntry } from '@/components/routing/types'
import { Button, EmptyState, Input, PreferencesColumns, Row, Sheet } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

type BGPRouteMapsPanelShape = {
  routeMaps: Record<string, RouteMapEntry[]>
  onChange: (v: Record<string, RouteMapEntry[]>) => void
  prefixListNames: string[]
  onDirty: () => void
}

export function BGPRouteMapsPanel({
  routeMaps, onChange, prefixListNames, onDirty,
}: BGPRouteMapsPanelShape) {
  const changeCallbacks = useRef({ onChange, onDirty })
  useEffect(() => { changeCallbacks.current = { onChange, onDirty } }, [onChange, onDirty])
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
    changeCallbacks.current.onChange(
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
    changeCallbacks.current.onDirty()
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
