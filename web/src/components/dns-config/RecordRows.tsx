import { Checkbox } from '@/components/ui/Checkbox'
import { DnsRecord, DnsZone, fqdn, ownerFqdn, parseRecordLines, ptrExists, RECORD_TYPES, reverseTarget } from '@/components/dns-config/shared'
import { Button, Input, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Textarea, Label } from 'cheval-ui'
import { Plus, Search, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { AutoTextarea } from '@/components/ui/AutoTextarea'
import { plural } from '@/lib/utils'

type RecordRowsProps = {
  records: DnsRecord[]
  onChange: (r: DnsRecord[]) => void
  zoneName?: string
  zones?: DnsZone[]
  onZonesChange?: (z: DnsZone[]) => void
}

type RecordGroup = { items: IndexedRecord[] }

type IndexedRecord = { r: DnsRecord; i: number }

type RecordIndex = { i: number }

type ReverseRecord = { index: number; owner: string; value: string }

type ReverseMove = { from: ReverseRecord; to: ReverseRecord | null }

function sameReverse(a: ReverseRecord | null, b: ReverseRecord | null): boolean {
  return !!a && !!b && a.index === b.index && a.owner === b.owner && fqdn(a.value) === fqdn(b.value)
}

function isAddress(r: DnsRecord): boolean {
  return r.type === 'A' || r.type === 'AAAA'
}

function withoutPtr(zones: DnsZone[], ptr: ReverseRecord): DnsZone[] {
  return zones.map((z, i) => {
    if (i !== ptr.index) return z
    const records = (z.records ?? []).filter((x) => !(x.type === 'PTR' && x.name.trim() === ptr.owner && fqdn(x.value) === fqdn(ptr.value)))
    return { ...z, records: records.length ? records : undefined }
  })
}

function withPtr(zones: DnsZone[], ptr: ReverseRecord): DnsZone[] {
  if (ptrExists(zones[ptr.index], ptr.owner, ptr.value)) return zones
  return zones.map((z, i) => (i === ptr.index ? { ...z, records: [...(z.records ?? []), { name: ptr.owner, type: 'PTR', value: ptr.value }] } : z))
}

function remapIndices(indices: Set<number>, aligned: (DnsRecord | null)[]): Set<number> {
  const moved = new Set<number>()
  let kept = 0
  aligned.forEach((r, i) => {
    if (r === null) return
    if (indices.has(i)) moved.add(kept)
    kept++
  })
  return moved
}

export function RecordRows({ records, onChange, zoneName, zones, onZonesChange }: RecordRowsProps) {
  const [filter, setFilter] = useState('')
  const [pinned, setPinned] = useState<Set<number>>(new Set())
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [editing, setEditing] = useState<Set<number>>(new Set())
  const [pasteOpen, setPasteOpen] = useState(false)
  const [pasteText, setPasteText] = useState('')

  const reverseEnabled = !!(zoneName && zones && onZonesChange)

  const reverseOf = (r: DnsRecord): ReverseRecord | null => {
    if (!reverseEnabled || !isAddress(r) || !r.value.trim()) return null

    const target = reverseTarget(r.value, zones!)
    if (!target) return null

    return { index: target.index, owner: target.owner, value: ownerFqdn(r.name, zoneName!) }
  }

  const reverseMove = (r: DnsRecord, i: number, aligned: (DnsRecord | null)[], next: DnsRecord[]): ReverseMove[] => {
    const from = reverseOf(r)
    if (!from || !ptrExists(zones![from.index], from.owner, from.value)) return []

    const replacement = aligned[i]
    const to = replacement ? reverseOf(replacement) : null
    if (sameReverse(from, to)) return []
    if (replacement && !to && isAddress(replacement)) return []
    if (!to && next.some((candidate) => sameReverse(reverseOf(candidate), from))) return []

    return [{ from, to }]
  }

  const commit = (aligned: (DnsRecord | null)[]) => {
    const next = aligned.filter((r): r is DnsRecord => r !== null)
    const changed = aligned.flatMap((r, i) => (r !== null && r !== records[i] ? [i] : []))
    setPinned(remapIndices(new Set([...pinned, ...changed]), aligned))
    const forward = reverseEnabled ? zones!.findIndex((z) => z.name === zoneName) : -1
    const moves = forward < 0 ? [] : records.flatMap((r, i) => reverseMove(r, i, aligned, next))
    if (moves.length === 0) {
      onChange(next)
      return
    }

    let updated = zones!.map((z, i) => (i === forward ? { ...z, records: next.length ? next : undefined } : z))
    for (const { from, to } of moves) {
      updated = withoutPtr(updated, from)
      if (to) updated = withPtr(updated, to)
    }

    onZonesChange!(updated)
  }

  const set = (i: number, next: DnsRecord) => commit(records.map((r, j) => (j === i ? next : r)))

  const reverseInfoFor = (r: DnsRecord) => {
    if (!reverseEnabled || (r.type !== 'A' && r.type !== 'AAAA') || !r.value.trim()) return null

    const target = reverseTarget(r.value, zones!)
    if (!target) return null

    const value = ownerFqdn(r.name, zoneName!)
    return { ...target, value, checked: ptrExists(zones![target.index], target.owner, value) }
  }

  const toggleReverse = (r: DnsRecord, on: boolean) => {
    const target = reverseTarget(r.value, zones!)
    if (!target) return

    const value = ownerFqdn(r.name, zoneName!)
    const existing = zones![target.index].records ?? []
    const next = on
      ? [...existing, { name: target.owner, type: 'PTR', value }]
      : existing.filter((x) => !(x.type === 'PTR' && x.name.trim() === target.owner && fqdn(x.value) === fqdn(value)))
    if (on && ptrExists(zones![target.index], target.owner, value)) return

    onZonesChange!(zones!.map((z, i) => (i === target.index ? { ...z, records: next.length ? next : undefined } : z)))
  }

  const q = filter.trim().toLowerCase()
  const shown = records
    .map((r, i) => ({ r, i }))
    .filter(({ r, i }) => !q || pinned.has(i) || editing.has(i) || `${r.name} ${r.type} ${r.value}`.toLowerCase().includes(q))

  const add = (added: DnsRecord[]) => {
    onChange([...records, ...added])
    setPinned((prev) => new Set([...prev, ...added.map((_, k) => records.length + k)]))
  }

  const search = (value: string) => {
    setFilter(value)
    setPinned(new Set())
  }

  const groups: RecordGroup[] = []
  const groupOf = new Map<string, number>()
  for (const item of shown) {
    const key = editing.has(item.i) ? '\0editing' : item.r.name.trim().toLowerCase()
    let g = groupOf.get(key)
    if (g === undefined) {
      g = groups.length
      groupOf.set(key, g)
      groups.push({ items: [] })
    }

    groups[g].items.push(item)
  }

  const toggle = (i: number) => setSelected((prev) => {
    const next = new Set(prev)
    if (next.has(i)) {
      next.delete(i)
    } else {
      next.add(i)
    }
    return next
  })

  const allShownSelected = shown.length > 0 && shown.every(({ i }) => selected.has(i))
  const toggleAll = () => setSelected((prev) => {
    if (allShownSelected) {
      const next = new Set(prev)
      shown.forEach(({ i }) => next.delete(i))
      return next
    }
    return new Set([...prev, ...shown.map(({ i }) => i)])
  })

  const groupSelected = (items: RecordIndex[]) => items.length > 0 && items.every(({ i }) => selected.has(i))
  const toggleGroup = (items: RecordIndex[]) => setSelected((prev) => {
    const next = new Set(prev)
    if (groupSelected(items)) {
      items.forEach(({ i }) => next.delete(i))
    } else {
      items.forEach(({ i }) => next.add(i))
    }
    return next
  })

  const renameGroup = (items: RecordIndex[], name: string) => {
    const idx = new Set(items.map(({ i }) => i))
    commit(records.map((r, j) => (idx.has(j) ? { ...r, name } : r)))
  }

  const deleteSelected = () => {
    commit(records.map((r, i) => (selected.has(i) ? null : r)))
    setSelected(new Set())
  }

  const parsed = parseRecordLines(pasteText)
  const append = () => {
    if (parsed.records.length === 0) return
    add(parsed.records)
    setPasteText('')
    setPasteOpen(false)
  }

  return (
    <div className="space-y-2">
      {records.length > 0 && (
        <div className="flex items-center gap-2">
          <Checkbox
            className="shrink-0"
            checked={allShownSelected}
            onCheckedChange={toggleAll}
            title="Select all"
            aria-label="Select all records"
          />
          <div className="relative flex-1">
            <Search className="pointer-events-none absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="h-8 pl-7 font-mono text-xs"
              placeholder={`Filter ${records.length} records`}
              value={filter}
              onChange={(e) => search(e.target.value)}
            />
          </div>
          {selected.size > 0 && (
            <Button variant="destructive" size="sm" onClick={deleteSelected}>
              <Trash2 className="mr-1 h-3 w-3" /> Delete {selected.size}
            </Button>
          )}
        </div>
      )}

      {groups.map((g) => {
        const name = g.items[0].r.name
        const ownerKey = (n: string) => (zoneName ? ownerFqdn(n, zoneName) : n.trim().toLowerCase())
        const ownerRecords = records.filter((r) => ownerKey(r.name) === ownerKey(name))
        const cnameCount = ownerRecords.filter((r) => r.type.toUpperCase() === 'CNAME').length
        const cnameError = cnameCount > 1
          ? 'Only one CNAME target is allowed per name. For multiple addresses, use A/AAAA records.'
          : cnameCount > 0 && ownerRecords.length > 1
            ? 'A CNAME cannot share its name with other records.'
            : null
        return (
          <div key={g.items[0].i} className="space-y-2 rounded-xl bg-card p-3 [box-shadow:var(--card-shadow)]">
            <div className="flex items-center gap-2">
              <Checkbox
                className="shrink-0"
                aria-label={`Select records for ${name}`}
                checked={groupSelected(g.items)}
                onCheckedChange={() => toggleGroup(g.items)}
              />
              <AutoTextarea
                className="min-w-0 flex-1 font-mono text-xs sm:w-56 sm:flex-none"
                placeholder="@"
                value={name}
                onFocus={() => setEditing(new Set(g.items.map(({ i }) => i)))}
                onBlur={() => setEditing(new Set())}
                onChange={(v) => renameGroup(g.items, v)}
              />
              <span className="hidden text-xs text-muted-foreground sm:inline">{plural(g.items.length, 'record')}</span>
              <div className="hidden flex-1 sm:block" />
              <Button
                variant="ghost"
                size="sm"
                className="h-7 gap-1 text-xs text-muted-foreground"
                onClick={() => add([{ name, type: 'A', value: '' }])}
              >
                <Plus className="h-3 w-3" /> Add Type
              </Button>
            </div>

            {cnameError && <p role="alert" className="text-xs text-danger sm:pl-6">{cnameError}</p>}

            {g.items.map(({ r, i }) => {
              const rev = reverseInfoFor(r)
              return (
                <div key={i} className="flex flex-wrap items-center gap-2 border-t border-border/60 pt-2 sm:flex-nowrap sm:border-0 sm:pt-0 sm:pl-6">
                  <Checkbox
                    className="shrink-0"
                    aria-label={`Select ${r.type} record for ${r.name}`}
                    checked={selected.has(i)}
                    onCheckedChange={() => toggle(i)}
                  />
                  <Select value={r.type || 'A'} onValueChange={(v) => set(i, { ...r, type: v })}>
                    <SelectTrigger className="h-8 w-24 text-xs"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {RECORD_TYPES.map((t) => <SelectItem key={t} value={t} className="font-mono text-xs">{t}</SelectItem>)}
                    </SelectContent>
                  </Select>
                  {(r.type === 'MX' || r.type === 'SRV') && (
                    <Input
                      className="h-8 w-20 font-mono text-xs"
                      placeholder="prio"
                      value={r.priority ?? ''}
                      onChange={(e) => set(i, { ...r, priority: e.target.value ? Number(e.target.value) : undefined })}
                    />
                  )}
                  <AutoTextarea
                    className="order-last w-full font-mono text-xs sm:order-none sm:w-auto sm:flex-1"
                    placeholder="value"
                    value={r.value}
                    onChange={(v) => set(i, { ...r, value: v })}
                  />
                  <div className="flex-1 sm:hidden" />
                  {rev && (
                    <Label className="flex shrink-0 items-center gap-1 text-xs text-muted-foreground" title={`Create Reverse Record in ${rev.zoneName}`}>
                      <Checkbox
                        className="shrink-0"
                        aria-label="Create Reverse Record"
                        checked={rev.checked}
                        onCheckedChange={(checked) => toggleReverse(r, checked)}
                      />
                      PTR
                    </Label>
                  )}
                  <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => { commit(records.map((x, j) => (j === i ? null : x))); setSelected(new Set()) }} aria-label="Remove Record" title="Remove Record">
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              )
            })}
          </div>
        )
      })}

      {q && shown.length === 0 && (
        <p className="text-xs text-muted-foreground">No records match "{filter}".</p>
      )}

      <div className="flex gap-2">
        <Button variant="outline" size="sm" onClick={() => add([{ name: '@', type: 'A', value: '' }])}>
          <Plus className="mr-1 h-3 w-3" /> Add Record
        </Button>
        <Button variant="outline" size="sm" onClick={() => setPasteOpen((v) => !v)}>
          Paste Records
        </Button>
      </div>

      {pasteOpen && (
        <div className="space-y-2 rounded-md bg-muted/30 p-2">
          <Textarea
            className="min-h-[96px] font-mono text-xs"
            placeholder={'ns1    A     100.64.0.2\nwww    CNAME @\nmail   MX    10 mail.example.net.'}
            value={pasteText}
            onChange={(e) => setPasteText(e.target.value)}
          />
          <div className="flex items-center gap-2">
            <Button size="sm" onClick={append} disabled={parsed.records.length === 0}>
              Append {parsed.records.length || ''}
            </Button>
            <p className="text-xs text-muted-foreground">
              One record per line: <code>name [ttl] type value</code>
              {parsed.errors.length > 0 && (
                <span className="text-warning">, {plural(parsed.errors.length, 'line')} not parsed</span>
              )}
            </p>
          </div>
        </div>
      )}
    </div>
  )
}
