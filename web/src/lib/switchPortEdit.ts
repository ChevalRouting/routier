import type { TypesSwitchState as SwitchState } from '@/api'
import type { SwitchDefinition, SwitchPortDefinition } from './switches'

export type PortRole = 'port' | 'bond' | 'unmanaged'

export interface PortEdit {
  role: PortRole
  port: SwitchPortDefinition
  bondName: string
  bondMode: string
  members: string[]
}

export interface PortEditContext {
  ports: string[]
  definition: SwitchDefinition
  state: SwitchState | undefined
}

export interface ReleasedBond {
  name: string
  remaining: string[]
  recreated: boolean
}

export const DEFAULT_BOND_MODE = '802.3ad'

export const BOND_MODES = [DEFAULT_BOND_MODE, 'active-backup', 'balance-xor', 'balance-rr', 'balance-alb', 'balance-tlb', 'broadcast']

export function bondMembers(name: string, definition: SwitchDefinition, state: SwitchState | undefined): string[] {
  return definition.bonds?.[name]?.members ?? state?.bonds.find((bond) => bond.name === name)?.members ?? []
}

export function bondOfPort(port: string, definition: SwitchDefinition, state: SwitchState | undefined): string {
  if (definition.bonds?.[port]) return port
  const configured = Object.entries(definition.bonds ?? {}).find(([, bond]) => bond.members.includes(port))
  if (configured) return configured[0]
  return state?.ports.find((candidate) => candidate.name === port)?.bond ?? ''
}

export function currentPort(name: string, definition: SwitchDefinition, state: SwitchState | undefined): SwitchPortDefinition {
  const configured = definition.ports?.[name] ?? definition.bonds?.[name]?.port
  if (configured) return configured
  const live = state?.ports.find((port) => port.name === name)
  if (live?.mode === 'trunk') return { mode: 'trunk', native: live.pvid > 1 ? live.pvid : 0, tagged: live.tagged }
  return { mode: 'access', vlan: live?.pvid && live.pvid > 1 ? live.pvid : undefined }
}

export function nextBondName(definition: SwitchDefinition, state: SwitchState | undefined): string {
  const taken = new Set([...Object.keys(definition.bonds ?? {}), ...(state?.bonds ?? []).map((bond) => bond.name)])
  let index = 1
  while (taken.has(`bond${index}`)) index += 1
  return `bond${index}`
}

export function memberCandidates(state: SwitchState | undefined): string[] {
  return (state?.ports ?? []).filter((port) => port.type !== 'bond').map((port) => port.name)
}

export function sharedBond({ ports, definition, state }: PortEditContext): string {
  const bonds = new Set(ports.map((port) => bondOfPort(port, definition, state)))
  return bonds.size === 1 ? [...bonds][0] : ''
}

function initialRole(context: PortEditContext, bond: string): PortRole {
  if (bond) return 'bond'
  if (context.ports.length > 1) return 'port'
  return context.definition.ports?.[context.ports[0]] ? 'port' : 'unmanaged'
}

export function initialEdit(context: PortEditContext): PortEdit {
  const { ports, definition, state } = context
  const bond = sharedBond(context)
  const liveMode = state?.bonds.find((candidate) => candidate.name === bond)?.mode

  return {
    role: initialRole(context, bond),
    port: currentPort(bond || ports[0], definition, state),
    bondName: bond || nextBondName(definition, state),
    bondMode: definition.bonds?.[bond]?.mode ?? liveMode ?? DEFAULT_BOND_MODE,
    members: bond ? bondMembers(bond, definition, state) : ports,
  }
}

function claimed(edit: PortEdit, ports: string[]): string[] {
  if (edit.role === 'bond') return edit.members
  return ports
}

function bondModeOf(name: string, definition: SwitchDefinition, state: SwitchState | undefined): string {
  return definition.bonds?.[name]?.mode ?? state?.bonds.find((bond) => bond.name === name)?.mode ?? DEFAULT_BOND_MODE
}

function keepsBond(edit: PortEdit, bond: string, context: PortEditContext): boolean {
  const { definition, state } = context
  return bond === edit.bondName.trim() && sameMembers(bondMembers(bond, definition, state), edit.members) && bondModeOf(bond, definition, state) === edit.bondMode
}

function releasedBondNames(edit: PortEdit, context: PortEditContext): string[] {
  const { definition, state } = context
  const taken = claimed(edit, context.ports)
  const bonds = new Set(taken.map((port) => bondOfPort(port, definition, state)).filter(Boolean))
  if (edit.role === 'unmanaged') return [...bonds].filter((bond) => !!definition.bonds?.[bond])

  const original = sharedBond(context)
  if (edit.role === 'bond' && original) bonds.add(original)
  if (edit.role !== 'bond') return [...bonds]

  return [...bonds].filter((bond) => !keepsBond(edit, bond, context))
}

function sameMembers(a: string[], b: string[]): boolean {
  return a.length === b.length && a.every((member) => b.includes(member))
}

export function releasedBonds(edit: PortEdit, context: PortEditContext): ReleasedBond[] {
  const taken = new Set(claimed(edit, context.ports))
  const name = edit.bondName.trim()

  return releasedBondNames(edit, context).map((bond) => ({
    name: bond,
    remaining: bondMembers(bond, context.definition, context.state).filter((member) => !taken.has(member)),
    recreated: edit.role === 'bond' && bond === name,
  }))
}

function withoutPorts(ports: Record<string, SwitchPortDefinition> | undefined, names: string[]): Record<string, SwitchPortDefinition> {
  const next = { ...ports }
  names.forEach((name) => delete next[name])
  return next
}

export function applyEdit(edit: PortEdit, context: PortEditContext): SwitchDefinition {
  const { definition } = context
  const taken = claimed(edit, context.ports)
  const bonds = { ...definition.bonds }
  releasedBondNames(edit, context).forEach((bond) => delete bonds[bond])
  const ports = withoutPorts(definition.ports, taken)

  if (edit.role === 'port') taken.forEach((port) => { ports[port] = edit.port })
  if (edit.role === 'bond') bonds[edit.bondName.trim()] = { members: edit.members, mode: edit.bondMode === DEFAULT_BOND_MODE ? undefined : edit.bondMode, port: edit.port }

  return { ...definition, ports, bonds }
}

export function editError(edit: PortEdit, context: PortEditContext, vlanCount: number): string {
  if (edit.role === 'unmanaged') return ''
  if (edit.port.mode === 'access' && !edit.port.vlan) return vlanCount === 0 ? 'Declare VLAN interfaces first' : 'Pick a VLAN'
  if (edit.role !== 'bond') return ''

  const name = edit.bondName.trim()
  if (!name) return 'Name the bond'
  if (edit.members.length < 2) return 'A bond needs at least two members'
  if (context.definition.ports?.[name] || memberCandidates(context.state).includes(name)) return `${name} is a port name`

  const original = sharedBond(context)
  const clash = name !== original && !!(context.definition.bonds?.[name] || context.state?.bonds.some((bond) => bond.name === name))
  return clash ? `${name} already exists` : ''
}
