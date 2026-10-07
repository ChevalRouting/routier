import { PREFIX_PRESETS, PrefixListRow } from '@/components/routing/bgp/PrefixLists'
import { newId } from '@/components/routing/shared'
import { PrefixEntry } from '@/components/routing/types'
import { Button, Input, Label, NumberInput, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'

type PrefixListFormShape = {
  rows: PrefixListRow[]
  onChange: (v: PrefixListRow[]) => void
}

export function PrefixListForm({
  rows, onChange,
}: PrefixListFormShape) {
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
