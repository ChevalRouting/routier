import { RouteMapEntryCard, RouteMapRow } from '@/components/routing/bgp/RouteMaps'
import { newId } from '@/components/routing/shared'
import { Button } from 'cheval-ui'
import { Plus } from 'lucide-react'

type RouteMapFormShape = {
  rows: RouteMapRow[]
  onChange: (v: RouteMapRow[]) => void
  prefixListNames: string[]
  callTargets: string[]
  onDone: () => void
}

export function RouteMapForm({
  rows, onChange, prefixListNames, callTargets, onDone,
}: RouteMapFormShape) {
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
