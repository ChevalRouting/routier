export function cidrNetwork(cidr: string): string | null {
  const slash = cidr.indexOf('/')
  if (slash < 0) return null

  const ip = cidr.slice(0, slash).trim()
  const prefix = parseInt(cidr.slice(slash + 1), 10)
  if (isNaN(prefix)) return null

  const v6 = ip.includes(':')
  const bitsLen = v6 ? 128 : 32
  if (prefix < 0 || prefix > bitsLen) return null

  const addr = v6 ? parseV6(ip) : parseV4(ip)
  if (addr === null) return null

  const mask = prefix === 0 ? 0n : (~0n << BigInt(bitsLen - prefix)) & ((1n << BigInt(bitsLen)) - 1n)
  const net = addr & mask
  return (v6 ? formatV6(net) : formatV4(net)) + '/' + prefix
}

function parseV4(ip: string): bigint | null {
  const o = ip.split('.')
  if (o.length !== 4) return null

  let n = 0n
  for (const part of o) {
    const x = Number(part)
    if (!Number.isInteger(x) || x < 0 || x > 255) return null
    n = (n << 8n) | BigInt(x)
  }
  return n
}

function formatV4(n: bigint): string {
  return [24n, 16n, 8n, 0n].map((s) => ((n >> s) & 255n).toString()).join('.')
}

function parseV6(ip: string): bigint | null {
  const halves = ip.split('::')
  if (halves.length > 2) return null

  const head = halves[0] ? halves[0].split(':') : []
  const tail = halves.length === 2 && halves[1] ? halves[1].split(':') : []

  let groups: string[]
  if (halves.length === 2) {
    const missing = 8 - (head.length + tail.length)
    if (missing < 0) return null
    groups = [...head, ...Array(missing).fill('0'), ...tail]
  } else {
    groups = head
  }
  if (groups.length !== 8) return null

  let n = 0n
  for (const g of groups) {
    if (!/^[0-9a-fA-F]{1,4}$/.test(g)) return null
    n = (n << 16n) | BigInt(parseInt(g, 16))
  }
  return n
}

function formatV6(n: bigint): string {
  const groups: string[] = []
  for (let i = 7; i >= 0; i--) groups.push(((n >> BigInt(i * 16)) & 0xffffn).toString(16))

  let bestStart = -1
  let bestLen = 0
  let curStart = -1
  let curLen = 0
  for (let i = 0; i < 8; i++) {
    if (groups[i] === '0') {
      if (curStart < 0) curStart = i
      curLen++
      if (curLen > bestLen) { bestLen = curLen; bestStart = curStart }
    } else {
      curStart = -1
      curLen = 0
    }
  }

  if (bestLen > 1) {
    return groups.slice(0, bestStart).join(':') + '::' + groups.slice(bestStart + bestLen).join(':')
  }
  return groups.join(':')
}
