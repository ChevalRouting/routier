import type { BGPConfig } from '@/components/routing/types'

export function migrateBGPPolicies(bgp: BGPConfig): BGPConfig {
  const families = bgp.address_families ?? {}
  if (!Object.values(families).some((af) => af.route_map_in || af.route_map_out)) return bgp
  const neighbors = bgp.neighbors.map((neighbor) => {
    const afs = neighbor.address_families && Object.keys(neighbor.address_families).length
      ? neighbor.address_families
      : { [neighbor.address.includes(':') ? 'ipv6-unicast' : 'ipv4-unicast']: {} }
    const address_families = { ...afs }
    for (const [name, af] of Object.entries(afs)) {
      const family = families[name]
      if (!family?.route_map_in && !family?.route_map_out) continue
      address_families[name] = {
        ...af,
        route_map_in: af.route_map_in || family.route_map_in,
        route_map_out: af.route_map_out || family.route_map_out,
      }
    }
    return { ...neighbor, address_families }
  })
  const address_families = Object.fromEntries(Object.entries(families).map(([name, af]) => {
    const { route_map_in: _in, route_map_out: _out, ...rest } = af
    return [name, rest]
  }))
  return { ...bgp, neighbors, address_families }
}
