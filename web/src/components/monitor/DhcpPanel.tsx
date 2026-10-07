import { Card, CardContent, CardHeader, CardTitle, SectionNav, useTabState } from 'cheval-ui'
import { DhcpLogsPanel } from './DhcpPanelParts/DhcpLogsPanel'
import { DhcpStatusPanel } from './DhcpPanelParts/DhcpStatusPanel'

type StatsShape2 = { name: string; value: number }

type DhcpStatGridShape = { title: string; stats: StatsShape[] | null }

type StatsShape = { name: string; value: number }

type DHCPLOGTABSShape = { key: DhcpLogSource; label: string }

export const DHCP_KEY_STATS = [
  'declined-addresses',
  'pkt4-received', 'pkt4-ack-sent', 'pkt4-offer-sent',
  'pkt6-received', 'pkt6-reply-sent',
  'cumulative-assigned-addresses', 'cumulative-assigned-nas',
]

export function statValue(stats: StatsShape2[] | null | undefined, name: string): number | undefined {
  return stats?.find((s) => s.name === name)?.value
}

export function DhcpStatGrid({ title, stats }: DhcpStatGridShape) {
  if (!stats || stats.length === 0) {
    return (
      <Card>
        <CardHeader className="pb-2"><CardTitle className="text-sm">{title}</CardTitle></CardHeader>
        <CardContent><p className="text-xs text-muted-foreground italic">No statistics (server not running).</p></CardContent>
      </Card>
    )
  }
  const shown = DHCP_KEY_STATS.map((n) => ({ n, v: statValue(stats, n) })).filter((x) => x.v !== undefined)
  return (
    <Card>
      <CardHeader className="pb-2"><CardTitle className="text-sm">{title}</CardTitle></CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          {shown.map(({ n, v }) => (
            <div key={n} className="space-y-0.5">
              <div className="text-lg font-semibold tabular-nums">{v}</div>
              <div className="text-[11px] text-muted-foreground font-mono truncate" title={n}>{n}</div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

export type LeaseFamily = 'all' | 'dhcp4' | 'dhcp6'

export const NO_ZONE = '__none__'

export interface DnsZoneOption {
  name: string
  primaries?: string[]
}

export interface DnsServerSection {
  zones?: DnsZoneOption[]
}

export function forwardZoneNames(section: DnsServerSection | null): string[] {
  return (section?.zones ?? [])
    .filter((z) => (z.primaries ?? []).length === 0)
    .map((z) => z.name)
    .filter((n) => n && !n.endsWith('in-addr.arpa') && !n.endsWith('ip6.arpa'))
}

export function subnetService(cidr: string): string {
  return cidr.includes(':') ? 'dhcp6' : 'dhcp4'
}

export type DhcpLogSource = 'kea-dhcp4' | 'kea-dhcp6' | 'kea-dhcp-ddns'
export const DHCP_LOG_TABS: DHCPLOGTABSShape[] = [
  { key: 'kea-dhcp4', label: 'kea-dhcp4' },
  { key: 'kea-dhcp6', label: 'kea-dhcp6' },
  { key: 'kea-dhcp-ddns', label: 'kea-dhcp-ddns' },
]

export function DhcpPanel() {
  const [sub, setSub] = useTabState<'status' | 'logs'>('dhcp', 'status')
  return (
    <SectionNav
      items={[{ key: 'status', label: 'Status & Leases' }, { key: 'logs', label: 'Logs' }]}
      active={sub}
      onChange={setSub}
    >
      {sub === 'status' && <DhcpStatusPanel />}
      {sub === 'logs' && <DhcpLogsPanel />}
    </SectionNav>
  )
}

export type MonitorTab = 'system' | 'bgp' | 'traffic' | 'logs' | 'ha-status' | 'neighbors' | 'processes' | 'collection' | 'dhcp'

export { DhcpLeases } from './DhcpPanelParts/DhcpLeases'
export { DhcpLogsPanel } from './DhcpPanelParts/DhcpLogsPanel'
export { DhcpStatusPanel } from './DhcpPanelParts/DhcpStatusPanel'
