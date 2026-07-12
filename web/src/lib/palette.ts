
export const PROTO = {
  routier: '#16a34a',
  static: '#22c55e',
  ospf: '#9333ea',
  ospf6: '#0d9488',
  bgp: '#2563eb',
  zebra: '#3b82f6',
  kernel: '#6b7280',
  boot: '#9ca3af',
  dhcp: '#f59e0b',
  dhcpcd: '#f59e0b',
  ra: '#06b6d4',
} as const

export const TOPO = {
  bgp: '#2563eb',
  static: '#16a34a',
  ospf: '#9333ea',
  ospf6: '#0d9488',
  anycast: '#ea580c',
  tunnel: '#0891b2',
  wireguard: '#4f46e5',
  vrrp: '#0f766e',
  vrf: '#b45309',
} as const

const FALLBACK = '#6b7280'

export function protoColor(protocol: string): string {
  const p = protocol.toLowerCase()
  for (const [k, v] of Object.entries(PROTO)) {
    if (p.includes(k)) return v
  }
  return FALLBACK
}

export const STATE = {
  up: '#16a34a',
  down: '#dc2626',
  warn: '#d97706',
  unknown: '#6b7280',
} as const
