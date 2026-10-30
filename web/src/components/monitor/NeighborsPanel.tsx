import type { TypesNeighbor as Neighbor } from '@/api'
import { EmptyState, Pagination, SectionLabel, StateChip, usePagination, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'

type NeighborTableShape = { label: string; rows: Neighbor[] }

export function NeighborTable({ label, rows }: NeighborTableShape) {
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(rows, 25)
  return (
    <div className="space-y-2">
      <SectionLabel>{label}</SectionLabel>
      {rows.length === 0
        ? <EmptyState className="py-8" title="No Neighbors" />
        : (
          <>
            <div className="rounded-xl bg-card shadow-[var(--card-shadow)] overflow-x-auto">
              <Table className="w-full text-sm">
                <TableHeader>
                  <TableRow className="border-b border-border/60">
                    {['Address', 'Interface', 'MAC / Link-layer', 'State'].map((h) => (
                      <TableHead key={h} className="px-4 py-2.5 text-left text-xs font-semibold text-muted-foreground">{h}</TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pageItems.map((n, i) => (
                    <TableRow key={`${n.dst}-${n.dev}-${i}`} className="border-b border-border/40 last:border-0 hover:bg-accent/20">
                      <TableCell className="px-4 py-2 font-mono text-xs">{n.dst}</TableCell>
                      <TableCell className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.dev}</TableCell>
                      <TableCell className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.lladdr ?? '-'}</TableCell>
                      <TableCell className="px-4 py-2"><StateChip state={n.state} /></TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="entries" />
          </>
        )}
    </div>
  )
}

export { LLDPTable } from './NeighborsPanelParts/LLDPTable'
export { NeighborsPanel } from './NeighborsPanelParts/NeighborsPanel'
