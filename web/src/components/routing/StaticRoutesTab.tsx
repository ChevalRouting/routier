import { RouteSentence } from '@/components/routing/StaticRoutesParts/RouteSentence'
import { Button, EmptyState } from 'cheval-ui'
import { Plus, RouteOff } from 'lucide-react'
import React from 'react'
import { newId } from './shared'
import { StaticRoute } from './types'

type StaticRoutesTabShape = {
  rows: RouteRow[]
  setRows: React.Dispatch<React.SetStateAction<RouteRow[]>>
  onDirty: () => void
}

export interface RouteRow extends StaticRoute { id: number }

export function StaticRoutesTab({
  rows, setRows, onDirty,
}: StaticRoutesTabShape) {
  const add = () => { setRows((p) => [...p, { id: newId(), destination: '', via: '', dev: '', metric: 0 }]); onDirty() }
  const remove = (id: number) => { setRows((p) => p.filter((r) => r.id !== id)); onDirty() }
  const upd = (id: number, patch: Partial<RouteRow>) => { setRows((p) => p.map((r) => (r.id === id ? { ...r, ...patch } : r))); onDirty() }
  const addButton = (
    <Button variant="outline" size="sm" onClick={add} className="gap-1.5">
      <Plus className="h-4 w-4" />Add Route
    </Button>
  )

  if (rows.length === 0) return <EmptyState icon={<RouteOff />} title="No Static Routes" action={addButton} />

  return (
    <div className="space-y-4">
      <div className="flex justify-end">{addButton}</div>
      <div className="divide-y divide-border rounded-xl bg-card shadow-[var(--card-shadow)]">
        {rows.map((row) => (
          <RouteSentence key={row.id} row={row} onChange={(patch) => upd(row.id, patch)} onRemove={() => remove(row.id)} />
        ))}
      </div>
    </div>
  )
}
