import React, {} from 'react'
import { checkIP, checkCIDR } from '@/lib/validate'
import { Input } from 'cheval-ui'
import { NumberInput } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { Plus, Trash2, RouteOff } from 'lucide-react'
import { StaticRoute } from './types'
import { newId } from './shared'

export interface RouteRow extends StaticRoute { id: number }

export function StaticRoutesTab({
  rows, setRows, onDirty,
}: {
  rows: RouteRow[]
  setRows: React.Dispatch<React.SetStateAction<RouteRow[]>>
  onDirty: () => void
}) {
  const add = () => { setRows((p) => [...p, { id: newId(), destination: '', via: '', dev: '', metric: 0 }]); onDirty() }
  const remove = (id: number) => { setRows((p) => p.filter((r) => r.id !== id)); onDirty() }
  const upd = (id: number, patch: Partial<RouteRow>) => { setRows((p) => p.map((r) => (r.id === id ? { ...r, ...patch } : r))); onDirty() }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Static routes install fixed next-hops directly into the kernel routing table, independent of any
        dynamic routing protocol.
      </p>
      <div className="flex justify-end">
        <Button variant="outline" size="sm" onClick={add} className="gap-1.5">
          <Plus className="h-4 w-4" />Add Route
        </Button>
      </div>
      {rows.length === 0 ? (
        <EmptyState
          icon={<RouteOff />}
          title="No static routes"
          message="Install fixed next-hops into the kernel routing table, independent of dynamic protocols."
          action={<Button variant="outline" size="sm" onClick={add} className="gap-2"><Plus className="h-4 w-4" />Add route</Button>}
        />
      ) : (
        <div className="rounded-xl bg-card shadow-[var(--card-shadow)] overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-muted/50">
              <tr>
                <th className="text-left px-3 py-2 text-xs font-medium text-muted-foreground">Destination</th>
                <th className="text-left px-3 py-2 text-xs font-medium text-muted-foreground">Via</th>
                <th className="text-left px-3 py-2 text-xs font-medium text-muted-foreground">Dev</th>
                <th className="text-left px-3 py-2 text-xs font-medium text-muted-foreground w-24">Metric</th>
                <th className="w-10" />
              </tr>
            </thead>
            <tbody className="divide-y">
              {rows.map((row) => (
                <tr key={row.id}>
                  <td className="px-3 py-1.5">
                    <Input value={row.destination} onChange={(e) => upd(row.id, { destination: e.target.value })} placeholder="10.0.0.0/8" aria-invalid={row.destination && checkCIDR(row.destination) ? true : undefined} className={`font-mono text-xs h-8 border-0 shadow-none focus-visible:ring-1 ${row.destination && checkCIDR(row.destination) ? 'ring-1 ring-destructive' : ''}`} />
                  </td>
                  <td className="px-3 py-1.5">
                    <Input value={row.via} onChange={(e) => upd(row.id, { via: e.target.value })} placeholder="192.168.1.1" aria-invalid={row.via && checkIP(row.via) ? true : undefined} className={`font-mono text-xs h-8 border-0 shadow-none focus-visible:ring-1 ${row.via && checkIP(row.via) ? 'ring-1 ring-destructive' : ''}`} />
                  </td>
                  <td className="px-3 py-1.5">
                    <Input value={row.dev} onChange={(e) => upd(row.id, { dev: e.target.value })} placeholder="eth0 or wan" className="font-mono text-xs h-8 border-0 shadow-none focus-visible:ring-1" />
                  </td>
                  <td className="px-3 py-1.5">
                    <NumberInput value={row.metric || undefined} onChange={(v) => upd(row.id, { metric: v ?? 0 })} className="font-mono text-xs h-8 border-0 shadow-none focus-visible:ring-1 w-20" />
                  </td>
                  <td className="px-2">
                    <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => remove(row.id)}>
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
