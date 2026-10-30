import type { ConfigFlowCollector as FlowCollector } from '@/api'
import { isIP, isIPv6, isPort } from '@/lib/validate'

export function formatCollector(c: FlowCollector): string {
  if (!c.port) return c.address
  return isIPv6(c.address) ? `[${c.address}]:${c.port}` : `${c.address}:${c.port}`
}

type AddressParts = { address: string; port?: string }

function split(text: string): AddressParts {
  const bracketed = /^\[([^\]]+)\](?::(\d+))?$/.exec(text)
  if (bracketed) return { address: bracketed[1], port: bracketed[2] }
  const parts = text.split(':')
  if (parts.length === 2) return { address: parts[0], port: parts[1] }
  return { address: text }
}

export function parseCollector(text: string): FlowCollector | null {
  const { address, port } = split(text.trim())
  if (!isIP(address)) return null
  if (port === undefined) return { address }
  const n = Number(port)
  return /^\d+$/.test(port) && isPort(n) ? { address, port: n } : null
}
