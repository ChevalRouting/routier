import { emptyNeighbor, NeighborForm, NeighborRow } from '@/components/routing/bgp/Neighbors'
import { BGPNeighbor } from '@/components/routing/types'
import { Button, EmptyState, Label, PreferencesColumns, Sheet } from 'cheval-ui'
import { Plus } from 'lucide-react'
import { useState } from 'react'

type BGPNeighborsPanelShape = {
  neighbors: BGPNeighbor[]
  onChange: (v: BGPNeighbor[]) => void
  prefixListNames: string[]
  routeMapNames: string[]
  bfdProfileNames: string[]
  onDirty: () => void
}

export function BGPNeighborsPanel({
  neighbors, onChange, prefixListNames, routeMapNames, bfdProfileNames, onDirty,
}: BGPNeighborsPanelShape) {
  const [openIdx, setOpenIdx] = useState<number | null>(null)
  const [formDraft, setFormDraft] = useState<BGPNeighbor | null>(null)

  const update = (next: BGPNeighbor[]) => { onChange(next); onDirty() }

  const handleAddNew = () => { setFormDraft(emptyNeighbor()); setOpenIdx(-1) }
  const handleCommitNew = () => {
    if (!formDraft) return
    update([...neighbors, formDraft])
    setOpenIdx(null); setFormDraft(null)
  }
  const handleCloseSheet = () => { setOpenIdx(null); setFormDraft(null) }
  const remove = (idx: number) => {
    update(neighbors.filter((_, i) => i !== idx))
    if (openIdx === idx) handleCloseSheet()
  }

  const isAdding = openIdx === -1
  const openNeighbor = openIdx !== null && openIdx >= 0 ? neighbors[openIdx] : null
  const sheetTitle = isAdding ? 'Add Neighbor' : (openNeighbor?.address || 'Neighbor')

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label className="text-sm font-semibold">Neighbors</Label>
        <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
          <Plus className="h-3.5 w-3.5" />Add Neighbor
        </Button>
      </div>

      {neighbors.length === 0 ? (
        <EmptyState
          title="No neighbors"
          message="Add a BGP neighbor to exchange routes with a peer AS."
          action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add neighbor</Button>}
        />
      ) : (
        <PreferencesColumns>
          {neighbors.map((n, idx) => (
            <NeighborRow
              key={idx}
              neighbor={n}
              onEdit={() => setOpenIdx(idx)}
              onDelete={() => remove(idx)}
            />
          ))}
        </PreferencesColumns>
      )}

      <Sheet open={openIdx !== null} onClose={handleCloseSheet} title={sheetTitle} className="max-w-2xl">
        {openIdx !== null && (isAdding ? formDraft : openNeighbor) && (
          <NeighborForm
            neighbor={isAdding ? formDraft! : openNeighbor!}
            onChange={isAdding ? setFormDraft : (u) => update(neighbors.map((n, i) => i === openIdx ? u : n))}
            prefixListNames={prefixListNames}
            routeMapNames={routeMapNames}
            bfdProfileNames={bfdProfileNames}
            onAdd={isAdding ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
            onDirty={onDirty}
          />
        )}
      </Sheet>
    </div>
  )
}
