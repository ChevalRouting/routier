import { DnsRecord, DnsZone, fqdn, ownerFqdn, parseRecordLines, ptrExists, RECORD_TYPES, reverseTarget } from '@/components/dns-config/shared'
import { Button, Input, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Textarea, Switch } from 'cheval-ui'
import { Plus, Search, Trash2 } from 'lucide-react'
import { useState } from 'react'

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

export function RecordRows({ records, onChange, zoneName, zones, onZonesChange }: RecordRowsProps) {
  const [filter, setFilter] = useState('')
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [editing, setEditing] = useState<Set<number>>(new Set())
  const [pasteOpen, setPasteOpen] = useState(false)
  const [pasteText, setPasteText] = useState('')

  const set = (i: number, next: DnsRecord) => onChange(records.map((r, j) => (j === i ? next : r)))

  const reverseEnabled = !!(zoneName && zones && onZonesChange)

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
    .filter(({ r }) => !q || `${r.name} ${r.type} ${r.value}`.toLowerCase().includes(q))

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
    onChange(records.map((r, j) => (idx.has(j) ? { ...r, name } : r)))
  }

  const deleteSelected = () => {
    onChange(records.filter((_, i) => !selected.has(i)))
    setSelected(new Set())
  }

  const parsed = parseRecordLines(pasteText)
  const append = () => {
    if (parsed.records.length === 0) return
    onChange([...records, ...parsed.records])
    setPasteText('')
    setPasteOpen(false)
    setFilter('')
  }

  return (
    <div className="space-y-2">
      {records.length > 0 && (
        <div className="flex items-center gap-2">
          <Switch
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
              onChange={(e) => setFilter(e.target.value)}
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
        const ownerRecords = records.filter((r) => r.name.trim().toLowerCase() === name.trim().toLowerCase())
        const cnameCount = ownerRecords.filter((r) => r.type.toUpperCase() === 'CNAME').length
        const cnameError = cnameCount > 1
          ? 'Only one CNAME target is allowed per name. For multiple addresses, use A/AAAA records.'
          : cnameCount > 0 && ownerRecords.length > 1
            ? 'A CNAME cannot share its name with other records.'
            : null
        return (
          <div key={g.items[0].i} className="space-y-1.5 rounded-md border border-border/60 p-2">
            <div className="flex items-center gap-2">
              <Switch
                className="shrink-0"
                aria-label={`Select records for ${name}`}
                checked={groupSelected(g.items)}
                onCheckedChange={() => toggleGroup(g.items)}
              />
              <Input
                className="h-8 w-40 font-mono text-xs"
                placeholder="@"
                value={name}
                onFocus={() => setEditing(new Set(g.items.map(({ i }) => i)))}
                onBlur={() => setEditing(new Set())}
                onChange={(e) => renameGroup(g.items, e.target.value)}
              />
              <span className="text-xs text-muted-foreground">{g.items.length} record{g.items.length > 1 ? 's' : ''}</span>
              <div className="flex-1" />
              <Button
                variant="ghost"
                size="sm"
                className="h-7 gap-1 text-xs text-muted-foreground"
                onClick={() => { onChange([...records, { name, type: 'A', value: '' }]); setFilter('') }}
              >
                <Plus className="h-3 w-3" /> Add type
              </Button>
            </div>

            {cnameError && <p role="alert" className="pl-6 text-xs text-destructive">{cnameError}</p>}
            {cnameCount === 1 && !cnameError && (
              <p className="pl-6 text-xs text-muted-foreground">A CNAME supports one target and cannot share its name with other records.</p>
            )}

            {g.items.map(({ r, i }) => {
              const rev = reverseInfoFor(r)
              return (
                <div key={i} className="flex items-center gap-2 pl-6">
                  <Switch
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
                  <Input
                    className="h-8 flex-1 font-mono text-xs"
                    placeholder="value"
                    value={r.value}
                    onChange={(e) => set(i, { ...r, value: e.target.value })}
                  />
                  {(r.type === 'MX' || r.type === 'SRV') && (
                    <Input
                      className="h-8 w-20 font-mono text-xs"
                      placeholder="prio"
                      value={r.priority ?? ''}
                      onChange={(e) => set(i, { ...r, priority: e.target.value ? Number(e.target.value) : undefined })}
                    />
                  )}
                  {rev && (
                    <label className="flex shrink-0 items-center gap-1 text-xs text-muted-foreground" title={`Create the reverse (PTR) record in ${rev.zoneName}`}>
                      <Switch
                        className="shrink-0"
                        aria-label="Create reverse record"
                        checked={rev.checked}
                        onCheckedChange={(checked) => toggleReverse(r, checked)}
                      />
                      PTR
                    </label>
                  )}
                  <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => { onChange(records.filter((_, j) => j !== i)); setSelected(new Set()) }}>
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
        <Button variant="outline" size="sm" onClick={() => { onChange([...records, { name: '@', type: 'A', value: '' }]); setFilter('') }}>
          <Plus className="mr-1 h-3 w-3" /> Add record
        </Button>
        <Button variant="outline" size="sm" onClick={() => setPasteOpen((v) => !v)}>
          Paste records
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
              One record per line: <code>name [ttl] type value</code>.
              {parsed.errors.length > 0 && (
                <span className="text-amber-600 dark:text-amber-500"> {parsed.errors.length} line(s) unparsed.</span>
              )}
            </p>
          </div>
        </div>
      )}
    </div>
  )
}
