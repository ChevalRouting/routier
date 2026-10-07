import { Button, Card, EmptyState, Input, Label, NumberInput } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import React from 'react'
import { newId } from './shared'

type VRFsTabShape = {
  rows: VRFRow[]
  setRows: React.Dispatch<React.SetStateAction<VRFRow[]>>
  onDirty: () => void
}

export interface VRFRow { _id: number; name: string; table: number }

const TABLE_OFFSET = 40000

function nextTable(rows: VRFRow[]): number {
  let max = TABLE_OFFSET - 1
  for (const r of rows) {
    if (r.table > max) max = r.table
  }
  return max + 1
}

export function VRFsTab({
  rows, setRows, onDirty,
}: VRFsTabShape) {
  const add = () => { setRows((p) => [...p, { _id: newId(), name: '', table: nextTable(p) }]); onDirty() }
  const remove = (id: number) => { setRows((p) => p.filter((r) => r._id !== id)); onDirty() }
  const upd = (id: number, patch: Partial<VRFRow>) => { setRows((p) => p.map((r) => (r._id === id ? { ...r, ...patch } : r))); onDirty() }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        VRFs (Virtual Routing and Forwarding) keep separate routing tables on one device, isolating route
        domains so overlapping networks can coexist. New VRFs get the next free table ID from 40000; edit
        it if you need a specific one.
      </p>
      <div className="flex justify-end">
        <Button variant="outline" size="sm" onClick={add} className="gap-1.5">
          <Plus className="h-4 w-4" />Add VRF
        </Button>
      </div>
      {rows.length === 0 ? (
        <EmptyState
          title="No VRFs"
          message="Declare a VRF to isolate interfaces and routing tables."
          action={<Button variant="outline" size="sm" onClick={add} className="gap-2"><Plus className="h-4 w-4" />Add VRF</Button>}
        />
      ) : (
        <div className="grid gap-3 md:grid-cols-2">
          {rows.map((row) => (
            <Card key={row._id} className="space-y-4 p-4">
              <div className="flex items-center justify-between gap-3">
                <span className="font-mono text-sm font-semibold">{row.name || 'New VRF'}</span>
                <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => remove(row._id)}>
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-1.5">
                  <Label className="text-xs text-muted-foreground">Name</Label>
                    <Input
                      value={row.name}
                      onChange={(e) => upd(row._id, { name: e.target.value })}
                      placeholder="red"
                      className="font-mono text-sm"
                    />
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs text-muted-foreground">Table ID</Label>
                    <NumberInput
                      value={row.table || undefined}
                      onChange={(v) => upd(row._id, { table: v ?? 0 })}
                      placeholder="40000"
                      className="font-mono text-sm"
                    />
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
