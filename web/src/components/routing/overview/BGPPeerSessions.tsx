import { StateChip, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'
import { BGPPeerGroup } from './bgpPeers'

type BGPPeerSessionsShape = { peer: BGPPeerGroup }

export function BGPPeerSessions({ peer }: BGPPeerSessionsShape) {
  return (
    <div className="space-y-1.5">
      <div className="text-xs text-muted-foreground font-medium">Sessions ({peer.sessions.length})</div>
      <div className="max-h-96 overflow-auto rounded-xl bg-card shadow-[var(--card-shadow)]">
        <Table className="w-full text-xs">
          <TableHeader className="sticky top-0 z-10 bg-card">
            <TableRow className="border-b border-border text-muted-foreground">
              <TableHead className="px-3 py-2 text-left font-medium">Neighbor</TableHead>
              <TableHead className="px-3 py-2 text-left font-medium">Interface</TableHead>
              <TableHead className="px-3 py-2 text-left font-medium">Remote AS</TableHead>
              <TableHead className="px-3 py-2 text-left font-medium">State</TableHead>
              <TableHead className="px-3 py-2 text-left font-medium">Uptime</TableHead>
              <TableHead className="px-3 py-2 text-right font-medium">Prefixes</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {peer.sessions.map(({ neighbor, iface, status }) => (
              <TableRow key={neighbor.address} className="border-b border-border/40 last:border-0">
                <TableCell data-title className="px-3 py-1.5">
                  <div className="font-mono">{neighbor.address}</div>
                  {neighbor.description && <div className="text-muted-foreground">{neighbor.description}</div>}
                </TableCell>
                <TableCell className="px-3 py-1.5 font-mono text-muted-foreground">{iface ?? '-'}</TableCell>
                <TableCell className="px-3 py-1.5 font-mono">AS{neighbor.remote_asn}</TableCell>
                <TableCell className="px-3 py-1.5">{status ? <StateChip state={status.state} /> : '-'}</TableCell>
                <TableCell className="px-3 py-1.5 font-mono text-muted-foreground">{status?.uptime || '-'}</TableCell>
                <TableCell className="px-3 py-1.5 text-right font-mono">{status?.prefixes ?? '-'}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
