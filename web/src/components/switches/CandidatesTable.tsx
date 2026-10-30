import { Badge, Button, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'
import type { TypesSwitchCandidate as SwitchCandidate } from '@/api'

type CandidatesTableShape = { candidates: SwitchCandidate[]; onManage: (candidate: SwitchCandidate) => void }

export function CandidatesTable({ candidates, onManage }: CandidatesTableShape) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Device</TableHead>
          <TableHead>Management</TableHead>
          <TableHead>Seen On</TableHead>
          <TableHead />
        </TableRow>
      </TableHeader>
      <TableBody>
        {candidates.map((candidate) => (
          <TableRow key={candidate.chassis_id}>
            <TableCell>
              <div className="font-medium">{candidate.name || candidate.chassis_id}</div>
              <div className="max-w-md truncate text-xs text-muted-foreground" title={candidate.sys_descr}>{candidate.sys_descr || '-'}</div>
            </TableCell>
            <TableCell className="font-mono text-xs">{candidate.mgmt_ip || '-'}</TableCell>
            <TableCell className="text-xs">
              <span className="font-mono">{candidate.local_iface}</span> to <span className="font-mono">{candidate.port_id || '?'}</span>
              <div className="mt-1 flex gap-1">{candidate.protocols.map((protocol) => <Badge key={protocol} variant="neutral">{protocol}</Badge>)}</div>
            </TableCell>
            <TableCell className="text-right">
              <Button size="sm" variant="outline" title={candidate.model_id ? undefined : 'No Model Recognized, Pick One by Hand'} onClick={() => onManage(candidate)}>
                Manage
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
