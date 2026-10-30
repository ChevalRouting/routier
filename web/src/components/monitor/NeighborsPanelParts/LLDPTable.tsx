import type { TypesLLDPNeighbor as LLDPNeighbor } from '@/api'
import { EmptyState, Pagination, SectionLabel, StateChip, usePagination, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'

type LLDPTableShape = { label: string; rows: LLDPNeighbor[] }

export function LLDPTable({ label, rows }: LLDPTableShape) {
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(rows, 25)
  return (
    <div className="space-y-2">
      <SectionLabel>{label}</SectionLabel>
      {rows.length === 0
        ? <EmptyState className="py-8" title="No Link-Layer Neighbors" message="Enable LLDP under General › Monitoring" />
        : (
          <>
            <div className="rounded-xl bg-card shadow-[var(--card-shadow)] overflow-x-auto">
              <Table className="w-full text-sm">
                <TableHeader>
                  <TableRow className="border-b border-border/60">
                    {['Interface', 'Protocol', 'Chassis', 'Remote port', 'Mgmt IP', 'VLAN'].map((h) => (
                      <TableHead key={h} className="px-4 py-2.5 text-left text-xs font-semibold text-muted-foreground">{h}</TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pageItems.map((n, i) => (
                    <TableRow key={`${n.local_iface}-${n.chassis_id}-${i}`} className="border-b border-border/40 last:border-0 hover:bg-accent/20">
                      <TableCell className="px-4 py-2 font-mono text-xs">{n.local_iface}</TableCell>
                      <TableCell className="px-4 py-2"><StateChip state={n.protocol} /></TableCell>
                      <TableCell className="px-4 py-2 text-xs">
                        <div className="font-medium">{n.chassis_name || n.chassis_id || '-'}</div>
                        {n.sys_descr && <div className="text-muted-foreground truncate max-w-xs">{n.sys_descr}</div>}
                      </TableCell>
                      <TableCell className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.port_id || n.port_descr || '-'}</TableCell>
                      <TableCell className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.mgmt_ip || '-'}</TableCell>
                      <TableCell className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.vlan || '-'}</TableCell>
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
