export type ConfigLayer = 'advanced' | 'simple'

const STORAGE_KEY = 'routier_config_layer'

let active: ConfigLayer = localStorage.getItem(STORAGE_KEY) === 'advanced' ? 'advanced' : 'simple'
const listeners = new Set<(layer: ConfigLayer) => void>()

export function getConfigLayer(): ConfigLayer {
  return active
}

export function setConfigLayer(layer: ConfigLayer): void {
  active = layer
  localStorage.setItem(STORAGE_KEY, layer)
  listeners.forEach((listener) => listener(layer))
}

export function subscribeConfigLayer(listener: (layer: ConfigLayer) => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

const advancedOnlySections = new Set([
  'interfaces', 'tunnels', 'vrfs', 'routing', 'wireguard', 'nftables',
  'sysctl', 'services', 'logging', 'friends', 'conntrackd', 'ssh',
  'boot_modules', 'gai', 'dhcp', 'dns_server',
])

export function withConfigLayer(url: string, layer: ConfigLayer = active): string {
  if (!url.includes('/api/config')) return url
  const parsed = new URL(url, window.location.origin)
  if (!parsed.searchParams.has('layer')) {
    const section = parsed.pathname.match(/\/api\/config\/([^/]+)$/)?.[1]
    parsed.searchParams.set('layer', section && advancedOnlySections.has(section) ? 'advanced' : layer)
  }
  return url.startsWith('http') ? parsed.toString() : `${parsed.pathname}${parsed.search}`
}
