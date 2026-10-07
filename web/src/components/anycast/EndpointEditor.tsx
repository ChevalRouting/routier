import { CheckMode, DNSCheck, Endpoint, EndpointEditorProps, HTTPCheck, HTTP_VERBS, HeadersEditor } from '@/components/anycast/shared'
import { Badge, Button, Input, Label, Segmented, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import {
  ChevronDown, ChevronRight,
  Trash2
} from 'lucide-react'
import { useState } from 'react'

export function EndpointEditor({ endpoint, onChange, onDelete }: EndpointEditorProps) {
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
    <div className="rounded-md bg-card overflow-hidden shadow-[var(--card-shadow)]">
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
        <div className="m-2 mt-0 rounded-md bg-muted/30 px-3 py-3 space-y-4">
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
