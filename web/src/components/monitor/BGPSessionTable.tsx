import type { BGPSession } from '@/components/routing/overview/bgpPeers'
import { onActivate } from '@/lib/utils'
import { StateChip, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'

type BGPSessionTableShape = {
  sessions: BGPSession[]
  showInterface: boolean
  onSelect: (address: string) => void
}

function stateOrder(session: BGPSession): number {
  return session.status?.state === 'Established' ? 0 : 1
}

function compareSessions(a: BGPSession, b: BGPSession): number {
  const d = stateOrder(a) - stateOrder(b)
  return d !== 0 ? d : a.neighbor.remote_asn - b.neighbor.remote_asn
}

export function BGPSessionTable({ sessions, showInterface, onSelect }: BGPSessionTableShape) {
  return (
    <div className="rounded-xl bg-card shadow-[var(--card-shadow)] overflow-x-auto">
      <Table className="w-full text-sm md:table-fixed">
        <TableHeader>
          <TableRow className="border-b border-border text-xs text-muted-foreground">
            <TableHead className="text-left px-4 py-2.5 md:w-[30%]">Neighbor</TableHead>
            {showInterface && <TableHead className="text-left px-4 py-2.5 md:w-[14%]">Interface</TableHead>}
            <TableHead className="text-left px-4 py-2.5 md:w-[14%]">Remote AS</TableHead>
            <TableHead className="text-left px-4 py-2.5 md:w-[18%]">State</TableHead>
            <TableHead className="text-left px-4 py-2.5 md:w-[14%]">Uptime</TableHead>
            <TableHead className="text-right px-4 py-2.5">Prefixes</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {[...sessions].sort(compareSessions).map(({ neighbor, iface, status }) => (
            <TableRow key={neighbor.address} role="button" tabIndex={0} onClick={() => onSelect(neighbor.address)} onKeyDown={onActivate(() => onSelect(neighbor.address))} className="cursor-pointer">
              <TableCell data-title className="px-4 py-2.5">
                <div className="font-mono text-xs break-all">{neighbor.address}</div>
                {neighbor.description && <div className="text-xs text-muted-foreground mt-0.5">{neighbor.description}</div>}
              </TableCell>
              {showInterface && <TableCell className="px-4 py-2.5 font-mono text-xs text-muted-foreground">{iface ?? '-'}</TableCell>}
              <TableCell className="px-4 py-2.5 font-mono text-xs">AS{neighbor.remote_asn}</TableCell>
              <TableCell className="px-4 py-2.5">{status ? <StateChip state={status.state} /> : '-'}</TableCell>
              <TableCell className="px-4 py-2.5 font-mono text-xs text-muted-foreground">{status?.uptime || '-'}</TableCell>
              <TableCell className="px-4 py-2.5 text-right font-mono text-xs">{status?.prefixes ?? '-'}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
