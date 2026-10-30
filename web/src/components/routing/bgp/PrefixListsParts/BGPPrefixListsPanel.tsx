import type { RenderImportedPrefixList } from '@/api'
import { PrefixListForm, PrefixListRow } from '@/components/routing/bgp/PrefixLists'
import { newId } from '@/components/routing/shared'
import { PrefixEntry } from '@/components/routing/types'
import { ImportedPrefixListDialog } from '@/components/routing/bgp/PrefixListsParts/ImportedPrefixListDialog'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Badge, Button, EmptyState, Input, Label, Row, Sheet, Switch, PreferencesGroup } from 'cheval-ui'
import { Plus, Share2, Trash2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

type BGPPrefixListsPanelShape = {
  prefixLists: Record<string, PrefixEntry[]>
  onChange: (v: Record<string, PrefixEntry[]>) => void
  shared: string[]
  onSharedChange: (v: string[]) => void
  onDirty: () => void
}

export function BGPPrefixListsPanel({
  prefixLists, onChange, shared, onSharedChange, onDirty,
}: BGPPrefixListsPanelShape) {
  const toggleShared = (name: string, on: boolean) => {
    onSharedChange(on ? [...new Set([...shared, name])] : shared.filter((n) => n !== name))
    onDirty()
  }
  const { data: imported } = useFetch(() => api.apiFriendsBgpPoliciesGet())
  const importedLists = imported?.prefix_lists ?? []
  const changeCallbacks = useRef({ onChange, onDirty })
  useEffect(() => { changeCallbacks.current = { onChange, onDirty } }, [onChange, onDirty])
  const [local, setLocal] = useState<Record<string, PrefixListRow[]>>(() =>
    Object.fromEntries(
      Object.entries(prefixLists).map(([name, entries]) => [
        name,
        entries.map((e) => ({ ...e, _id: newId() })),
      ])
    )
  )
  const [newName, setNewName] = useState('')
  const [openName, setOpenName] = useState<string | null>(null)
  const [importedOpen, setImportedOpen] = useState<RenderImportedPrefixList | null>(null)
  const firstRun = useRef(true)

  useEffect(() => {
    if (firstRun.current) { firstRun.current = false; return }
    changeCallbacks.current.onChange(
      Object.fromEntries(
        Object.entries(local).map(([name, rows]) => [
          name,
          rows.map(({ _id: _, ...rest }) => rest as PrefixEntry),
        ])
      )
    )
    changeCallbacks.current.onDirty()
  }, [local])

  const addList = () => {
    const name = newName.trim().toUpperCase()
    if (!name || name in local) return
    setLocal((p) => ({ ...p, [name]: [] }))
    setNewName('')
    setOpenName(name)
  }

  const deleteList = (name: string) => {
    setLocal((p) => { const next = { ...p }; delete next[name]; return next })
    if (shared.includes(name)) onSharedChange(shared.filter((n) => n !== name))
  }

  const setRows = (name: string, rows: PrefixListRow[]) =>
    setLocal((p) => ({ ...p, [name]: rows }))

  const names = Object.keys(local)

  const addForm = (
    <div className="flex items-center gap-2">
      <Input
        value={newName}
        onChange={(e) => setNewName(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && addList()}
        placeholder="List name (MY-LIST-IN)"
        className="font-mono text-sm max-w-xs"
      />
      <Button variant="outline" size="sm" onClick={addList} disabled={!newName.trim()} className="gap-1.5">
        <Plus className="h-4 w-4" />Add List
      </Button>
    </div>
  )

  return (
    <div className="space-y-4">
      {names.length === 0 ? (
        <EmptyState
          title="No Prefix Lists"
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
                    <Label htmlFor={`shared-pl-${name}`} className="text-xs text-muted-foreground cursor-pointer">Shared</Label>
                    <Switch id={`shared-pl-${name}`} checked={shared.includes(name)} onCheckedChange={(v) => toggleShared(name, v)} />
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6 text-muted-foreground hover:text-destructive"
                    onClick={() => deleteList(name)}
                   aria-label="Delete Prefix List" title="Delete Prefix List">
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </Row>
              )
            })}
          </PreferencesGroup>
        </>
      )}

      {importedLists.length > 0 && (
        <div className="space-y-2">
          <div className="text-xs font-semibold text-muted-foreground">Imported from Friends (read-only)</div>
          <PreferencesGroup>
            {importedLists.map((pl) => (
              <Row
                key={pl.name}
                onClick={() => setImportedOpen(pl)}
                title={(
                  <span className="flex items-center gap-2">
                    <span className="font-mono">{pl.name}</span>
                    {pl.shared
                      ? <Badge variant="secondary" className="gap-1 text-xs"><Share2 className="h-3 w-3" />Shared</Badge>
                      : <Badge variant="outline" className="text-xs">{pl.sources.join(', ')}</Badge>}
                  </span>
                )}
                subtitle={`${pl.entries.length} ${pl.entries.length === 1 ? 'entry' : 'entries'}${pl.shared ? ` · from ${pl.sources.join(', ')}` : ''}`}
              />
            ))}
          </PreferencesGroup>
        </div>
      )}

      {importedOpen && <ImportedPrefixListDialog list={importedOpen} onClose={() => setImportedOpen(null)} />}

      <Sheet open={openName !== null} onClose={() => setOpenName(null)} title={openName ? `Prefix list ${openName}` : 'Prefix list'}>
        {openName !== null && local[openName] && (
          <PrefixListForm rows={local[openName]} onChange={(rows) => setRows(openName, rows)} />
        )}
      </Sheet>
    </div>
  )
}
