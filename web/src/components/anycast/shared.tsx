import { Badge, Button, Input, Label } from 'cheval-ui'
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
        <Label className="text-xs">Headers</Label>
        <Button type="button" variant="outline" size="sm" onClick={() => onChange({ ...headers, '': '' })} className="h-6 gap-1 text-xs px-2">
          <Plus className="h-3 w-3" />Add
        </Button>
      </div>
      {entries.length === 0 && (
        <p className="text-xs text-muted-foreground italic">No custom headers</p>
      )}
      {entries.map(([k, v], idx) => (
        <div key={idx} className="flex gap-1.5 items-center">
          <Input value={k} onChange={(e) => set(k, e.target.value, v)} placeholder="Header-Name" className="font-mono text-xs h-7 flex-1" />
          <Input value={v} onChange={(e) => set(k, k, e.target.value)} placeholder="value" className="font-mono text-xs h-7 flex-1" />
          <Button type="button" variant="ghost" size="icon" onClick={() => remove(k)} className="h-7 w-7 shrink-0 hover:text-destructive">
            <Trash2 className="h-3 w-3" />
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
    <div
      onClick={onEdit}
      className="flex items-start gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors"
    >
      <span className="font-mono font-semibold text-sm w-28 shrink-0 truncate pt-0.5">
        {service.name || <span className="text-muted-foreground italic font-normal">unnamed</span>}
      </span>
      <div className="flex flex-wrap gap-1 flex-1 min-w-0">
        <Badge variant={service.active ? 'default' : 'secondary'} className="text-xs">
          {service.active ? 'active' : 'inactive'}
        </Badge>
        {(service.anycast_ips ?? []).map((ip) => (
          <Badge key={ip} variant="outline" className="text-xs font-mono">{ip}</Badge>
        ))}
        {(service.endpoints ?? []).length > 0 && (
          <Badge variant="outline" className="text-xs">
            {(service.endpoints ?? []).length} endpoint{(service.endpoints ?? []).length !== 1 ? 's' : ''}
          </Badge>
        )}
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={(e) => { e.stopPropagation(); onDelete() }}
        className="h-7 w-7 p-0 hover:text-destructive shrink-0"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}
