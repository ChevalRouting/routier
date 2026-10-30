
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
  up: '#26a269',
  standby: '#3584e4',
  down: '#e01b24',
  warn: '#e5a50a',
  unknown: '#77767b',
} as const

export const ON_COLOR = '#ffffff'

export const PRIMARY = 'hsl(var(--primary))'

export const PRIMARY_FOREGROUND = 'hsl(var(--primary-foreground))'

export const BORDER = 'hsl(var(--border))'

export const SERIES = {
  cpu: '#3584e4',
  mem: '#9141ac',
  rx: '#3584e4',
  tx: '#26a269',
  rtt: '#3584e4',
} as const

export const SERIES_CYCLE = ['#3584e4', '#26a269', '#e5a50a', '#9141ac', '#e66100', '#2190a4', '#c64600', '#865e3c'] as const

export const VRRP_STATE_COLOR: Record<string, string> = {
  MASTER: STATE.up,
  BACKUP: STATE.standby,
  FAULT: STATE.down,
  UNKNOWN: STATE.unknown,
}

export const OSPF_STATE_COLOR: Record<string, string> = {
  'Full/DR': STATE.up,
  'Full/BDR': STATE.up,
  'Full/Other': STATE.up,
  '2-Way/Other': STATE.warn,
}

const TOPO_PREFIXES: [string, keyof typeof TOPO][] = [
  ['bgp', 'bgp'], ['static', 'static'], ['ospf6', 'ospf6'], ['ospf', 'ospf'], ['anycast', 'anycast'],
  ['tunnel', 'tunnel'], ['wg', 'wireguard'], ['vrrp', 'vrrp'], ['vrf', 'vrf'],
]

export function topoNodeColor(id: string): string {
  return TOPO[TOPO_PREFIXES.find(([prefix]) => id.startsWith(prefix))?.[1] ?? 'static'] ?? FALLBACK
}
