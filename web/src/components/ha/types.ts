export interface VRRPInstance {
  name?: string
  friend?: string
  id: number
  interface: string
  transport?: string
  vips: string[]
  priority?: number
  password?: string
  track_interfaces?: string[]
  virtual_routes?: string[]
  allow_inbound?: boolean
  bgp_prepend?: VRRPBGPPrepend
}

export interface VRRPBGPPrepend {
  count?: number
  neighbors?: string[]
  prefixes?: string[]
}

export interface Conntrackd {
  interface: string
  address: string
  peer_ips: string[]
  port?: number
  allow_inbound?: boolean
}

export interface HAConfigData {
  vrrp?: VRRPInstance[]
  conntrackd?: Conntrackd | null
}
