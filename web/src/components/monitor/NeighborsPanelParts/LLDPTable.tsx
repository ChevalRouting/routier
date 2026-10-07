import type { TypesLLDPNeighbor as LLDPNeighbor } from '@/api'
import { EmptyState, Pagination, SectionLabel, StateChip, usePagination } from 'cheval-ui'

type LLDPTableShape = { label: string; rows: LLDPNeighbor[] }

export function LLDPTable({ label, rows }: LLDPTableShape) {
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(rows, 25)
  return (
    <div className="space-y-2">
      <SectionLabel>{label}</SectionLabel>
      {rows.length === 0
        ? <EmptyState className="py-8" title="No link-layer neighbors" message="Enable LLDP/CDP under General > Monitoring to discover directly connected switches and routers." />
        : (
          <>
            <div className="rounded-xl bg-card shadow-[var(--card-shadow)] overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border/60">
                    {['Interface', 'Protocol', 'Chassis', 'Remote port', 'Mgmt IP', 'VLAN'].map((h) => (
                      <th key={h} className="px-4 py-2.5 text-left text-xs font-semibold text-muted-foreground">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((n, i) => (
                    <tr key={`${n.local_iface}-${n.chassis_id}-${i}`} className="border-b border-border/40 last:border-0 hover:bg-accent/20">
                      <td className="px-4 py-2 font-mono text-xs">{n.local_iface}</td>
                      <td className="px-4 py-2"><StateChip state={n.protocol} /></td>
                      <td className="px-4 py-2 text-xs">
                        <div className="font-medium">{n.chassis_name || n.chassis_id || '-'}</div>
                        {n.sys_descr && <div className="text-muted-foreground truncate max-w-xs">{n.sys_descr}</div>}
                      </td>
                      <td className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.port_id || n.port_descr || '-'}</td>
                      <td className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.mgmt_ip || '-'}</td>
                      <td className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.vlan || '-'}</td>
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
