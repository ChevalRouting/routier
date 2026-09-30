import { Link } from 'react-router-dom'
import { ArrowRightLeft, Globe, Network, Search, Server, Activity } from 'lucide-react'
import { Badge, Button, Card, CardContent, CardHeader, CardTitle, PageHeader, Spinner } from 'cheval-ui'
import { api, configLayerRequest } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { SystemOverview } from '@/components/SystemOverview'
import { NeighborsPanel } from '@/components/monitor/NeighborsPanel'
import { ProbePanel } from '@/components/ProbePanel'
import type { SimpleProjection } from '@/components/simple/types'
import type { TypesSystemNic as SystemNic } from '@/api'
import type { KeaSubnetView } from '@/api'

function matchingNic(nics: SystemNic[], selector: string): SystemNic | undefined {
  const mac = selector.match(/^mac\((.+)\)$/)?.[1]
  const name = selector.startsWith('name=') ? selector.slice(5) : selector
  return nics.find((nic) => mac ? nic.mac?.toLowerCase() === mac.toLowerCase() : nic.name === name)
}

function EditLink({ to }: { to: string }) {
  return <Button asChild variant="ghost" size="sm"><Link to={to}>Configure</Link></Button>
}

export default function SimpleConfig() {
  const { data } = useFetch<SimpleProjection>(() => configLayerRequest('/api/config'))
  const { data: nics } = useFetch<SystemNic[]>(() => api.apiSystemNicsGet())
  const { data: dhcpSubnets } = useFetch<KeaSubnetView[]>(() => api.apiDhcpGet() as Promise<KeaSubnetView[]>)
  if (!data) return <Spinner />

  const internet = data.sections.internet
  const wanNic = internet ? matchingNic(nics ?? [], internet.select) : undefined
  const wanAddresses = wanNic?.addrs?.length ? wanNic.addrs : internet?.addresses ?? []

  return (
    <div className="space-y-6">
      <PageHeader title="Overview" description="The essential configuration for this router." />

      <SystemOverview />

      <ProbePanel />

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <div className="flex items-center gap-2"><Globe className="h-4 w-4 text-muted-foreground" /><CardTitle className="text-base">Internet</CardTitle></div>
            <EditLink to="/simple/internet" />
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div><p className="text-xs text-muted-foreground">WAN interface</p><p className="mt-1 font-mono text-sm font-medium">{wanNic?.name || internet?.select || 'Not configured'}</p></div>
              <div><p className="text-xs text-muted-foreground">Connection</p><p className="mt-1 text-sm font-medium">{internet?.mode === 'dhcp' ? 'Automatic (DHCP)' : internet?.mode || 'Not configured'}</p></div>
              <div className="sm:col-span-2"><p className="text-xs text-muted-foreground">WAN / public addresses</p><div className="mt-1.5 flex flex-wrap gap-1.5">{wanAddresses.length ? wanAddresses.map((address) => <Badge key={address} variant="outline" className="font-mono">{address}</Badge>) : <span className="text-sm text-muted-foreground">No address assigned</span>}</div></div>
              {!!internet?.vips?.length && <div className="sm:col-span-2"><p className="text-xs text-muted-foreground">Failover public addresses</p><div className="mt-1.5 flex flex-wrap gap-1.5">{internet.vips.map((address) => <Badge key={address} variant="secondary" className="font-mono">{address}</Badge>)}</div></div>}
              {internet?.gateway && <div><p className="text-xs text-muted-foreground">Gateway</p><p className="mt-1 font-mono text-sm">{internet.gateway}</p></div>}
              {!!internet?.dns?.length && <div><p className="text-xs text-muted-foreground">DNS</p><p className="mt-1 font-mono text-sm">{internet.dns.join(', ')}</p></div>}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <div className="flex items-center gap-2"><Server className="h-4 w-4 text-muted-foreground" /><CardTitle className="text-base">Router</CardTitle></div>
            <EditLink to="/simple/system" />
          </CardHeader>
          <CardContent><p className="text-xs text-muted-foreground">Hostname</p><p className="mt-1 font-mono text-lg font-semibold">{data.sections.system.hostname || 'Not named'}</p></CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <div className="flex items-center gap-2"><Search className="h-4 w-4 text-muted-foreground" /><CardTitle className="text-base">DNS</CardTitle></div>
          <EditLink to="/simple/dns" />
        </CardHeader>
        <CardContent className="space-y-2">
          <div className="flex items-center gap-2"><Badge variant={data.sections.dns?.enabled ? 'secondary' : 'outline'}>{data.sections.dns?.enabled ? 'Running' : 'Disabled'}</Badge>{data.sections.dns?.dnssec && <Badge variant="outline">DNSSEC</Badge>}</div>
          {data.sections.dns?.enabled && <p className="text-sm text-muted-foreground">Available on {data.sections.dns.networks.length} local network{data.sections.dns.networks.length === 1 ? '' : 's'} via {data.sections.dns.upstreams.join(', ') || 'no upstream'}</p>}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <div className="flex items-center gap-2"><Network className="h-4 w-4 text-muted-foreground" /><CardTitle className="text-base">Local networks</CardTitle></div>
          <EditLink to="/simple/networks" />
        </CardHeader>
        <CardContent>
          {data.sections.networks.length === 0 ? <p className="text-sm text-muted-foreground">No local networks configured.</p> : (
            <div className="grid gap-3 md:grid-cols-2">
              {data.sections.networks.map((network) => (
                <div key={network.id || network.name} className="rounded-lg bg-muted/30 p-3">
                  <div className="flex items-center justify-between gap-2"><p className="font-medium">{network.name || 'Unnamed network'}</p><div className="flex gap-1">{network.manage_dhcp && <Badge variant="secondary">DHCPv4</Badge>}{network.manage_dhcp6 && <Badge variant="secondary">DHCPv6</Badge>}</div></div>
                  <div className="mt-2 flex flex-wrap gap-1.5">{network.addresses.length ? network.addresses.map((address) => <Badge key={address} variant="outline" className="font-mono">{address}</Badge>) : <span className="text-xs text-muted-foreground">No router address</span>}</div>
                  {network.pool && <p className="mt-2 text-xs text-muted-foreground">Pool <span className="font-mono text-foreground">{network.pool}</span></p>}
                  {(network.manage_dhcp || network.manage_dhcp6) && <p className="mt-2 text-xs text-muted-foreground">Active leases <span className="font-medium text-foreground">{(dhcpSubnets ?? []).filter((subnet) => network.addresses.some((address) => networkPrefix(subnet.subnet) === networkPrefix(address))).reduce((total, subnet) => total + (subnet.leases?.length ?? 0), 0)}</span></p>}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <div className="flex items-center gap-2"><ArrowRightLeft className="h-4 w-4 text-muted-foreground" /><CardTitle className="text-base">Port forwards</CardTitle></div>
          <EditLink to="/simple/port-forwards" />
        </CardHeader>
        <CardContent><p className="text-2xl font-semibold">{data.sections.port_forwards.length}</p><p className="text-xs text-muted-foreground">Configured inbound forwards</p></CardContent>
      </Card>

      <Card>
        <CardHeader><div className="flex items-center gap-2"><Activity className="h-4 w-4 text-muted-foreground" /><CardTitle className="text-base">Monitoring · Neighbors</CardTitle></div></CardHeader>
        <CardContent><NeighborsPanel /></CardContent>
      </Card>
    </div>
  )
}

function networkPrefix(address: string): string {
  const [ip, bits] = address.split('/')
  if (!ip || !bits) return address
  if (ip.includes(':')) {
    const value = ipv6Value(ip)
    const width = Number(bits)
    if (value === null || width < 0 || width > 128) return address.toLowerCase()
    const mask = width === 0 ? 0n : ((1n << BigInt(width)) - 1n) << BigInt(128 - width)
    return `${(value & mask).toString(16).padStart(32, '0')}/${width}`
  }
  const value = ip.split('.').reduce((out, part) => (out << 8) | Number(part), 0) >>> 0
  const width = Number(bits)
  const mask = width === 0 ? 0 : (0xffffffff << (32 - width)) >>> 0
  const network = (value & mask) >>> 0
  return `${[24, 16, 8, 0].map((shift) => (network >>> shift) & 255).join('.')}/${width}`
}

function ipv6Value(address: string): bigint | null {
  const halves = address.toLowerCase().split('::')
  if (halves.length > 2) return null
  const left = halves[0] ? halves[0].split(':') : []
  const right = halves.length === 2 && halves[1] ? halves[1].split(':') : []
  const missing = 8 - left.length - right.length
  if (missing < 0 || (halves.length === 1 && missing !== 0)) return null
  const groups = [...left, ...Array(missing).fill('0'), ...right]
  if (groups.length !== 8 || groups.some((group) => !/^[0-9a-f]{1,4}$/.test(group))) return null
  return groups.reduce((value, group) => (value << 16n) | BigInt(`0x${group}`), 0n)
}
