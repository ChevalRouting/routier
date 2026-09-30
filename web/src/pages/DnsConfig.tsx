import { useState } from 'react'
import { Plus, Search, Trash2 } from 'lucide-react'
import { Button } from 'cheval-ui'
import { Input } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Switch } from 'cheval-ui'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { Textarea } from 'cheval-ui'
import { TagInput } from 'cheval-ui'
import { reverseDnsName } from '@/lib/utils'

const RECORD_TYPES = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'SRV', 'PTR', 'NS', 'CAA', 'SSHFP', 'TLSA']
const MODE_INFERRED = '__inferred__'
const MODES = [MODE_INFERRED, 'forwarder', 'authoritative', 'both']

const fqdn = (s: string) => {
  const t = s.trim().toLowerCase()
  return !t || t.endsWith('.') ? t : `${t}.`
}

const soaEmail = (email: string) => {
  const at = email.indexOf('@')
  if (at < 0) return fqdn(email)
  return fqdn(`${email.slice(0, at).replace(/\./g, '\\.')}.${email.slice(at + 1)}`)
}

function ownerFqdn(name: string, zoneName: string): string {
  const n = name.trim()
  if (!n || n === '@') return fqdn(zoneName)
  if (n.endsWith('.')) return n.toLowerCase()
  return fqdn(`${n}.${zoneName}`)
}

interface ReverseTarget {
  index: number
  owner: string
  zoneName: string
}

function reverseTarget(value: string, zones: DnsZone[]): ReverseTarget | null {
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

function ptrExists(zone: DnsZone, owner: string, value: string): boolean {
  const want = fqdn(value)
  return (zone.records ?? []).some((r) => r.type === 'PTR' && r.name.trim() === owner && fqdn(r.value) === want)
}

function impliedZoneRecords(zone: DnsZone): DnsRecord[] {
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

interface DnsRecord {
  name: string
  type: string
  value: string
  ttl?: number
  priority?: number
}

interface DnsSOA {
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

interface DnsForward {
  domain: string
  servers?: string[]
  dnssec?: boolean
}

interface DnsCache {
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

function listenOptions(ifaces: Record<string, IfaceData>, vrrpIfaces: string[]): string[] {
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

function ListenBuilder({ value, options, onChange }: {
  value: string[]
  options: string[]
  onChange: (v: string[]) => void
}) {
  const [pick, setPick] = useState('')

  const add = (entry: string) => {
    if (!entry || value.includes(entry)) return
    onChange([...value, entry])
    setPick('')
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-2">
        {value.map((v) => (
          <span key={v} className="flex items-center gap-1 rounded-md bg-muted/40 px-2 py-1 font-mono text-xs">
            {v}
            <button type="button" onClick={() => onChange(value.filter((x) => x !== v))} className="text-muted-foreground hover:text-foreground">
              <Trash2 className="h-3 w-3" />
            </button>
          </span>
        ))}
        {value.length === 0 && <span className="text-xs text-muted-foreground">No listen address set</span>}
      </div>
      <div className="flex gap-2">
        <Select value={pick} onValueChange={add}>
          <SelectTrigger className="h-8 w-64 text-xs">
            <SelectValue placeholder="Add an interface reference" />
          </SelectTrigger>
          <SelectContent>
            {options.filter((o) => !value.includes(o)).map((o) => (
              <SelectItem key={o} value={o} className="font-mono text-xs">{o}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <p className="text-xs text-muted-foreground">
        <code>iface(name)</code> and <code>vips(name)</code> resolve at render time, so they survive an interface being renumbered.
      </p>
    </div>
  )
}

function ForwardRows({ forwards, onChange }: {
  forwards: DnsForward[]
  onChange: (f: DnsForward[]) => void
}) {
  const set = (i: number, next: DnsForward) => onChange(forwards.map((f, j) => (j === i ? next : f)))

  return (
    <div className="space-y-2">
      {forwards.map((f, i) => (
        <div key={i} className="space-y-2 rounded-md bg-muted/30 p-2">
          <div className="flex items-center gap-2">
            <Input
              className="h-8 flex-1 font-mono text-xs"
              placeholder="domain"
              value={f.domain}
              onChange={(e) => set(i, { ...f, domain: e.target.value })}
            />
            <label className="flex items-center gap-1 text-xs text-muted-foreground">
              <Switch checked={!!f.dnssec} onCheckedChange={(v) => set(i, { ...f, dnssec: v || undefined })} />
              signed
            </label>
            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => onChange(forwards.filter((_, j) => j !== i))}>
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
          <div>
            <Label className="text-xs">Recursors</Label>
            <TagInput
              values={f.servers ?? []}
              onChange={(v) => set(i, { ...f, servers: v.length ? v : undefined })}
              placeholder="server IP"
            />
          </div>
        </div>
      ))}
      <Button variant="outline" size="sm" onClick={() => onChange([...forwards, { domain: '' }])}>
        <Plus className="mr-1 h-3 w-3" /> Forward a domain
      </Button>
    </div>
  )
}

function parseRecordLines(text: string): { records: DnsRecord[]; errors: string[] } {
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

function RecordRows({ records, onChange, zoneName, zones, onZonesChange }: {
  records: DnsRecord[]
  onChange: (r: DnsRecord[]) => void
  zoneName?: string
  zones?: DnsZone[]
  onZonesChange?: (z: DnsZone[]) => void
}) {
  const [filter, setFilter] = useState('')
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [pasteOpen, setPasteOpen] = useState(false)
  const [pasteText, setPasteText] = useState('')

  const set = (i: number, next: DnsRecord) => onChange(records.map((r, j) => (j === i ? next : r)))

  const reverseEnabled = !!(zoneName && zones && onZonesChange)

  const reverseInfoFor = (r: DnsRecord) => {
    if (!reverseEnabled || (r.type !== 'A' && r.type !== 'AAAA') || !r.value.trim()) return null

    const target = reverseTarget(r.value, zones!)
    if (!target) return null

    const value = ownerFqdn(r.name, zoneName!)
    return { ...target, value, checked: ptrExists(zones![target.index], target.owner, value) }
  }

  const toggleReverse = (r: DnsRecord, on: boolean) => {
    const target = reverseTarget(r.value, zones!)
    if (!target) return

    const value = ownerFqdn(r.name, zoneName!)
    const existing = zones![target.index].records ?? []
    const next = on
      ? [...existing, { name: target.owner, type: 'PTR', value }]
      : existing.filter((x) => !(x.type === 'PTR' && x.name.trim() === target.owner && fqdn(x.value) === fqdn(value)))
    if (on && ptrExists(zones![target.index], target.owner, value)) return

    onZonesChange!(zones!.map((z, i) => (i === target.index ? { ...z, records: next.length ? next : undefined } : z)))
  }

  const q = filter.trim().toLowerCase()
  const shown = records
    .map((r, i) => ({ r, i }))
    .filter(({ r }) => !q || `${r.name} ${r.type} ${r.value}`.toLowerCase().includes(q))

  const groups: { items: { r: DnsRecord; i: number }[] }[] = []
  const groupOf = new Map<string, number>()
  for (const item of shown) {
    const key = item.r.name.trim().toLowerCase()
    let g = groupOf.get(key)
    if (g === undefined) {
      g = groups.length
      groupOf.set(key, g)
      groups.push({ items: [] })
    }

    groups[g].items.push(item)
  }

  const toggle = (i: number) => setSelected((prev) => {
    const next = new Set(prev)
    if (next.has(i)) {
      next.delete(i)
    } else {
      next.add(i)
    }
    return next
  })

  const allShownSelected = shown.length > 0 && shown.every(({ i }) => selected.has(i))
  const toggleAll = () => setSelected((prev) => {
    if (allShownSelected) {
      const next = new Set(prev)
      shown.forEach(({ i }) => next.delete(i))
      return next
    }
    return new Set([...prev, ...shown.map(({ i }) => i)])
  })

  const groupSelected = (items: { i: number }[]) => items.length > 0 && items.every(({ i }) => selected.has(i))
  const toggleGroup = (items: { i: number }[]) => setSelected((prev) => {
    const next = new Set(prev)
    if (groupSelected(items)) {
      items.forEach(({ i }) => next.delete(i))
    } else {
      items.forEach(({ i }) => next.add(i))
    }
    return next
  })

  const renameGroup = (items: { i: number }[], name: string) => {
    const idx = new Set(items.map(({ i }) => i))
    onChange(records.map((r, j) => (idx.has(j) ? { ...r, name } : r)))
  }

  const deleteSelected = () => {
    onChange(records.filter((_, i) => !selected.has(i)))
    setSelected(new Set())
  }

  const parsed = parseRecordLines(pasteText)
  const append = () => {
    if (parsed.records.length === 0) return
    onChange([...records, ...parsed.records])
    setPasteText('')
    setPasteOpen(false)
  }

  return (
    <div className="space-y-2">
      {records.length > 0 && (
        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            className="h-4 w-4 shrink-0 accent-primary"
            checked={allShownSelected}
            onChange={toggleAll}
            title="Select all"
          />
          <div className="relative flex-1">
            <Search className="pointer-events-none absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="h-8 pl-7 font-mono text-xs"
              placeholder={`Filter ${records.length} records`}
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
            />
          </div>
          {selected.size > 0 && (
            <Button variant="destructive" size="sm" onClick={deleteSelected}>
              <Trash2 className="mr-1 h-3 w-3" /> Delete {selected.size}
            </Button>
          )}
        </div>
      )}

      {groups.map((g) => {
        const name = g.items[0].r.name
        return (
          <div key={g.items[0].i} className="space-y-1.5 rounded-md border border-border/60 p-2">
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                className="h-4 w-4 shrink-0 accent-primary"
                checked={groupSelected(g.items)}
                onChange={() => toggleGroup(g.items)}
              />
              <Input
                className="h-8 w-40 font-mono text-xs"
                placeholder="@"
                value={name}
                onChange={(e) => renameGroup(g.items, e.target.value)}
              />
              <span className="text-xs text-muted-foreground">{g.items.length} record{g.items.length > 1 ? 's' : ''}</span>
              <div className="flex-1" />
              <Button
                variant="ghost"
                size="sm"
                className="h-7 gap-1 text-xs text-muted-foreground"
                onClick={() => onChange([...records, { name, type: 'A', value: '' }])}
              >
                <Plus className="h-3 w-3" /> Add type
              </Button>
            </div>

            {g.items.map(({ r, i }) => {
              const rev = reverseInfoFor(r)
              return (
                <div key={i} className="flex items-center gap-2 pl-6">
                  <input
                    type="checkbox"
                    className="h-4 w-4 shrink-0 accent-primary"
                    checked={selected.has(i)}
                    onChange={() => toggle(i)}
                  />
                  <Select value={r.type || 'A'} onValueChange={(v) => set(i, { ...r, type: v })}>
                    <SelectTrigger className="h-8 w-24 text-xs"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {RECORD_TYPES.map((t) => <SelectItem key={t} value={t} className="font-mono text-xs">{t}</SelectItem>)}
                    </SelectContent>
                  </Select>
                  <Input
                    className="h-8 flex-1 font-mono text-xs"
                    placeholder="value"
                    value={r.value}
                    onChange={(e) => set(i, { ...r, value: e.target.value })}
                  />
                  {(r.type === 'MX' || r.type === 'SRV') && (
                    <Input
                      className="h-8 w-20 font-mono text-xs"
                      placeholder="prio"
                      value={r.priority ?? ''}
                      onChange={(e) => set(i, { ...r, priority: e.target.value ? Number(e.target.value) : undefined })}
                    />
                  )}
                  {rev && (
                    <label className="flex shrink-0 items-center gap-1 text-xs text-muted-foreground" title={`Create the reverse (PTR) record in ${rev.zoneName}`}>
                      <input
                        type="checkbox"
                        className="h-4 w-4 accent-primary"
                        checked={rev.checked}
                        onChange={(e) => toggleReverse(r, e.target.checked)}
                      />
                      PTR
                    </label>
                  )}
                  <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => { onChange(records.filter((_, j) => j !== i)); setSelected(new Set()) }}>
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              )
            })}
          </div>
        )
      })}

      {q && shown.length === 0 && (
        <p className="text-xs text-muted-foreground">No records match "{filter}".</p>
      )}

      <div className="flex gap-2">
        <Button variant="outline" size="sm" onClick={() => onChange([...records, { name: '@', type: 'A', value: '' }])}>
          <Plus className="mr-1 h-3 w-3" /> Add record
        </Button>
        <Button variant="outline" size="sm" onClick={() => setPasteOpen((v) => !v)}>
          Paste records
        </Button>
      </div>

      {pasteOpen && (
        <div className="space-y-2 rounded-md bg-muted/30 p-2">
          <Textarea
            className="min-h-[96px] font-mono text-xs"
            placeholder={'ns1    A     100.64.0.2\nwww    CNAME @\nmail   MX    10 mail.example.net.'}
            value={pasteText}
            onChange={(e) => setPasteText(e.target.value)}
          />
          <div className="flex items-center gap-2">
            <Button size="sm" onClick={append} disabled={parsed.records.length === 0}>
              Append {parsed.records.length || ''}
            </Button>
            <p className="text-xs text-muted-foreground">
              One record per line: <code>name [ttl] type value</code>.
              {parsed.errors.length > 0 && (
                <span className="text-amber-600 dark:text-amber-500"> {parsed.errors.length} line(s) unparsed.</span>
              )}
            </p>
          </div>
        </div>
      )}
    </div>
  )
}

export function ZoneEditor({ zone, onChange, onRemove, zones, onZonesChange }: {
  zone: DnsZone
  onChange: (z: DnsZone) => void
  onRemove: () => void
  zones?: DnsZone[]
  onZonesChange?: (z: DnsZone[]) => void
}) {
  const secondary = (zone.primaries ?? []).length > 0
  const soa = zone.soa ?? {}
  const implied = impliedZoneRecords(zone)
  const glueMissing = (zone.nameservers ?? []).some((ns) => {
    const norm = (s: string) => s.replace(/\.$/, '').toLowerCase()
    if (!norm(ns).endsWith(norm(zone.name))) return false
    return !(zone.records ?? []).some((r) => {
      const owner = r.name === '@' ? norm(zone.name) : norm(r.name).endsWith(norm(zone.name)) ? norm(r.name) : `${norm(r.name)}.${norm(zone.name)}`
      return owner === norm(ns) && (r.type === 'A' || r.type === 'AAAA')
    })
  })

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-2">
        <h2 className="truncate font-mono text-sm font-medium">{zone.name || 'new zone'}</h2>
        <Button variant="ghost" size="sm" className="gap-1.5 text-muted-foreground" onClick={onRemove}>
          <Trash2 className="h-4 w-4" /> Remove
        </Button>
      </div>

      <div className="grid gap-3 sm:grid-cols-3">
          <div>
            <Label className="text-xs">Zone</Label>
            <Input className="h-8 font-mono text-xs" value={zone.name} onChange={(e) => onChange({ ...zone, name: e.target.value })} />
          </div>
          <div>
            <Label className="text-xs">Default TTL</Label>
            <Input className="h-8 font-mono text-xs" placeholder="3600" value={zone.ttl ?? ''} onChange={(e) => onChange({ ...zone, ttl: e.target.value ? Number(e.target.value) : undefined })} />
          </div>
          <div>
            <Label className="text-xs">SOA e-mail</Label>
            <Input className="h-8 font-mono text-xs" placeholder="hostmaster@example.net" value={soa.email ?? ''} onChange={(e) => onChange({ ...zone, soa: { ...soa, email: e.target.value || undefined } })} />
          </div>
        </div>

        <div>
          <Label className="text-xs">Nameservers</Label>
          <TagInput
            values={zone.nameservers ?? []}
            onChange={(v) => onChange({ ...zone, nameservers: v.length ? v : undefined })}
            placeholder="ns1.example.net."
          />
          {glueMissing && (
            <p className="mt-1 text-xs text-amber-600 dark:text-amber-500">
              A nameserver inside this zone has no A or AAAA record. BIND rejects the zone without one.
            </p>
          )}
        </div>

        <div>
          <Label className="text-xs">Transferred in from</Label>
          <TagInput
            values={zone.primaries ?? []}
            onChange={(v) => onChange({ ...zone, primaries: v.length ? v : undefined })}
            placeholder="primary server IP (leave empty for a primary zone)"
          />
        </div>

        {!secondary && (
          <div>
            <Label className="text-xs">Records</Label>
            {implied.length > 0 && (
              <div className="mb-2 space-y-2">
                {implied.map((r, k) => (
                  <div key={k} className="flex items-center gap-2 opacity-60">
                    <Input disabled className="h-8 w-32 font-mono text-xs" value={r.name} />
                    <Input disabled className="h-8 w-24 font-mono text-xs" value={r.type} />
                    <Input disabled className="h-8 flex-1 font-mono text-xs" value={r.value} />
                    <span className="w-8 shrink-0 text-center text-[10px] uppercase text-muted-foreground">auto</span>
                  </div>
                ))}
                <p className="text-xs text-muted-foreground">
                  Generated from the fields above. Add A or AAAA glue records below for any nameserver inside this zone.
                </p>
              </div>
            )}
            <RecordRows
              records={zone.records ?? []}
              onChange={(r) => onChange({ ...zone, records: r.length ? r : undefined })}
              zoneName={zone.name}
              zones={zones}
              onZonesChange={onZonesChange}
            />
          </div>
        )}
    </div>
  )
}

export function ResolverSettings({ cfg, set, ifaces, ifaceNames, vrrpIfaces }: {
  cfg: DnsServerData
  set: DnsSet
  ifaces: Record<string, IfaceData>
  ifaceNames: string[]
  vrrpIfaces: string[]
}) {
  const listensOnVIP = (cfg.listen ?? []).some((l) => l.startsWith('vips('))

  return (
    <div className="space-y-5">
      <label className="flex items-center gap-3">
        <Switch checked={!!cfg.enabled} onCheckedChange={(v) => set('enabled', v || undefined)} />
        <div>
          <div className="text-sm font-medium">Enable DNS server</div>
          <div className="text-xs text-muted-foreground">Render and run a local BIND name server from this config.</div>
        </div>
      </label>

      <div className="space-y-5">
          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <Label className="text-xs">Mode</Label>
              <Select value={cfg.mode ?? MODE_INFERRED} onValueChange={(v) => set('mode', v === MODE_INFERRED ? undefined : v)}>
                <SelectTrigger className="h-8 text-xs"><SelectValue placeholder="inferred" /></SelectTrigger>
                <SelectContent>
                  {MODES.map((m) => (
                    <SelectItem key={m} value={m} className="text-xs">{m === MODE_INFERRED ? 'inferred from config' : m}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="mt-1 text-xs text-muted-foreground">
                <code>authoritative</code> answers only for its own zones and refuses everything else.
              </p>
            </div>
            <div>
              <Label className="text-xs">Port</Label>
              <Input className="h-8 font-mono text-xs" placeholder="53" value={cfg.port ?? ''} onChange={(e) => set('port', e.target.value ? Number(e.target.value) : undefined)} />
            </div>
          </div>

          <div>
            <Label className="text-xs">Listen addresses</Label>
            <ListenBuilder
              value={cfg.listen ?? []}
              options={listenOptions(ifaces ?? {}, vrrpIfaces)}
              onChange={(v) => set('listen', v.length ? v : undefined)}
            />
            {listensOnVIP && (
              <p className="mt-1 text-xs text-muted-foreground">
                A VRRP address is in the list, so Routier enables <code>ip_nonlocal_bind</code> so BIND can bind it while this node is BACKUP.
              </p>
            )}
          </div>

          <div>
            <Label className="text-xs">Allowed clients</Label>
            <TagInput
              values={cfg.allow_from ?? []}
              onChange={(v) => set('allow_from', v.length ? v : undefined)}
              placeholder="10.0.0.0/24"
            />
            <p className="mt-1 text-xs text-muted-foreground">Required. Also scopes the firewall rule, so the two cannot disagree.</p>
          </div>

          <div>
            <Label className="text-xs">Open the firewall on</Label>
            <div className="flex flex-wrap gap-2 pt-1">
              {ifaceNames.map((n) => {
                const on = (cfg.allow_inbound ?? []).includes(n)
                return (
                  <button
                    key={n}
                    type="button"
                    onClick={() => set('allow_inbound', on
                      ? (cfg.allow_inbound ?? []).filter((x) => x !== n)
                      : [...(cfg.allow_inbound ?? []), n])}
                    className={`rounded-md border px-2 py-1 font-mono text-xs ${on ? 'border-primary bg-primary/10' : 'border-border'}`}
                  >
                    {n}
                  </button>
                )
              })}
            </div>
          </div>

          <div>
            <Label className="text-xs">Upstreams</Label>
            <TagInput
              values={cfg.upstreams ?? []}
              onChange={(v) => set('upstreams', v.length ? v : undefined)}
              placeholder="1.1.1.1"
            />
          </div>

          <div>
            <Label className="text-xs">Forwarding zones</Label>
            <ForwardRows forwards={cfg.forward ?? []} onChange={(f) => set('forward', f.length ? f : undefined)} />
          </div>

          <div className="flex flex-col gap-3 sm:flex-row sm:gap-8">
            <label className="flex items-center gap-3">
              <Switch checked={!!cfg.dnssec} onCheckedChange={(v) => set('dnssec', v || undefined)} />
              <div>
                <div className="text-sm font-medium">Validate DNSSEC</div>
                <div className="text-xs text-muted-foreground">Internal domains are exempted automatically.</div>
              </div>
            </label>
            <label className="flex items-center gap-3">
              <Switch checked={!!cfg.log_queries} onCheckedChange={(v) => set('log_queries', v || undefined)} />
              <div>
                <div className="text-sm font-medium">Log queries</div>
                <div className="text-xs text-muted-foreground">Needed for the live query stream.</div>
              </div>
            </label>
          </div>
      </div>
    </div>
  )
}

export function CacheSettings({ cfg, set }: { cfg: DnsServerData; set: DnsSet }) {
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
