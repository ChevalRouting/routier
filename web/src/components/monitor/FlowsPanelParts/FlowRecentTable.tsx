import type { TypesFlow as Flow } from '@/api'
import { EmptyState, fmtBytes, Input, Pagination, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, usePagination } from 'cheval-ui'
import { Search } from 'lucide-react'
import { endpoint, protocolName } from './flowFormat'

type FlowRecentTableShape = {
  flows: Flow[]
  address: string
  onAddressChange: (address: string) => void
  ifaceLabel: (device: string) => string
}

export function FlowRecentTable({ flows, address, onAddressChange, ifaceLabel }: FlowRecentTableShape) {
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(flows, 25)

  return (
    <div className="space-y-3">
      <div className="relative max-w-xs">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50" />
        <Input placeholder="Filter by Address…" value={address} onChange={(e) => onAddressChange(e.target.value.trim())} className="pl-8 h-8 text-xs font-mono" />
      </div>
      {flows.length === 0 ? (
        <EmptyState className="py-8" title={address ? 'No Results' : 'No Flows'} />
      ) : (
        <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Source</TableHead>
                <TableHead>Destination</TableHead>
                <TableHead>Protocol</TableHead>
                <TableHead>Interfaces</TableHead>
                <TableHead className="text-right">Bytes</TableHead>
                <TableHead className="text-right">Packets</TableHead>
                <TableHead>Last Seen</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pageItems.map((f) => (
                <TableRow key={f.id}>
                  <TableCell className="font-mono text-xs break-all">{endpoint(f.src, f.src_port, f.proto)}</TableCell>
                  <TableCell className="font-mono text-xs break-all">{endpoint(f.dst, f.dst_port, f.proto)}</TableCell>
                  <TableCell className="text-xs">{protocolName(f.proto)}</TableCell>
                  <TableCell className="font-mono text-xs">{ifaceLabel(f.in_if)} → {f.out_if ? ifaceLabel(f.out_if) : 'local'}</TableCell>
                  <TableCell className="text-right tabular-nums">{fmtBytes(f.bytes)}</TableCell>
                  <TableCell className="text-right tabular-nums">{f.packets.toLocaleString()}</TableCell>
                  <TableCell className="font-mono text-xs">{new Date(f.end * 1000).toLocaleTimeString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="flows" />
        </>
      )}
    </div>
  )
}
