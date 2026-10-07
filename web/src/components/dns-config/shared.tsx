import { reverseDnsName } from '@/lib/utils'
import { Input, Label, Switch } from 'cheval-ui'

type ParseRecordLinesShape = { records: DnsRecord[]; errors: string[] }

type CacheSettingsShape = { cfg: DnsServerData; set: DnsSet }

export const RECORD_TYPES = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'SRV', 'PTR', 'NS', 'CAA', 'SSHFP', 'TLSA']

export const MODE_INFERRED = '__inferred__'

export const MODES = [MODE_INFERRED, 'forwarder', 'authoritative', 'both']

export const fqdn = (s: string) => {
  const t = s.trim().toLowerCase()
  return !t || t.endsWith('.') ? t : `${t}.`
}

export const soaEmail = (email: string) => {
  const at = email.indexOf('@')
  if (at < 0) return fqdn(email)
  return fqdn(`${email.slice(0, at).replace(/\./g, '\\.')}.${email.slice(at + 1)}`)
}

export function ownerFqdn(name: string, zoneName: string): string {
  const n = name.trim()
  if (!n || n === '@') return fqdn(zoneName)
  if (n.endsWith('.')) return n.toLowerCase()
  return fqdn(`${n}.${zoneName}`)
}

export interface ReverseTarget {
  index: number
  owner: string
  zoneName: string
}

export function reverseTarget(value: string, zones: DnsZone[]): ReverseTarget | null {
  const rev = reverseDnsName(value)
  if (!rev) return null

  const target = rev.replace(/\.$/, '').toLowerCase()
  let index = -1
  let bestLen = -1
  zones.forEach((z, i) => {
    if ((z.primaries ?? []).length > 0) return
    const zn = z.name.trim().replace(/\.$/, '').toLowerCase()
    if (!zn) return

    if ((target === zn || target.endsWith(`.${zn}`)) && zn.length > bestLen) {
      bestLen = zn.length
      index = i
    }
  })

  if (index < 0) return null

  const zn = zones[index].name.trim().replace(/\.$/, '').toLowerCase()
  const owner = target === zn ? '@' : target.slice(0, target.length - zn.length - 1)
  return { index, owner, zoneName: zn }
}

export function ptrExists(zone: DnsZone, owner: string, value: string): boolean {
  const want = fqdn(value)
  return (zone.records ?? []).some((r) => r.type === 'PTR' && r.name.trim() === owner && fqdn(r.value) === want)
}

export function impliedZoneRecords(zone: DnsZone): DnsRecord[] {
  if (!zone.name) return []
  const soa = zone.soa ?? {}
  const nss = zone.nameservers ?? []
  const primary = soa.primary ? fqdn(soa.primary) : nss[0] ? fqdn(nss[0]) : fqdn(`ns.${zone.name}`)
  const email = soa.email ? soaEmail(soa.email) : fqdn(`hostmaster.${zone.name}`)

  return [
    { name: '@', type: 'SOA', value: `${primary} ${email}` },
    ...nss.map((ns) => ({ name: '@', type: 'NS', value: fqdn(ns) })),
  ]
}

export interface DnsRecord {
  name: string
  type: string
  value: string
  ttl?: number
  priority?: number
}

export interface DnsSOA {
  primary?: string
  email?: string
  serial?: number
  refresh?: number
  retry?: number
  expire?: number
  minimum?: number
}

export interface DnsZone {
  name: string
  ttl?: number
  nameservers?: string[]
  soa?: DnsSOA
  records?: DnsRecord[]
  primaries?: string[]
  dnssec?: boolean
}

export interface DnsForward {
  domain: string
  servers?: string[]
  dnssec?: boolean
}

export interface DnsCache {
  disabled?: boolean
  size?: string
  min_ttl?: number
  max_ttl?: number
  max_negative_ttl?: number
  serve_expired?: boolean
}

export type DnsSet = <K extends keyof DnsServerData>(k: K, v: DnsServerData[K]) => void

export interface DnsServerData {
  enabled?: boolean
  mode?: string
  listen?: string[]
  port?: number
  allow_from?: string[]
  allow_inbound?: string[]
  upstreams?: string[]
  forward?: DnsForward[]
  cache?: DnsCache
  dnssec?: boolean
  log_queries?: boolean
  zones?: DnsZone[]
}

export interface IfaceData {
  addresses?: string[]
}

export function listenOptions(ifaces: Record<string, IfaceData>, vrrpIfaces: string[]): string[] {
  const out: string[] = ['127.0.0.1', '0.0.0.0']
  for (const [name, iface] of Object.entries(ifaces ?? {})) {
    if ((iface?.addresses ?? []).some((a) => a !== 'dhcp' && a !== 'dhcp4' && a !== 'dhcp6' && a !== 'slaac')) {
      out.push(`iface(${name})`)
    }
  }

  for (const name of vrrpIfaces) {
    out.push(`vips(${name})`)
  }

  return out
}

export function parseRecordLines(text: string): ParseRecordLinesShape {
  const records: DnsRecord[] = []
  const errors: string[] = []
  for (const raw of text.split('\n')) {
    const line = raw.trim()
    if (!line || line.startsWith(';') || line.startsWith('#')) continue

    const tokens = line.split(/\s+/).filter((t) => t.toUpperCase() !== 'IN')
    let i = 1
    let ttl: number | undefined
    if (tokens.length > i && /^\d+$/.test(tokens[i])) {
      ttl = Number(tokens[i])
      i++
    }

    const name = tokens[0]
    const type = tokens[i]?.toUpperCase()
    const rest = tokens.slice(i + 1)
    if (!name || !type || !RECORD_TYPES.includes(type) || rest.length === 0) {
      errors.push(line)
      continue
    }

    const rec: DnsRecord = { name, type, value: rest.join(' ') }
    if (ttl) rec.ttl = ttl
    if ((type === 'MX' || type === 'SRV') && rest.length > 1 && /^\d+$/.test(rest[0])) {
      rec.priority = Number(rest[0])
      rec.value = rest.slice(1).join(' ')
    }

    records.push(rec)
  }

  return { records, errors }
}

export function CacheSettings({ cfg, set }: CacheSettingsShape) {
  const cache = cfg.cache ?? {}

  return (
    <div className="space-y-4">
      <label className="flex items-center gap-3">
        <Switch
          checked={!!cache.disabled}
          onCheckedChange={(v) => set('cache', { ...cache, disabled: v || undefined })}
        />
        <div>
          <div className="text-sm font-medium">Disable caching</div>
          <div className="text-xs text-muted-foreground">Clamps every TTL to zero rather than sizing the cache to nothing.</div>
        </div>
      </label>

      {!cache.disabled && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div>
            <Label className="text-xs">Max cache size</Label>
            <Input className="h-8 font-mono text-xs" placeholder="64m" value={cache.size ?? ''} onChange={(e) => set('cache', { ...cache, size: e.target.value || undefined })} />
          </div>
          <div>
            <Label className="text-xs">Min TTL</Label>
            <Input className="h-8 font-mono text-xs" value={cache.min_ttl ?? ''} onChange={(e) => set('cache', { ...cache, min_ttl: e.target.value ? Number(e.target.value) : undefined })} />
          </div>
          <div>
            <Label className="text-xs">Max TTL</Label>
            <Input className="h-8 font-mono text-xs" value={cache.max_ttl ?? ''} onChange={(e) => set('cache', { ...cache, max_ttl: e.target.value ? Number(e.target.value) : undefined })} />
          </div>
          <div>
            <Label className="text-xs">Max negative TTL</Label>
            <Input className="h-8 font-mono text-xs" value={cache.max_negative_ttl ?? ''} onChange={(e) => set('cache', { ...cache, max_negative_ttl: e.target.value ? Number(e.target.value) : undefined })} />
          </div>
        </div>
      )}
    </div>
  )
}
