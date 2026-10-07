import type { TypesNeighbor as Neighbor } from '@/api'
import { EmptyState, Pagination, SectionLabel, StateChip, usePagination } from 'cheval-ui'

type NeighborTableShape = { label: string; rows: Neighbor[] }

export function NeighborTable({ label, rows }: NeighborTableShape) {
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(rows, 25)
  return (
    <div className="space-y-2">
      <SectionLabel>{label}</SectionLabel>
      {rows.length === 0
        ? <EmptyState className="py-8" title="No neighbors" message="Discovered L2/L3 neighbors will show up here." />
        : (
          <>
            <div className="rounded-xl bg-card shadow-[var(--card-shadow)] overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border/60">
                    {['Address', 'Interface', 'MAC / Link-layer', 'State'].map((h) => (
                      <th key={h} className="px-4 py-2.5 text-left text-xs font-semibold text-muted-foreground">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((n, i) => (
                    <tr key={`${n.dst}-${n.dev}-${i}`} className="border-b border-border/40 last:border-0 hover:bg-accent/20">
                      <td className="px-4 py-2 font-mono text-xs">{n.dst}</td>
                      <td className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.dev}</td>
                      <td className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.lladdr ?? '-'}</td>
                      <td className="px-4 py-2"><StateChip state={n.state} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="entries" />
          </>
        )}
    </div>
  )
}

export { LLDPTable } from './NeighborsPanelParts/LLDPTable'
export { NeighborsPanel } from './NeighborsPanelParts/NeighborsPanel'
