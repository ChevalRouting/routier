import React, {} from 'react'
import { Input } from '@/components/ui/input'
import { NumberInput } from '@/components/ui/number-input'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/EmptyState'
import { Plus, Trash2 } from 'lucide-react'
import { newId } from './shared'

export interface VRFRow { _id: number; name: string; table: number }

export function VRFsTab({
  rows, setRows, onDirty,
}: {
  rows: VRFRow[]
  setRows: React.Dispatch<React.SetStateAction<VRFRow[]>>
  onDirty: () => void
}) {
  const add = () => { setRows((p) => [...p, { _id: newId(), name: '', table: 0 }]); onDirty() }
  const remove = (id: number) => { setRows((p) => p.filter((r) => r._id !== id)); onDirty() }
  const upd = (id: number, patch: Partial<VRFRow>) => { setRows((p) => p.map((r) => (r._id === id ? { ...r, ...patch } : r))); onDirty() }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        VRFs (Virtual Routing and Forwarding) keep separate routing tables on one device, isolating route
        domains so overlapping networks can coexist.
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
        <div className="rounded-xl border border-border overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-muted/50">
              <tr>
                <th className="text-left px-3 py-2 text-xs font-medium text-muted-foreground">Name</th>
                <th className="text-left px-3 py-2 text-xs font-medium text-muted-foreground w-36">Table ID</th>
                <th className="w-10" />
              </tr>
            </thead>
            <tbody className="divide-y">
              {rows.map((row) => (
                <tr key={row._id}>
                  <td className="px-3 py-1.5">
                    <Input
                      value={row.name}
                      onChange={(e) => upd(row._id, { name: e.target.value })}
                      placeholder="red"
                      className="font-mono text-xs h-8 border-0 shadow-none focus-visible:ring-1"
                    />
                  </td>
                  <td className="px-3 py-1.5">
                    <NumberInput
                      value={row.table || undefined}
                      onChange={(v) => upd(row._id, { table: v ?? 0 })}
                      placeholder="100"
                      className="font-mono text-xs h-8 border-0 shadow-none focus-visible:ring-1 w-28"
                    />
                  </td>
                  <td className="px-2">
                    <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => remove(row._id)}>
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

