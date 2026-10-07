import type { TypesApplyLogRecord as ApplyLogRecord, ConfigMonitoringConfig as MonitoringConfig } from '@/api'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { Badge, Button, EntryRow, PreferencesGroup, SectionLabel, Spinner } from 'cheval-ui'
import { ExternalLink, FileJson } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'

type HostnameTabShape = { onStateChange: OnStateChange }

type INTERVALKEYSShape = { key: keyof NonNullable<MonitoringConfig['collection']>; label: string }

type SYSTEMTABSShape = { key: SystemTab; label: string }

export interface TabSaveState {
  isDirty: boolean
  saving: boolean
  save: () => void
}

export type OnStateChange = (s: TabSaveState) => void

export function HostnameTab({ onStateChange }: HostnameTabShape) {
  const { data, isLoading } = useFetch<string>(() => api.apiConfigSectionGet({ section: 'hostname' }) as unknown as Promise<string>)
  const [hostname, setHostname] = useState('')
  const [initialized, setInitialized] = useState(false)
  const { isDirty, markDirty, save, saving } = usePageSave('hostname')
  useDataRefresh(() => setInitialized(false))

  useEffect(() => {
    if (data !== null && data !== undefined && !initialized) {
      setHostname(data); setInitialized(true)
    }
  }, [data, initialized])

  const handleSave = useCallback(() => save(hostname), [save, hostname])
  useEffect(() => { onStateChange({ isDirty, saving, save: handleSave }) }, [isDirty, saving, handleSave, onStateChange])

  if (isLoading) return <Spinner />

  return (
    <div className="max-w-xl">
      <PreferencesGroup description="Written to /etc/hostname and applied immediately on config apply.">
        <EntryRow
          id="hostname"
          title="Hostname"
          value={hostname}
          onChange={(e) => { setHostname(e.target.value); markDirty() }}
          placeholder="router.example.com"
          className="font-mono"
        />
      </PreferencesGroup>
    </div>
  )
}

export interface DNSConfig { nameservers?: string[]; search?: string[] }

export interface SSHConfig {
  port?: number; permit_root_login?: string; password_auth?: string; pubkey_auth?: string
  allow_tcp_forwarding?: string; x11_forwarding?: string; max_auth_tries?: number
  login_grace_time?: number; client_alive_interval?: number; client_alive_count_max?: number
  allow_users?: string[]; allow_groups?: string[]; banner?: string
}

export const YES_NO = ['yes', 'no']

export const ROOT_LOGIN_OPTS = ['yes', 'no', 'prohibit-password', 'forced-commands-only']

export const TCP_FWD_OPTS = ['yes', 'no', 'local', 'remote']

export interface SysctlEntry { key: string; value: string }

export function parseEntries(data: unknown): SysctlEntry[] {
  if (!data || typeof data !== 'object' || Array.isArray(data)) return []
  return Object.entries(data as Record<string, string>).map(([key, value]) => ({ key, value }))
}

export function toRecord(entries: SysctlEntry[]): Record<string, string> {
  const r: Record<string, string> = {}
  for (const { key, value } of entries) { if (key.trim()) r[key.trim()] = value }
  return r
}

export function resultBadge(r: ApplyLogRecord) {
  if (r.confirmed_at) return <Badge className="bg-success/15 text-success border-success/25">confirmed</Badge>
  if (r.rolledback_at || r.result === 'rolledback') return <Badge className="bg-danger/15 text-danger border-danger/25">rolled back</Badge>
  if (r.result === 'failed') return <Badge className="bg-danger/15 text-danger border-danger/25">failed</Badge>
  if (r.result === 'applied') return <Badge className="bg-warning/15 text-warning border-warning/25">applied</Badge>
  return <Badge variant="outline">{r.result || 'running'}</Badge>
}

export function ApiDocsTab() {
  return (
    <div className="space-y-4">
      <SectionLabel>API Documentation</SectionLabel>
      <p className="text-sm text-muted-foreground">
        Interactive OpenAPI documentation for the Routier API, generated from the server.
      </p>
      <div className="flex flex-wrap gap-2">
        <Button asChild variant="outline" className="gap-2">
          <a href="/api/v1/docs/" target="_blank" rel="noopener noreferrer">
            <ExternalLink className="h-4 w-4" />Open API docs
          </a>
        </Button>
        <Button asChild variant="outline" className="gap-2">
          <a href="/api/v1/docs/openapi.json" target="_blank" rel="noopener noreferrer">
            <FileJson className="h-4 w-4" />OpenAPI spec
          </a>
        </Button>
      </div>
    </div>
  )
}

export const INTERVAL_KEYS: Array<INTERVALKEYSShape> = [
  { key: 'iface',     label: 'Interfaces' },
  { key: 'system',    label: 'CPU / RAM' },
  { key: 'bgp',       label: 'BGP' },
  { key: 'proto',     label: 'Protocols' },
  { key: 'neighbors', label: 'Neighbors' },
  { key: 'lldp',      label: 'LLDP / CDP' },
  { key: 'routes',    label: 'Route cache TTL' },
]

export type SystemTab = 'services' | 'users' | 'hostname' | 'dns' | 'ssh' | 'sysctl' | 'monitoring' | 'updates' | 'apply' | 'snapshots' | 'backup' | 'apikeys' | 'apidocs'

export const SYSTEM_TABS: SYSTEMTABSShape[] = [
  { key: 'hostname',  label: 'Hostname' },
  { key: 'dns',       label: 'DNS' },
  { key: 'ssh',       label: 'SSH' },
  { key: 'users',     label: 'Users' },
  { key: 'services',  label: 'Services' },
  { key: 'sysctl',    label: 'Sysctl' },
  { key: 'monitoring', label: 'Monitoring' },
  { key: 'updates',   label: 'Updates' },
  { key: 'apply',     label: 'Apply History' },
  { key: 'snapshots', label: 'Snapshots' },
  { key: 'backup',    label: 'Backup' },
  { key: 'apikeys',   label: 'API Keys' },
  { key: 'apidocs',   label: 'API Docs' },
]

export const CONFIG_TABS: SystemTab[] = ['hostname', 'dns', 'ssh', 'sysctl', 'monitoring']

export const EMPTY_SAVE: TabSaveState = { isDirty: false, saving: false, save: () => {} }
