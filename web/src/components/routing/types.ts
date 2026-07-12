import { RADVDConfig } from './RadvdTab'

export interface StaticRoute {
  destination: string
  via: string
  dev: string
  metric: number
}

export interface BGPAddressFamily {
  networks?: string[]
  redistribute?: string[]
  route_map_in?: string
  route_map_out?: string
  default_originate?: boolean
  maximum_paths?: number
  import_vrf?: string[]
  extra?: string[]
}

export interface BGPNeighborAF {
  disabled?: boolean
  soft_reconfiguration?: boolean
  next_hop_self?: boolean
  route_reflector_client?: boolean
  remove_private_as?: boolean
  allowas_in?: number
  weight?: number
  prefix_list_in?: string
  prefix_list_out?: string
  route_map_in?: string
  route_map_out?: string
  default_originate?: boolean
  extra?: string[]
}

export interface PrefixEntry {
  seq: number
  action: 'permit' | 'deny'
  prefix: string
  ge?: number
  le?: number
}

export interface RouteMapEntry {
  seq: number
  action: 'permit' | 'deny'
  match?: Record<string, string>
  set?: Record<string, string>
  call?: string
  on_match?: string
  continue?: number
}

export interface BGPNeighbor {
  address: string
  remote_asn: number
  description?: string
  password?: string
  update_source?: string
  ebgp_multihop?: number
  disable_connected_check?: boolean
  passive?: boolean
  shutdown?: boolean
  bfd?: boolean
  bfd_profile?: string
  extra?: string[]
  address_families?: Record<string, BGPNeighborAF>
}

export interface BFDProfile {
  name: string
  detect_multiplier?: number
  receive_interval?: number
  transmit_interval?: number
  minimum_ttl?: number
  echo_mode?: boolean
  passive_mode?: boolean
  extra?: string[]
}

export interface BFDConfig {
  profiles: BFDProfile[]
}

export interface BGPConfig {
  asn: number
  router_id: string
  no_ebgp_requires_policy?: boolean
  no_default_ipv4_unicast?: boolean
  no_import_check?: boolean
  neighbors: BGPNeighbor[]
  address_families?: Record<string, BGPAddressFamily>
  prefix_lists?: Record<string, PrefixEntry[]>
  route_maps?: Record<string, RouteMapEntry[]>
  extra?: string[]
}

export interface OSPFArea {
  id: string
  networks: string[]
  type: string
  stub_no_summary?: boolean
  ranges?: string[]
  default_cost?: number
  auth?: string
  extra?: string[]
}

export interface OSPFInterfaceConfig {
  area?: string
  cost?: number
  hello_interval?: number
  dead_interval?: number
  network_type?: string
  priority?: number
  retransmit_interval?: number
  transmit_delay?: number
  mtu_ignore?: boolean
  bfd?: boolean
  auth_type?: string
  auth_key?: string
  auth_key_id?: number
  extra?: string[]
}

export interface OSPFConfig {
  router_id: string
  areas: OSPFArea[]
  passive_interfaces: string[]
  redistribute: string[]
  interfaces?: Record<string, OSPFInterfaceConfig>
  default_information_originate?: boolean
  reference_bandwidth?: number
  distance?: number
  extra?: string[]
}

export interface OSPF6Area {
  id: string
  ranges: string[]
  type?: string
  stub_no_summary?: boolean
  default_cost?: number
  extra?: string[]
}

export interface OSPF6InterfaceConfig {
  area?: string
  cost?: number
  hello_interval?: number
  dead_interval?: number
  network_type?: string
  priority?: number
  retransmit_interval?: number
  transmit_delay?: number
  mtu_ignore?: boolean
  bfd?: boolean
  extra?: string[]
}

export interface OSPF6Config {
  router_id: string
  areas: OSPF6Area[]
  passive_interfaces: string[]
  redistribute: string[]
  interfaces?: Record<string, OSPF6InterfaceConfig>
  default_information_originate?: boolean
  reference_bandwidth?: number
  distance?: number
  extra?: string[]
}

export interface VRFRouting {
  static?: StaticRoute[]
  bgp?: BGPConfig
  ospf?: OSPFConfig
  ospf6?: OSPF6Config
}

export interface PBRNexthop {
  address?: string
  dev?: string
  nexthop_vrf?: string
}

export interface PBRNexthopGroup {
  nexthops: PBRNexthop[]
}

export interface PBRMapEntry {
  seq: number
  match_src?: string
  match_dst?: string
  set_nexthop_group?: string
  set_nexthop?: string
}

export interface PBRConfig {
  nexthop_groups?: Record<string, PBRNexthopGroup>
  maps?: Record<string, PBRMapEntry[]>
  policies?: Record<string, string>
}

export interface RoutingConfig {
  static: StaticRoute[]
  bgp?: BGPConfig
  ospf?: OSPFConfig
  ospf6?: OSPF6Config
  radvd?: RADVDConfig
  anycast?: unknown
  vrfs?: Record<string, VRFRouting>
  pbr?: PBRConfig
  bfd?: BFDConfig
}

export const STANDARD_AFS = [
  'ipv4 unicast',
  'ipv6 unicast',
  'ipv4 multicast',
  'ipv6 multicast',
  'l2vpn evpn',
]

export const OSPF_TYPES = ['normal', 'stub', 'nssa']
export const OSPF_NETWORK_TYPES = ['broadcast', 'non-broadcast', 'point-to-point', 'point-to-multipoint']
export type Tab = 'static' | 'bgp' | 'ospf' | 'ospf6' | 'bfd' | 'radvd' | 'vrfs' | 'pbr' | 'nat'
export type BGPSubTab = 'general' | 'neighbors' | 'afs' | 'prefix-lists' | 'route-maps'
export type OSPFSubTab = 'general' | 'areas' | 'interfaces'
export type OSPF6SubTab = 'general' | 'areas' | 'interfaces'
export type PBRSubTab = 'nexthop-groups' | 'maps' | 'policies'

export function defaultBGP(): BGPConfig {
  return { asn: 65000, router_id: '', neighbors: [], address_families: {}, prefix_lists: {}, route_maps: {} }
}
export function defaultOSPF(): OSPFConfig {
  return { router_id: '', areas: [], passive_interfaces: [], redistribute: [], interfaces: {} }
}
export function defaultOSPF6(): OSPF6Config {
  return { router_id: '', areas: [], passive_interfaces: [], redistribute: [], interfaces: {} }
}
