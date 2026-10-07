import { PrefixListForm, PrefixListRow } from '@/components/routing/bgp/PrefixLists'
import { newId } from '@/components/routing/shared'
import { PrefixEntry } from '@/components/routing/types'
import { Button, EmptyState, Input, PreferencesColumns, Row, Sheet } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

type BGPPrefixListsPanelShape = {
  prefixLists: Record<string, PrefixEntry[]>
  onChange: (v: Record<string, PrefixEntry[]>) => void
  onDirty: () => void
}

export function BGPPrefixListsPanel({
  prefixLists, onChange, onDirty,
}: BGPPrefixListsPanelShape) {
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

  const deleteList = (name: string) =>
    setLocal((p) => { const next = { ...p }; delete next[name]; return next })

  const setRows = (name: string, rows: PrefixListRow[]) =>
    setLocal((p) => ({ ...p, [name]: rows }))

  const names = Object.keys(local)

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Input
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && addList()}
          placeholder="List name (MY-LIST-IN)"
          className="font-mono text-sm max-w-xs"
        />
        <Button variant="outline" size="sm" onClick={addList} disabled={!newName.trim()} className="gap-1.5">
          <Plus className="h-3.5 w-3.5" />Add List
        </Button>
      </div>

      {names.length === 0 ? (
        <EmptyState
          title="No prefix lists"
          message="Prefix lists match route prefixes for filtering in route maps and neighbor policies."
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
                  onClick={(e) => { e.stopPropagation(); deleteList(name) }}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </Row>
            )
          })}
        </PreferencesColumns>
      )}

      <Sheet open={openName !== null} onClose={() => setOpenName(null)} title={openName ? `Prefix list ${openName}` : 'Prefix list'} className="max-w-2xl">
        {openName !== null && local[openName] && (
          <PrefixListForm rows={local[openName]} onChange={(rows) => setRows(openName, rows)} />
        )}
      </Sheet>
    </div>
  )
}
