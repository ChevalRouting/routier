import type { TypesFlowStatus as FlowStatus } from '@/api'
import { StatusChip, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'

type FlowCollectorsShape = { status: FlowStatus }

export function FlowCollectors({ status }: FlowCollectorsShape) {
  if (status.collectors.length === 0) return null

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Collector</TableHead>
          <TableHead>Source</TableHead>
          <TableHead className="text-right">Datagrams</TableHead>
          <TableHead className="text-right">Errors</TableHead>
          <TableHead>State</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {status.collectors.map((c) => (
          <TableRow key={c.address}>
            <TableCell className="font-mono text-xs">{c.address}</TableCell>
            <TableCell className="font-mono text-xs">{c.source ?? '-'}</TableCell>
            <TableCell className="text-right tabular-nums">{c.sent.toLocaleString()}</TableCell>
            <TableCell className="text-right tabular-nums">{c.errors.toLocaleString()}</TableCell>
            <TableCell data-wide>
              <div className="space-y-1">
                <StatusChip intent={c.error ? 'danger' : 'success'} label={c.error ? 'Failing' : 'Sending'} />
                {c.error && <p className="text-xs text-muted-foreground break-all">{c.error}</p>}
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
