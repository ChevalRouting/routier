import { activeBaseUrl } from './instance'
import { getToken } from './utils'

export interface DnsServiceState {
  service: string
  running: boolean
}

export interface DnsStatus {
  version?: string
  boot_time?: string
  config_time?: string
  zones: number
  recursive_high_water: number
  running: boolean
}

export interface DnsStat {
  name: string
  section?: string
  value: number
}

export interface DnsZoneRecord {
  name: string
  type: string
  value: string
  ttl?: number
  priority?: number
}

export interface DnsZoneView {
  name: string
  records?: DnsZoneRecord[]
  serial: number
  answered: boolean
}

export interface DnsOverview {
  running: boolean
  status?: DnsStatus
  listen?: string[]
  zones?: DnsZoneView[]
  stats?: DnsStat[]
}

export interface DnsSummary {
  mode: string
  recurses: boolean
  port: number
  allow_from?: string[]
  upstreams?: string[]
  overview?: DnsOverview
}

export interface DnsStats {
  services?: DnsServiceState[]
  status?: DnsStatus
  stats?: DnsStat[]
}

export interface DnsAnswer {
  name: string
  ttl: number
  class: string
  type: string
  data: string
}

export interface DnsQueryResult {
  name: string
  type: string
  rcode: string
  flags?: string[]
  answers?: DnsAnswer[]
  server?: string
  query_ms: number
}

interface Envelope<T> {
  result?: T
  error?: string
  status?: string
  code?: number
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(activeBaseUrl() + path, {
    ...init,
    headers: {
      Authorization: `Bearer ${getToken() ?? ''}`,
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })

  const body = (await res.json().catch(() => ({}))) as Envelope<T>
  if (!res.ok) {
    throw new Error(body.error || `${res.status} ${res.statusText}`)
  }

  return body.result as T
}

export function dnsSummary(): Promise<DnsSummary> {
  return request<DnsSummary>('/api/dns')
}

export function dnsStats(): Promise<DnsStats> {
  return request<DnsStats>('/api/dns/stats')
}

export function dnsZones(): Promise<DnsZoneView[]> {
  return request<DnsZoneView[]>('/api/dns/zones')
}

export function dnsReloadZone(name: string): Promise<string> {
  return request<string>(`/api/dns/zones/${encodeURIComponent(name)}/reload`, { method: 'POST' })
}

export function dnsFlush(name?: string): Promise<string> {
  return request<string>('/api/dns/cache/flush', {
    method: 'POST',
    body: JSON.stringify({ name: name ?? '' }),
  })
}

export function dnsRestart(): Promise<string> {
  return request<string>('/api/dns/restart', { method: 'POST' })
}

export function dnsQuery(name: string, type?: string): Promise<DnsQueryResult> {
  return request<DnsQueryResult>('/api/dns/query', {
    method: 'POST',
    body: JSON.stringify({ name, type: type ?? '' }),
  })
}

export interface DnsDdnsZoneView {
  name: string
  records?: DnsZoneRecord[]
  error?: string
}

export function dnsDdns(): Promise<DnsDdnsZoneView[]> {
  return request<DnsDdnsZoneView[]>('/api/dns/ddns')
}

export function dnsDdnsDelete(zone: string, rec: { name: string; type: string; value?: string }): Promise<string> {
  return request<string>(`/api/dns/ddns/${encodeURIComponent(zone)}/records`, {
    method: 'DELETE',
    body: JSON.stringify(rec),
  })
}
