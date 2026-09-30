export interface Internet {
  interface: string
  select: string
  mode: 'dhcp' | 'static' | 'disabled'
  ipv6: 'slaac' | 'dhcp6' | 'static' | 'disabled'
  addresses?: string[]
  gateway?: string
  gateway_v6?: string
  dns?: string[]
  vips?: string[]
}

export interface Network {
  id: string
  name: string
  select: string
  addresses: string[]
  manage_dhcp: boolean
  manage_dhcp6?: boolean
  pool?: string
  pool_v6?: string
  gateway?: string
  dns?: string[]
  exclusions?: string[]
  valid_lifetime?: number
  valid_lifetime_v6?: number
  masquerade: boolean
  editable: boolean
  issue?: string
}

export interface PortForward {
  id: string
  name?: string
  proto: string
  port: string
  to_host: string
  to_port?: string
  editable: boolean
  issue?: string
}

export interface SimpleDNS {
  enabled: boolean
  upstreams: string[]
  networks: string[]
  allow_wan: boolean
  wan_allow_from: string[]
  dnssec: boolean
  cache: boolean
  prefetch: boolean
  serve_expired: boolean
  zones: import('@/pages/DnsConfig').DnsZone[]
}

export interface SimpleProjection {
  layer: string
  sections: {
    internet?: Internet
    networks: Network[]
    port_forwards: PortForward[]
    dns: SimpleDNS
    system: { hostname: string }
  }
  issues?: { section: string; message: string }[]
  losses?: { path: string; message: string }[]
}
