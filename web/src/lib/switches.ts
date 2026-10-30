import type { FaceplatePort, VLANOption } from 'cheval-ui'
import type { TypesSwitchPort as SwitchPort, TypesSwitchState as SwitchState } from '@/api'

export interface SwitchPortDefinition {
  mode: 'access' | 'trunk'
  vlan?: number
  native?: number
  tagged?: number[]
}

export interface SwitchBondDefinition {
  members: string[]
  mode?: string
  port?: SwitchPortDefinition
}

export interface SwitchDefinition {
  model: string
  host: string
  scheme?: 'https' | 'http'
  port?: number
  tls_skip_verify?: boolean
  interface?: string
  uplink?: string
  ports?: Record<string, SwitchPortDefinition>
  bonds?: Record<string, SwitchBondDefinition>
}

export type SwitchDefinitions = Record<string, SwitchDefinition>

type VLANTag = { id: number }

type InterfaceShape = { type?: string; vlan?: VLANTag }

export type InterfacesShape = Record<string, InterfaceShape>

export function faceplatePorts(state: SwitchState | undefined): FaceplatePort[] {
  return (state?.ports ?? []).map((port) => ({
    name: port.name, type: port.type, running: port.running, disabled: port.disabled, mode: port.mode,
    pvid: port.pvid, tagged: port.tagged, bond: port.bond || undefined, rate: port.rate, comment: port.comment,
  }))
}

function sameSet(a: number[], b: number[]): boolean {
  const left = [...new Set(a)].sort((x, y) => x - y)
  const right = [...new Set(b)].sort((x, y) => x - y)
  return left.length === right.length && left.every((value, index) => value === right[index])
}

function sameMembers(a: string[], b: string[]): boolean {
  const left = [...a].sort()
  const right = [...b].sort()
  return left.length === right.length && left.every((value, index) => value === right[index])
}

export function trunkTagged(want: SwitchPortDefinition): number[] {
  return [...new Set((want.tagged ?? []).filter((vid) => vid !== want.native))].sort((a, b) => a - b)
}

export function portDiffers(live: SwitchPort | undefined, want: SwitchPortDefinition): boolean {
  if (!live || !live.bridge) return true
  if (want.mode === 'access') return live.mode !== 'access' || live.pvid !== want.vlan || live.tagged.length > 0
  return live.mode !== 'trunk' || live.pvid !== Math.max(want.native ?? 0, 1) || !sameSet(live.tagged, trunkTagged(want))
}

export function pendingPorts(definition: SwitchDefinition | undefined, state: SwitchState | undefined): string[] {
  if (!definition || !state) return []
  const live = new Map(state.ports.map((port) => [port.name, port]))
  const liveBonds = new Map(state.bonds.map((bond) => [bond.name, bond]))
  const pending: string[] = []
  for (const [name, want] of Object.entries(definition.ports ?? {})) {
    if (portDiffers(live.get(name), want)) pending.push(name)
  }
  for (const [name, bond] of Object.entries(definition.bonds ?? {})) {
    const existing = liveBonds.get(name)
    const changed = !existing || !sameMembers(existing.members, bond.members) || (!!bond.port && portDiffers(live.get(name), bond.port))
    if (changed) pending.push(name, ...bond.members)
  }
  return pending
}

export function vlanOptions(interfaces: InterfacesShape | null, state: SwitchState | undefined): VLANOption[] {
  const options = new Map<number, VLANOption>()
  for (const [name, iface] of Object.entries(interfaces ?? {})) {
    if (iface.type === 'vlan' && iface.vlan?.id) options.set(iface.vlan.id, { vid: iface.vlan.id, name })
  }
  for (const vlan of state?.vlans ?? []) {
    if (vlan.id > 1 && !options.has(vlan.id)) options.set(vlan.id, { vid: vlan.id })
  }
  return [...options.values()].sort((a, b) => a.vid - b.vid)
}

export function describePortDefinition(want: SwitchPortDefinition): string {
  if (want.mode === 'access') return `access on VLAN ${want.vlan}`
  const native = want.native && want.native > 1 ? `native ${want.native}` : 'no native VLAN'
  return `trunk, ${native}, tagged ${trunkTagged(want).join(', ') || 'none'}`
}

export function desiredPort(name: string, definition: SwitchDefinition | undefined): string {
  const port = definition?.ports?.[name] ?? definition?.bonds?.[name]?.port
  if (port) return describePortDefinition(port)
  const bond = Object.entries(definition?.bonds ?? {}).find(([, value]) => value.members.includes(name))
  return bond ? `member of ${bond[0]}` : ''
}

export function portLink(port: SwitchPort): string {
  if (port.disabled) return 'disabled'
  return port.running ? port.rate || 'up' : 'down'
}
