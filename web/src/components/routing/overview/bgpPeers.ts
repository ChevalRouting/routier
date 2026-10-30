import type { TypesBGPPeerSummary as BGPPeerSummary } from '@/api'
import { cidrContains } from '@/lib/cidr'
import { STATE } from '@/lib/palette'
import { BGPNeighbor, WgIface } from './types'

export interface BGPSession {
  neighbor: BGPNeighbor
  iface?: string
  status?: BGPPeerSummary
}

export interface BGPPeerGroup {
  id: string
  name: string
  asns: number[]
  ifaces: string[]
  sessions: BGPSession[]
}

export interface BGPPeerHealth {
  up: number
  total: number
  known: boolean
  color: string
}

export interface BGPTunnel {
  iface?: string
  health: BGPPeerHealth
}

const ESTABLISHED = 'Established'

function neighborIface(address: string, wireguard: Record<string, WgIface>): string | undefined {
  const scope = address.split('%')[1]
  if (scope) return scope
  return Object.entries(wireguard).find(([, w]) => (w.addresses ?? []).some((a) => cidrContains(a, address)))?.[0]
}

function endpointHost(endpoint?: string): string | undefined {
  if (!endpoint) return undefined
  const match = endpoint.match(/^\[(.+)\]:\d+$/) ?? endpoint.match(/^([^:]+):\d+$/)
  return match ? match[1] : endpoint
}

function remoteKey(iface: string, wireguard: Record<string, WgIface>): string {
  const w = wireguard[iface]
  if (!w) return iface
  return w.friend || endpointHost(w.peers?.find((p) => p.endpoint)?.endpoint) || iface
}

function commonWords(texts: string[]): string {
  if (texts.length === 0) return ''

  const [first, ...rest] = texts.map((t) => t.split(/\s+/))
  const common: string[] = []
  for (const [i, word] of first.entries()) {
    if (!rest.every((words) => words[i] === word)) break
    common.push(word)
  }
  return common.join(' ')
}

function groupName(key: string, sessions: BGPSession[], wireguard: Record<string, WgIface>): string {
  const friend = sessions.map((s) => s.iface && wireguard[s.iface]?.friend).find(Boolean)
  if (friend) return friend

  const descriptions = sessions.map((s) => s.neighbor.description ?? '')
  return commonWords(descriptions) || key
}

export function groupBGPPeers(neighbors: BGPNeighbor[], wireguard: Record<string, WgIface>, statuses: BGPPeerSummary[]): BGPPeerGroup[] {
  const statusByAddress = new Map(statuses.map((s) => [s.address.split('%')[0], s]))
  const groups = new Map<string, BGPPeerGroup>()

  for (const neighbor of neighbors) {
    const iface = neighborIface(neighbor.address, wireguard)
    const key = iface ? remoteKey(iface, wireguard) : neighbor.address
    const group = groups.get(key) ?? { id: `bgp-${key}`, name: key, asns: [], ifaces: [], sessions: [] }
    group.sessions.push({ neighbor, iface, status: statusByAddress.get(neighbor.address.split('%')[0]) })
    if (!group.asns.includes(neighbor.remote_asn)) group.asns.push(neighbor.remote_asn)
    if (iface && !group.ifaces.includes(iface)) group.ifaces.push(iface)
    groups.set(key, group)
  }

  return [...groups.entries()].map(([key, group]) => ({ ...group, name: groupName(key, group.sessions, wireguard) }))
}

export function sessionHealth(sessions: BGPSession[]): BGPPeerHealth {
  const total = sessions.length
  const known = sessions.some((s) => s.status)
  const up = sessions.filter((s) => s.status?.state === ESTABLISHED).length
  if (!known) return { up, total, known, color: STATE.unknown }
  if (up === total) return { up, total, known, color: STATE.up }
  if (up === 0) return { up, total, known, color: STATE.down }
  return { up, total, known, color: STATE.warn }
}

export function peerHealth(group: BGPPeerGroup): BGPPeerHealth {
  return sessionHealth(group.sessions)
}

export function peerTunnels(group: BGPPeerGroup): BGPTunnel[] {
  if (group.ifaces.length === 0) return [{ health: peerHealth(group) }]
  return group.ifaces.map((iface) => ({ iface, health: sessionHealth(group.sessions.filter((s) => s.iface === iface)) }))
}
