import type { RenderImportedRouteMap } from '@/api'
import { RouteMapForm, RouteMapRow } from '@/components/routing/bgp/RouteMaps'
import { fromRecord, newId, toRecord } from '@/components/routing/shared'
import { RouteMapEntry } from '@/components/routing/types'
import { ImportedRouteMapDialog } from '@/components/routing/bgp/RouteMapsParts/ImportedRouteMapDialog'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Badge, Button, EmptyState, Input, Label, Row, Sheet, Switch, PreferencesGroup } from 'cheval-ui'
import { Plus, Share2, Trash2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

type BGPRouteMapsPanelShape = {
  routeMaps: Record<string, RouteMapEntry[]>
  onChange: (v: Record<string, RouteMapEntry[]>) => void
  shared: string[]
  onSharedChange: (v: string[]) => void
  prefixListNames: string[]
  onDirty: () => void
}

export function BGPRouteMapsPanel({
  routeMaps, onChange, shared, onSharedChange, prefixListNames, onDirty,
}: BGPRouteMapsPanelShape) {
  const toggleShared = (name: string, on: boolean) => {
    onSharedChange(on ? [...new Set([...shared, name])] : shared.filter((n) => n !== name))
    onDirty()
  }
  const { data: imported } = useFetch(() => api.apiFriendsBgpPoliciesGet())
  const importedMaps = imported?.route_maps ?? []
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
  const [importedOpen, setImportedOpen] = useState<RenderImportedRouteMap | null>(null)
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

  const deleteMap = (name: string) => {
    setLocal((p) => { const next = { ...p }; delete next[name]; return next })
    if (shared.includes(name)) onSharedChange(shared.filter((n) => n !== name))
  }

  const setRows = (name: string, rows: RouteMapRow[]) =>
    setLocal((p) => ({ ...p, [name]: rows }))

  const names = Object.keys(local)

  const addForm = (
    <div className="flex items-center gap-2">
      <Input
        value={newName}
        onChange={(e) => setNewName(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && addMap()}
        placeholder="Route map name (RM-IN)"
        className="font-mono text-sm max-w-xs"
      />
      <Button variant="outline" size="sm" onClick={addMap} disabled={!newName.trim()} className="gap-1.5">
        <Plus className="h-4 w-4" />Add Map
      </Button>
    </div>
  )

  return (
    <div className="space-y-4">
      {names.length === 0 ? (
        <EmptyState
          title="No Route Maps"
          action={addForm}
        />
      ) : (
        <>
          {addForm}
          <PreferencesGroup>
            {names.map((name) => {
              const rows = local[name]
              return (
                <Row
                  key={name}
                  title={(
                    <span className="flex items-center gap-2">
                      <Button variant="link" onClick={() => setOpenName(name)} className="h-auto p-0 font-mono text-foreground">
                        {name}
                      </Button>
                      {shared.includes(name) && (
                        <Badge variant="secondary" className="gap-1 text-xs"><Share2 className="h-3 w-3" />Shared</Badge>
                      )}
                    </span>
                  )}
                  subtitle={`${rows.length} ${rows.length === 1 ? 'entry' : 'entries'}`}
                >
                  <div className="flex items-center gap-1.5">
                    <Label htmlFor={`shared-rm-${name}`} className="text-xs text-muted-foreground cursor-pointer">Shared</Label>
                    <Switch id={`shared-rm-${name}`} checked={shared.includes(name)} onCheckedChange={(v) => toggleShared(name, v)} />
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6 text-muted-foreground hover:text-destructive"
                    onClick={() => deleteMap(name)}
                   aria-label="Delete Route Map" title="Delete Route Map">
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </Row>
              )
            })}
          </PreferencesGroup>
        </>
      )}

      {importedMaps.length > 0 && (
        <div className="space-y-2">
          <div className="text-xs font-semibold text-muted-foreground">Imported from Friends (read-only)</div>
          <PreferencesGroup>
            {importedMaps.map((rm) => (
              <Row
                key={rm.name}
                onClick={() => setImportedOpen(rm)}
                title={(
                  <span className="flex items-center gap-2">
                    <span className="font-mono">{rm.name}</span>
                    {rm.shared
                      ? <Badge variant="secondary" className="gap-1 text-xs"><Share2 className="h-3 w-3" />Shared</Badge>
                      : <Badge variant="outline" className="text-xs">{rm.sources.join(', ')}</Badge>}
                  </span>
                )}
                subtitle={`${rm.entries.length} ${rm.entries.length === 1 ? 'entry' : 'entries'}${rm.shared ? ` · from ${rm.sources.join(', ')}` : ''}`}
              />
            ))}
          </PreferencesGroup>
        </div>
      )}

      {importedOpen && <ImportedRouteMapDialog map={importedOpen} onClose={() => setImportedOpen(null)} />}

      <Sheet open={openName !== null} onClose={() => setOpenName(null)} title={openName ? `Route map ${openName}` : 'Route map'}>
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
