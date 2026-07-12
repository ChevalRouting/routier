export function isIPv4(s: string): boolean {
  const parts = s.split('.')
  if (parts.length !== 4) return false
  return parts.every((p) => /^\d{1,3}$/.test(p) && Number(p) <= 255 && String(Number(p)) === p)
}

export function isIPv6(s: string): boolean {
  if (s === '' || s.indexOf(':') === -1) return false
  const halves = s.split('::')
  if (halves.length > 2) return false

  const parseGroups = (g: string): string[] | null => {
    if (g === '') return []
    const groups = g.split(':')
    return groups.every((h) => /^[0-9a-fA-F]{1,4}$/.test(h)) ? groups : null
  }

  if (halves.length === 2) {
    const left = parseGroups(halves[0])
    const right = parseGroups(halves[1])
    if (left === null || right === null) return false
    return left.length + right.length <= 7
  }
  const groups = parseGroups(s)
  return groups !== null && groups.length === 8
}

export function isIP(s: string): boolean {
  return isIPv4(s) || isIPv6(s)
}

export function isCIDR(s: string): boolean {
  const slash = s.lastIndexOf('/')
  if (slash === -1) return false
  const ip = s.slice(0, slash)
  const prefix = s.slice(slash + 1)
  if (!/^\d{1,3}$/.test(prefix)) return false
  const n = Number(prefix)
  if (isIPv4(ip)) return n >= 0 && n <= 32
  if (isIPv6(ip)) return n >= 0 && n <= 128
  return false
}

export function isIPOrCIDR(s: string): boolean {
  return isIP(s) || isCIDR(s)
}

export function isPort(n: number): boolean {
  return Number.isInteger(n) && n >= 1 && n <= 65535
}

export function isASN(n: number): boolean {
  return Number.isInteger(n) && n >= 1 && n <= 4294967295
}

export function isVLANId(n: number): boolean {
  return Number.isInteger(n) && n >= 1 && n <= 4094
}

export function isMTU(n: number): boolean {
  return Number.isInteger(n) && n >= 576 && n <= 9000
}

export function isMAC(s: string): boolean {
  return /^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$/.test(s)
}

export function isHostname(s: string): boolean {
  if (s.length === 0 || s.length > 253) return false
  return s.split('.').every((label) => /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/.test(label))
}

const opt = (s: string, ok: boolean, msg: string): string | null => (s.trim() === '' || ok ? null : msg)

export const checkIP = (s: string): string | null => opt(s, isIP(s), 'Enter a valid IP address')
export const checkCIDR = (s: string): string | null => opt(s, isCIDR(s), 'Enter a valid CIDR (e.g. 10.0.0.0/24)')
export const checkIPOrCIDR = (s: string): string | null => opt(s, isIPOrCIDR(s), 'Enter a valid IP or CIDR')
export const checkMAC = (s: string): string | null => opt(s, isMAC(s), 'Enter a valid MAC address')
export const checkHostname = (s: string): string | null => opt(s, isHostname(s), 'Enter a valid hostname')

export const checkASN = (s: string): string | null =>
  s.trim() === '' || isASN(Number(s)) ? null : 'ASN must be 1–4294967295'
export const checkPort = (s: string): string | null =>
  s.trim() === '' || isPort(Number(s)) ? null : 'Port must be 1–65535'
export const checkVLANId = (s: string): string | null =>
  s.trim() === '' || isVLANId(Number(s)) ? null : 'VLAN id must be 1–4094'
export const checkMTU = (s: string): string | null =>
  s.trim() === '' || isMTU(Number(s)) ? null : 'MTU must be 576–9000'
