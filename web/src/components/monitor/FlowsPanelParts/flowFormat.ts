const PROTOCOLS: Record<number, string> = {
  1: 'ICMP',
  2: 'IGMP',
  6: 'TCP',
  17: 'UDP',
  41: 'IPv6',
  47: 'GRE',
  50: 'ESP',
  51: 'AH',
  58: 'ICMPv6',
  89: 'OSPF',
  112: 'VRRP',
  132: 'SCTP',
}

const PORTED = new Set([6, 17, 132])

export function protocolName(proto: number): string {
  return PROTOCOLS[proto] ?? String(proto)
}

export function serviceLabel(key: string): string {
  const [proto, port] = key.split('/').map(Number)
  return PORTED.has(proto) ? `${protocolName(proto)} ${port}` : protocolName(proto)
}

export function endpoint(address: string, port: number, proto: number): string {
  if (!PORTED.has(proto)) return address
  return address.includes(':') ? `[${address}]:${port}` : `${address}:${port}`
}

export function topKeyLabel(dimension: string, key: string): string {
  if (dimension === 'service') return serviceLabel(key)
  if (dimension === 'protocol') return protocolName(Number(key))
  if (dimension === 'conversation') return key.replace(' > ', ' → ')
  return key || 'local'
}
