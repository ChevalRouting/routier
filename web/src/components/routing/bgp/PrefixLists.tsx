import { useState, useEffect, useRef } from 'react'
import { Input } from 'cheval-ui'
import { NumberInput } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { Sheet } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { PreferencesColumns, Row } from 'cheval-ui'
import { PrefixEntry } from '../types'
import { newId } from '../shared'

export type PrefixListRow = PrefixEntry & { _id: number }

export const PREFIX_PRESETS: { label: string; desc: string; entries: Omit<PrefixEntry, 'seq'>[] }[] = [
  { label: 'Any', desc: 'permit 0.0.0.0/0 le 32', entries: [{ action: 'permit', prefix: '0.0.0.0/0', le: 32 }] },
  { label: 'Default only', desc: 'permit 0.0.0.0/0', entries: [{ action: 'permit', prefix: '0.0.0.0/0' }] },
  { label: 'Default only (v6)', desc: 'permit ::/0', entries: [{ action: 'permit', prefix: '::/0' }] },
  { label: 'Deny default', desc: 'deny 0.0.0.0/0', entries: [{ action: 'deny', prefix: '0.0.0.0/0' }] },
  {
    label: 'RFC1918', desc: 'permit the three private ranges',
    entries: [
      { action: 'permit', prefix: '10.0.0.0/8', le: 32 },
      { action: 'permit', prefix: '172.16.0.0/12', le: 32 },
      { action: 'permit', prefix: '192.168.0.0/16', le: 32 },
    ],
  },
]

export function PrefixListForm({
  rows, onChange,
}: {
  rows: PrefixListRow[]
  onChange: (v: PrefixListRow[]) => void
}) {
  const nextSeq = () => rows.reduce((m, r) => Math.max(m, r.seq), 0) + 10
  const upd = (id: number, patch: Partial<PrefixListRow>) =>
    onChange(rows.map((r) => (r._id === id ? { ...r, ...patch } : r)))
  const del = (id: number) => onChange(rows.filter((r) => r._id !== id))
  const addEntry = () => onChange([...rows, { _id: newId(), seq: nextSeq(), action: 'permit', prefix: '' }])
  const applyPreset = (entries: Omit<PrefixEntry, 'seq'>[]) => {
    let seq = nextSeq()
    onChange([...rows, ...entries.map((e) => ({ ...e, _id: newId(), seq: (seq += 10) - 10 }))])
  }

  return (
    <div className="space-y-3">
      <div className="space-y-2">
        <Label className="text-xs font-semibold text-muted-foreground">Quick add</Label>
        <div className="flex flex-wrap gap-1.5">
          {PREFIX_PRESETS.map((p) => (
            <Button key={p.label} variant="outline" size="sm" className="h-7 text-xs gap-1" title={p.desc} onClick={() => applyPreset(p.entries)}>
              <Plus className="h-3 w-3" />{p.label}
            </Button>
          ))}
        </div>
      </div>

      {rows.length > 0 && (
        <div className="rounded bg-card shadow-[var(--card-shadow)] overflow-x-auto">
          <table className="w-full text-xs">
            <thead className="bg-muted/50">
              <tr>
                <th className="text-left px-2 py-1.5 font-medium text-muted-foreground w-16">Seq</th>
                <th className="text-left px-2 py-1.5 font-medium text-muted-foreground w-24">Action</th>
                <th className="text-left px-2 py-1.5 font-medium text-muted-foreground">Prefix</th>
                <th className="text-left px-2 py-1.5 font-medium text-muted-foreground w-16">≥ ge</th>
                <th className="text-left px-2 py-1.5 font-medium text-muted-foreground w-16">≤ le</th>
                <th className="w-8" />
              </tr>
            </thead>
            <tbody className="divide-y">
              {[...rows].sort((a, b) => a.seq - b.seq).map((row) => (
                <tr key={row._id}>
                  <td className="px-2 py-1">
                    <NumberInput
                      value={row.seq || undefined}
                      onChange={(v) => upd(row._id, { seq: v ?? 0 })}
                      className="font-mono h-7 text-xs border-0 shadow-none focus-visible:ring-1 w-14"
                    />
                  </td>
                  <td className="px-2 py-1">
                    <Select value={row.action} onValueChange={(v) => upd(row._id, { action: v as 'permit' | 'deny' })}>
                      <SelectTrigger className="h-7 text-xs font-mono w-20"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="permit">permit</SelectItem>
                        <SelectItem value="deny">deny</SelectItem>
                      </SelectContent>
                    </Select>
                  </td>
                  <td className="px-2 py-1">
                    <Input
                      value={row.prefix}
                      onChange={(e) => upd(row._id, { prefix: e.target.value })}
                      placeholder="10.0.0.0/8 or any"
                      className="font-mono h-7 text-xs border-0 shadow-none focus-visible:ring-1"
                    />
                  </td>
                  <td className="px-2 py-1">
                    <NumberInput
                      value={row.ge || undefined}
                      onChange={(v) => upd(row._id, { ge: v })}
                      placeholder="–"
                      className="font-mono h-7 text-xs border-0 shadow-none focus-visible:ring-1 w-14"
                    />
                  </td>
                  <td className="px-2 py-1">
                    <NumberInput
                      value={row.le || undefined}
                      onChange={(v) => upd(row._id, { le: v })}
                      placeholder="–"
                      className="font-mono h-7 text-xs border-0 shadow-none focus-visible:ring-1 w-14"
                    />
                  </td>
                  <td className="px-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 text-muted-foreground hover:text-destructive"
                      onClick={() => del(row._id)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <Button variant="outline" size="sm" className="h-7 text-xs gap-1" onClick={addEntry}>
        <Plus className="h-3 w-3" />Add Entry
      </Button>
    </div>
  )
}

export function BGPPrefixListsPanel({
  prefixLists, onChange, onDirty,
}: {
  prefixLists: Record<string, PrefixEntry[]>
  onChange: (v: Record<string, PrefixEntry[]>) => void
  onDirty: () => void
}) {
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
    onChange(
      Object.fromEntries(
        Object.entries(local).map(([name, rows]) => [
          name,
          rows.map(({ _id: _, ...rest }) => rest as PrefixEntry),
        ])
      )
    )
    onDirty()
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
          action={<Button variant="outline" size="sm" onClick={addList} className="gap-2"><Plus className="h-4 w-4" />Add list</Button>}
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
