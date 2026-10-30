import { PREFIX_PRESETS, PrefixListRow } from '@/components/routing/bgp/PrefixLists'
import { newId } from '@/components/routing/shared'
import { PrefixEntry } from '@/components/routing/types'
import { Button, Input, Label, NumberInput, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'
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
        <Label className="text-xs font-semibold text-muted-foreground">Quick Add</Label>
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
          <Table className="w-full text-xs">
            <TableHeader className="bg-muted/50">
              <TableRow>
                <TableHead className="text-left px-2 py-1.5 font-medium text-muted-foreground w-16">Seq</TableHead>
                <TableHead className="text-left px-2 py-1.5 font-medium text-muted-foreground w-24">Action</TableHead>
                <TableHead className="text-left px-2 py-1.5 font-medium text-muted-foreground">Prefix</TableHead>
                <TableHead className="text-left px-2 py-1.5 font-medium text-muted-foreground w-16">≥ ge</TableHead>
                <TableHead className="text-left px-2 py-1.5 font-medium text-muted-foreground w-16">≤ le</TableHead>
                <TableHead className="w-8" />
              </TableRow>
            </TableHeader>
            <TableBody className="divide-y">
              {[...rows].sort((a, b) => a.seq - b.seq).map((row) => (
                <TableRow key={row._id}>
                  <TableCell className="px-2 py-1">
                    <NumberInput
                      value={row.seq || undefined}
                      onChange={(v) => upd(row._id, { seq: v ?? 0 })}
                      className="font-mono h-7 text-xs border-0 shadow-none focus-visible:ring-1 w-14"
                    />
                  </TableCell>
                  <TableCell className="px-2 py-1">
                    <Select value={row.action} onValueChange={(v) => upd(row._id, { action: v as 'permit' | 'deny' })}>
                      <SelectTrigger className="h-7 text-xs font-mono w-20"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="permit">permit</SelectItem>
                        <SelectItem value="deny">deny</SelectItem>
                      </SelectContent>
                    </Select>
                  </TableCell>
                  <TableCell className="px-2 py-1">
                    <Input
                      value={row.prefix}
                      onChange={(e) => upd(row._id, { prefix: e.target.value })}
                      placeholder="10.0.0.0/8 or any"
                      className="font-mono h-7 text-xs border-0 shadow-none focus-visible:ring-1"
                    />
                  </TableCell>
                  <TableCell className="px-2 py-1">
                    <NumberInput
                      value={row.ge || undefined}
                      onChange={(v) => upd(row._id, { ge: v })}
                      placeholder="–"
                      className="font-mono h-7 text-xs border-0 shadow-none focus-visible:ring-1 w-14"
                    />
                  </TableCell>
                  <TableCell className="px-2 py-1">
                    <NumberInput
                      value={row.le || undefined}
                      onChange={(v) => upd(row._id, { le: v })}
                      placeholder="–"
                      className="font-mono h-7 text-xs border-0 shadow-none focus-visible:ring-1 w-14"
                    />
                  </TableCell>
                  <TableCell className="px-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 text-muted-foreground hover:text-destructive"
                      onClick={() => del(row._id)}
                     aria-label="Remove Entry" title="Remove Entry">
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
      <Button variant="outline" size="sm" className="h-7 text-xs gap-1" onClick={addEntry}>
        <Plus className="h-3 w-3" />Add Entry
      </Button>
    </div>
  )
}
