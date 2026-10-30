import { plural } from '@/lib/utils'
import { EntityRow } from '@/components/ui/EntityRow'
import { Badge, Button, Input, Label, StatusChip } from 'cheval-ui'
import {
  Plus, Trash2
} from 'lucide-react'

export interface HTTPCheck {
  verb?: string
  url?: string
  expected_code?: number
  headers?: Record<string, string>
  body?: string
  timeout?: number
}

export interface DNSCheck {
  resolver?: string
  type?: string
  query?: string
  expected?: string
  timeout?: number
}

export interface Endpoint {
  ip: string
  interface?: string
  distance?: number
  http_check?: HTTPCheck
  dns_check?: DNSCheck
}

export interface AnycastService {
  name: string
  active?: boolean
  anycast_ips?: string[]
  endpoints?: Endpoint[]
}

export interface AnycastConfig {
  services?: AnycastService[]
}

export interface RoutingSection {
  anycast?: AnycastConfig
  [key: string]: unknown
}

export const HTTP_VERBS = ['GET', 'POST', 'PUT', 'HEAD', 'DELETE', 'PATCH']

export function emptyService(): AnycastService {
  return { name: '', active: true, anycast_ips: [], endpoints: [] }
}

export function emptyEndpoint(): Endpoint {
  return { ip: '', distance: 0 }
}

export type CheckMode = 'none' | 'http' | 'dns'

export interface HeadersEditorProps {
  headers: Record<string, string>
  onChange: (h: Record<string, string>) => void
}

export function HeadersEditor({ headers, onChange }: HeadersEditorProps) {
  const entries = Object.entries(headers)

  const set = (oldKey: string, newKey: string, val: string) => {
    const next: Record<string, string> = {}
    for (const [k, v] of Object.entries(headers)) {
      next[k === oldKey ? newKey : k] = k === oldKey ? val : v
    }
    onChange(next)
  }

  const remove = (key: string) => {
    const next = { ...headers }
    delete next[key]
    onChange(next)
  }

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label>Headers</Label>
        <Button type="button" variant="outline" size="sm" onClick={() => onChange({ ...headers, '': '' })} className="gap-1.5">
          <Plus className="h-4 w-4" />Add Header
        </Button>
      </div>
      {entries.length === 0 && (
        <p className="text-sm text-muted-foreground">No custom headers</p>
      )}
      {entries.map(([k, v], idx) => (
        <div key={idx} className="flex flex-wrap items-center gap-1.5">
          <Input value={k} onChange={(e) => set(k, e.target.value, v)} placeholder="Header-Name" aria-label="Header Name" className="min-w-0 flex-1 basis-48 font-mono" />
          <Input value={v} onChange={(e) => set(k, k, e.target.value)} placeholder="value" aria-label="Header Value" className="min-w-0 flex-[2] basis-64 font-mono" />
          <Button type="button" variant="ghost" size="icon" onClick={() => remove(k)} aria-label="Remove Header" title="Remove Header" className="shrink-0 text-muted-foreground hover:text-destructive">
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      ))}
    </div>
  )
}

export interface EndpointEditorProps {
  endpoint: Endpoint
  onChange: (e: Endpoint) => void
  onDelete: () => void
}

export interface ServiceFormProps {
  service: AnycastService
  onChange: (s: AnycastService) => void
  onAdd?: () => void
  onDone: () => void
}

export interface ServiceRowProps {
  service: AnycastService
  onEdit: () => void
  onDelete: () => void
}

export function ServiceRow({ service, onEdit, onDelete }: ServiceRowProps) {
  return (
    <EntityRow title={service.name} onOpen={onEdit} onRemove={onDelete} removeLabel="Remove Service">
      <StatusChip intent={service.active ? 'success' : 'neutral'} label={service.active ? 'Active' : 'Inactive'} />
      {(service.anycast_ips ?? []).map((ip) => <Badge key={ip} variant="outline" className="font-mono">{ip}</Badge>)}
      {(service.endpoints ?? []).length > 0 && <Badge variant="outline">{plural((service.endpoints ?? []).length, 'endpoint')}</Badge>}
    </EntityRow>
  )
}
