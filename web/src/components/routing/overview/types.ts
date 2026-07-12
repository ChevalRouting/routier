import '@xyflow/react/dist/style.css'

export interface StaticRoute { destination: string; via?: string; dev?: string; metric?: number }
export interface BGPNeighbor { address: string; remote_asn: number; description?: string }
export interface BGP { asn?: number; router_id?: string; neighbors?: BGPNeighbor[] }
export interface OSPFArea { id: string; networks?: string[]; type?: string }
export interface OSPF { router_id?: string; areas?: OSPFArea[] }
export interface OSPF6Area { id: string; ranges?: string[]; type?: string }
export interface OSPF6 { router_id?: string; areas?: OSPF6Area[] }
export interface VRFRouting { bgp?: BGP; ospf?: OSPF; ospf6?: OSPF6; static?: StaticRoute[] }
export interface AnycastService { name: string; active?: boolean; anycast_ips?: string[] }
export interface RoutingConfig {
  static?: StaticRoute[]
  bgp?: BGP
  ospf?: OSPF
  ospf6?: OSPF6
  anycast?: { services?: AnycastService[] }
  vrfs?: Record<string, VRFRouting>
}
export interface Tunnel { mode: string; local?: string; remote?: string }
export interface WgIface { listen_port?: number; peers?: unknown[] }
