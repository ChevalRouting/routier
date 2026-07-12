import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

const TOKEN_KEY = 'routier_token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

function expandIPv6(addr: string): number[] {
  const zoneIdx = addr.indexOf('%')
  if (zoneIdx >= 0) addr = addr.slice(0, zoneIdx)
  if (addr === '::') return new Array(16).fill(0)

  const dbl = addr.indexOf('::')
  let left: string[], right: string[]
  if (dbl === -1) {
    left = addr.split(':')
    right = []
  } else {
    left = dbl > 0 ? addr.slice(0, dbl).split(':') : []
    right = dbl < addr.length - 2 ? addr.slice(dbl + 2).split(':') : []
  }

  const fill = 8 - left.length - right.length
  const groups = [
    ...left.map(h => parseInt(h || '0', 16)),
    ...new Array(fill).fill(0),
    ...right.map(h => parseInt(h || '0', 16)),
  ]

  const bytes: number[] = []
  for (const g of groups) bytes.push((g >> 8) & 0xff, g & 0xff)
  return bytes
}

function parseCIDR(cidr: string, family: string): [number[], number] {
  const isDefault = cidr === 'default'
  const raw = isDefault ? (family === 'ipv4' ? '0.0.0.0' : '::') : (cidr.includes('/') ? cidr.split('/')[0] : cidr)
  const pfx = isDefault ? 0 : (cidr.includes('/') ? parseInt(cidr.split('/')[1]) : (family === 'ipv4' ? 32 : 128))
  const bytes = family === 'ipv4' ? raw.split('.').map(Number) : expandIPv6(raw)
  return [bytes, pfx]
}

export function compareIPRoutes(
  a: { dst: string; family: string },
  b: { dst: string; family: string },
): number {
  if (a.family !== b.family) return a.family === 'ipv4' ? -1 : 1
  const [aBytes, aPfx] = parseCIDR(a.dst, a.family)
  const [bBytes, bPfx] = parseCIDR(b.dst, b.family)
  if (aPfx !== bPfx) return aPfx - bPfx
  for (let i = 0; i < Math.min(aBytes.length, bBytes.length); i++) {
    if (aBytes[i] !== bBytes[i]) return aBytes[i] - bBytes[i]
  }
  return 0
}
