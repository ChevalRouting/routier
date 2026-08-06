export interface Instance {
  id: string
  label: string
  baseUrl: string
}

export const SELF: Instance = { id: 'self', label: 'This Instance', baseUrl: '' }

const INSTANCE_KEY = 'routier_active_instance'

let active: Instance = loadActive()

function loadActive(): Instance {
  const raw = localStorage.getItem(INSTANCE_KEY)
  if (!raw) return SELF

  try {
    const inst = JSON.parse(raw) as Instance
    if (inst && inst.id && inst.id !== 'self') return inst
  } catch {
    return SELF
  }

  return SELF
}

type InstanceListener = (inst: Instance) => void
const listeners = new Set<InstanceListener>()

export function subscribeInstance(fn: InstanceListener): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

export function getActiveInstance(): Instance {
  return active
}

export function activeBaseUrl(): string {
  return active.baseUrl
}

export function friendBaseUrl(name: string): string {
  return `/api/friends/${encodeURIComponent(name)}/proxy`
}

export function switchInstance(inst: Instance): void {
  active = inst
  if (inst.id === 'self') localStorage.removeItem(INSTANCE_KEY)
  else localStorage.setItem(INSTANCE_KEY, JSON.stringify(inst))

  listeners.forEach((fn) => fn(inst))
}
