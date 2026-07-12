import { useState, useEffect } from 'react'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { useDataRefresh } from '@/lib/dataVersion'
import SaveButton from '@/components/SaveButton'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Sheet } from '@/components/ui/sheet'
import TagInput from '@/components/TagInput'
import {
  Plus, Trash2, Radio, ChevronDown, ChevronRight,
} from 'lucide-react'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import { PageHeader } from '@/components/PageHeader'
import { EmptyState } from '@/components/EmptyState'
import { Pagination, usePagination } from '@/components/Pagination'
import { PreferencesGroup, PreferencesColumns, EntryRow, SwitchRow } from '@/components/Preferences'
import { Segmented } from '@/components/ui/segmented'
import { ReloadButton } from '@/components/ReloadButton'
import { checkIPOrCIDR } from '@/lib/validate'
import { Spinner } from '@/components/Spinner'

interface HTTPCheck {
  verb?: string
  url?: string
  expected_code?: number
  headers?: Record<string, string>
  body?: string
  timeout?: number
}

interface DNSCheck {
  resolver?: string
  type?: string
  query?: string
  expected?: string
  timeout?: number
}

interface Endpoint {
  ip: string
  interface?: string
  distance?: number
  http_check?: HTTPCheck
  dns_check?: DNSCheck
}

interface AnycastService {
  name: string
  active?: boolean
  anycast_ips?: string[]
  endpoints?: Endpoint[]
}

interface AnycastConfig {
  services?: AnycastService[]
}

interface RoutingSection {
  anycast?: AnycastConfig
  [key: string]: unknown
}

const HTTP_VERBS = ['GET', 'POST', 'PUT', 'HEAD', 'DELETE', 'PATCH']

function emptyService(): AnycastService {
  return { name: '', active: true, anycast_ips: [], endpoints: [] }
}

function emptyEndpoint(): Endpoint {
  return { ip: '', distance: 0 }
}

type CheckMode = 'none' | 'http' | 'dns'

interface HeadersEditorProps {
  headers: Record<string, string>
  onChange: (h: Record<string, string>) => void
}

function HeadersEditor({ headers, onChange }: HeadersEditorProps) {
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

interface EndpointEditorProps {
  endpoint: Endpoint
  onChange: (e: Endpoint) => void
  onDelete: () => void
}

function EndpointEditor({ endpoint, onChange, onDelete }: EndpointEditorProps) {
  const [expanded, setExpanded] = useState(!endpoint.ip)
  const checkMode: CheckMode = endpoint.http_check ? 'http' : endpoint.dns_check ? 'dns' : 'none'

  const set = <K extends keyof Endpoint>(key: K, val: Endpoint[K]) =>
    onChange({ ...endpoint, [key]: val })

  const setCheckMode = (mode: CheckMode) => {
    const next = { ...endpoint }
    delete next.http_check
    delete next.dns_check
    if (mode === 'http') next.http_check = { verb: 'GET', expected_code: 200 }
    if (mode === 'dns') next.dns_check = {}
    onChange(next)
  }

  const setHTTP = (patch: Partial<HTTPCheck>) =>
    onChange({ ...endpoint, http_check: { ...endpoint.http_check, ...patch } })

  const setDNS = (patch: Partial<DNSCheck>) =>
    onChange({ ...endpoint, dns_check: { ...endpoint.dns_check, ...patch } })

  return (
    <div className="border rounded-md overflow-hidden">
      <div className="flex items-center gap-2 px-3 py-2 bg-muted/30">
        <button type="button" onClick={() => setExpanded(!expanded)} className="text-muted-foreground hover:text-foreground">
          {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </button>
        <span className="font-mono text-sm flex-1">{endpoint.ip || <span className="text-muted-foreground italic">New endpoint</span>}</span>
        {endpoint.distance !== undefined && endpoint.distance > 0 && (
          <span className="text-xs text-muted-foreground">distance {endpoint.distance}</span>
        )}
        <Badge variant="outline" className="text-xs">{checkMode === 'none' ? 'no check' : checkMode.toUpperCase()}</Badge>
        <Button variant="ghost" size="sm" onClick={onDelete} className="h-7 w-7 p-0 hover:text-destructive shrink-0">
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>

      {expanded && (
        <div className="px-3 py-3 space-y-4 border-t bg-background">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-xs">Endpoint IP</Label>
              <Input value={endpoint.ip} onChange={(e) => set('ip', e.target.value)} placeholder="10.0.0.1" className="font-mono text-sm" />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Interface <span className="text-muted-foreground">(optional)</span></Label>
              <Input value={endpoint.interface ?? ''} onChange={(e) => set('interface', e.target.value || undefined)} placeholder="eth0" className="font-mono text-sm" />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Distance <span className="text-muted-foreground">(0 = default)</span></Label>
              <Input
                value={endpoint.distance ?? 0}
                onChange={(e) => set('distance', Number(e.target.value))}
                placeholder="0"
                className="font-mono text-sm"
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label className="text-xs">Health check</Label>
            <Segmented
              value={checkMode}
              onChange={setCheckMode}
              options={[
                { value: 'none', label: 'None' },
                { value: 'http', label: 'HTTP' },
                { value: 'dns', label: 'DNS' },
              ]}
            />
          </div>

          {checkMode === 'http' && endpoint.http_check && (
            <div className="space-y-3 pl-3 border-l-2 border-muted">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label className="text-xs">Verb</Label>
                  <Select value={endpoint.http_check.verb ?? 'GET'} onValueChange={(v) => setHTTP({ verb: v })}>
                    <SelectTrigger className="h-8 text-sm font-mono"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {HTTP_VERBS.map((v) => <SelectItem key={v} value={v}>{v}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs">Expected code</Label>
                  <Input value={endpoint.http_check.expected_code ?? 200} onChange={(e) => setHTTP({ expected_code: Number(e.target.value) })} placeholder="200" className="font-mono text-sm h-8" />
                </div>
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs">URL</Label>
                <Input value={endpoint.http_check.url ?? ''} onChange={(e) => setHTTP({ url: e.target.value })} placeholder="http://10.0.0.1/health" className="font-mono text-sm" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label className="text-xs">Body</Label>
                  <Input value={endpoint.http_check.body ?? ''} onChange={(e) => setHTTP({ body: e.target.value })} placeholder="optional" className="text-sm" />
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs">Timeout (s)</Label>
                  <Input value={endpoint.http_check.timeout ?? ''} onChange={(e) => setHTTP({ timeout: e.target.value ? Number(e.target.value) : undefined })} placeholder="5" className="font-mono text-sm" />
                </div>
              </div>
              <HeadersEditor headers={endpoint.http_check.headers ?? {}} onChange={(h) => setHTTP({ headers: Object.keys(h).length ? h : undefined })} />
            </div>
          )}

          {checkMode === 'dns' && endpoint.dns_check && (
            <div className="space-y-3 pl-3 border-l-2 border-muted">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label className="text-xs">Resolver IP</Label>
                  <Input value={endpoint.dns_check.resolver ?? ''} onChange={(e) => setDNS({ resolver: e.target.value })} placeholder="8.8.8.8" className="font-mono text-sm" />
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs">Record type</Label>
                  <Input value={endpoint.dns_check.type ?? ''} onChange={(e) => setDNS({ type: e.target.value })} placeholder="A" className="font-mono text-sm" />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label className="text-xs">Query</Label>
                  <Input value={endpoint.dns_check.query ?? ''} onChange={(e) => setDNS({ query: e.target.value })} placeholder="example.com" className="font-mono text-sm" />
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs">Expected answer</Label>
                  <Input value={endpoint.dns_check.expected ?? ''} onChange={(e) => setDNS({ expected: e.target.value })} placeholder="1.2.3.4" className="font-mono text-sm" />
                </div>
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs">Timeout (s)</Label>
                <Input value={endpoint.dns_check.timeout ?? ''} onChange={(e) => setDNS({ timeout: e.target.value ? Number(e.target.value) : undefined })} placeholder="5" className="font-mono text-sm w-28" />
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

interface ServiceFormProps {
  service: AnycastService
  onChange: (s: AnycastService) => void
  onAdd?: () => void
  onDone: () => void
}

function ServiceForm({ service, onChange, onAdd, onDone }: ServiceFormProps) {
  const set = <K extends keyof AnycastService>(key: K, val: AnycastService[K]) =>
    onChange({ ...service, [key]: val })

  const updateEndpoint = (idx: number, ep: Endpoint) => {
    const next = [...(service.endpoints ?? [])]
    next[idx] = ep
    set('endpoints', next)
  }

  const deleteEndpoint = (idx: number) =>
    set('endpoints', (service.endpoints ?? []).filter((_, i) => i !== idx))

  const addEndpoint = () =>
    set('endpoints', [...(service.endpoints ?? []), emptyEndpoint()])

  return (
    <>
      <PreferencesGroup>
        <EntryRow
          title="Service Name"
          autoFocus
          value={service.name}
          onChange={(e) => set('name', e.target.value)}
          placeholder="my-service"
          className="font-mono"
        />
        <SwitchRow
          title="Active"
          checked={service.active ?? false}
          onCheckedChange={(v) => set('active', v)}
        />
      </PreferencesGroup>

      <PreferencesGroup title="Anycast IPs">
        <div className="px-4 py-3">
          <TagInput
            values={service.anycast_ips ?? []}
            onChange={(v) => set('anycast_ips', v)}
            placeholder="192.0.2.1/32"
            validate={checkIPOrCIDR}
            mono
          />
        </div>
      </PreferencesGroup>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-sm font-semibold">
            Endpoints
            <span className="ml-1 text-xs font-normal text-muted-foreground">
              ({(service.endpoints ?? []).length})
            </span>
          </Label>
          <Button type="button" variant="outline" size="sm" onClick={addEndpoint} className="h-7 gap-1 text-xs">
            <Plus className="h-3 w-3" />Add Endpoint
          </Button>
        </div>
        {(service.endpoints ?? []).length === 0 && (
          <p className="text-sm text-muted-foreground italic">No endpoints configured.</p>
        )}
        {(service.endpoints ?? []).map((ep, idx) => (
          <EndpointEditor
            key={idx}
            endpoint={ep}
            onChange={(updated) => updateEndpoint(idx, updated)}
            onDelete={() => deleteEndpoint(idx)}
          />
        ))}
      </div>

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add Service</Button>}
      </div>
    </>
  )
}

interface ServiceRowProps {
  service: AnycastService
  onEdit: () => void
  onDelete: () => void
}

function ServiceRow({ service, onEdit, onDelete }: ServiceRowProps) {
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

export default function Anycast() {
  const { data, isLoading, reload } = useFetch<RoutingSection>(
    () => api.apiConfigSectionGet({ section: 'routing' }) as Promise<RoutingSection>
  )
  const [routing, setRouting] = useState<RoutingSection | null>(null)
  const [initialized, setInitialized] = useState(false)
  useDataRefresh(() => setInitialized(false))
  const [openIdx, setOpenIdx] = useState<number | null>(null)
  const [formDraft, setFormDraft] = useState<AnycastService | null>(null)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('routing')

  useEffect(() => {
    if (data && !initialized) {
      setRouting(data)
      setInitialized(true)
    }
  }, [data, initialized])

  const services = routing?.anycast?.services ?? []

  const updateServices = (next: AnycastService[]) => {
    setRouting({ ...routing, anycast: { services: next } })
    markDirty()
  }

  const deleteService = (idx: number) => {
    updateServices(services.filter((_, i) => i !== idx))
    if (openIdx === idx) setOpenIdx(null)
  }

  const handleAddNew = () => { setFormDraft(emptyService()); setOpenIdx(-1) }

  const handleCommitNew = () => {
    if (!formDraft) return
    updateServices([...services, formDraft])
    setOpenIdx(null); setFormDraft(null)
  }

  const handleCloseSheet = () => { setOpenIdx(null); setFormDraft(null) }

  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(services, 12)

  if (isLoading) return <Spinner />

  const isAdding = openIdx === -1
  const openSvc = openIdx !== null && openIdx >= 0 ? services[openIdx] : null
  const sheetTitle = isAdding ? 'Add Service' : (openSvc?.name || 'Service')

  return (
    <div className="space-y-6">
      <PageHeader title="Anycast" description="Manage anycast services and health-checked endpoints" action={
        <div className="flex items-center gap-2">
          <SaveButton isDirty={isDirty} saving={saving} onClick={() => routing && save(routing)} onCancel={() => { setRouting(null); setInitialized(false); reset() }} />
          <ReloadButton onClick={() => { setRouting(null); setInitialized(false); reload() }} />
        </div>
      } />

      {services.length === 0 ? (
        <EmptyState
          className="max-w-2xl mx-auto"
          icon={<Radio />}
          title="No anycast services"
          message="Define an anycast service to advertise a shared address across healthy endpoints."
          action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add service</Button>}
        />
      ) : (
        <PreferencesColumns
          title="Services"
          header={
            <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
              <Plus className="h-4 w-4" />Add
            </Button>
          }
        >
          {pageItems.map((svc, localIdx) => {
            const idx = page * pageSize + localIdx
            return (
              <ServiceRow
                key={idx}
                service={svc}
                onEdit={() => setOpenIdx(idx)}
                onDelete={() => deleteService(idx)}
              />
            )
          })}
        </PreferencesColumns>
      )}

      <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="services" />

      <Sheet
        open={openIdx !== null}
        onClose={handleCloseSheet}
        title={sheetTitle}
        className="max-w-2xl"
      >
        {openIdx !== null && (isAdding ? formDraft : openSvc) && (
          <ServiceForm
            service={isAdding ? formDraft! : openSvc!}
            onChange={isAdding ? setFormDraft : (u) => updateServices(services.map((s, i) => i === openIdx ? u : s))}
            onAdd={isAdding ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
          />
        )}
      </Sheet>
    </div>
  )
}
