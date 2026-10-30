import { CheckMode, DNSCheck, Endpoint, EndpointEditorProps, HTTPCheck, HTTP_VERBS, HeadersEditor } from '@/components/anycast/shared'
import { Button, ComboRow, EntryRow, PreferencesGroup, Segmented } from 'cheval-ui'
import { Trash2 } from 'lucide-react'
import { SelectRow } from '@/components/ui/SelectRow'

export function EndpointEditor({ endpoint, onChange, onDelete }: EndpointEditorProps) {
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

  const num = (value: string) => (value ? Number(value) : undefined)

  return (
    <PreferencesGroup
      title={endpoint.ip || 'New Endpoint'}
      header={(
        <Button variant="ghost" size="icon" onClick={onDelete} aria-label="Remove Endpoint" title="Remove Endpoint" className="text-muted-foreground hover:text-destructive">
          <Trash2 className="h-4 w-4" />
        </Button>
      )}
    >
      <EntryRow title="Endpoint IP" value={endpoint.ip} onChange={(e) => set('ip', e.target.value)} placeholder="10.0.0.1" className="font-mono" />
      <EntryRow title="Interface" value={endpoint.interface ?? ''} onChange={(e) => set('interface', e.target.value || undefined)} placeholder="eth0" className="font-mono" />
      <EntryRow title="Distance" value={endpoint.distance ?? 0} onChange={(e) => set('distance', Number(e.target.value))} placeholder="0" className="font-mono" help="0 uses the default" />
      <ComboRow title="Health Check">
        <Segmented
          value={checkMode}
          onChange={setCheckMode}
          options={[
            { value: 'none', label: 'None' },
            { value: 'http', label: 'HTTP' },
            { value: 'dns', label: 'DNS' },
          ]}
        />
      </ComboRow>
      {checkMode === 'http' && endpoint.http_check && (
        <>
          <SelectRow title="Verb" value={endpoint.http_check.verb ?? 'GET'} onChange={(v) => setHTTP({ verb: v ?? 'GET' })} options={HTTP_VERBS} />
          <EntryRow title="URL" value={endpoint.http_check.url ?? ''} onChange={(e) => setHTTP({ url: e.target.value })} placeholder="http://10.0.0.1/health" className="font-mono" />
          <EntryRow title="Expected Code" value={endpoint.http_check.expected_code ?? 200} onChange={(e) => setHTTP({ expected_code: Number(e.target.value) })} placeholder="200" className="font-mono" />
          <EntryRow title="Body" value={endpoint.http_check.body ?? ''} onChange={(e) => setHTTP({ body: e.target.value })} />
          <EntryRow title="Timeout (s)" value={endpoint.http_check.timeout ?? ''} onChange={(e) => setHTTP({ timeout: num(e.target.value) })} placeholder="5" className="font-mono" />
          <div className="px-4 py-3">
            <HeadersEditor headers={endpoint.http_check.headers ?? {}} onChange={(h) => setHTTP({ headers: Object.keys(h).length ? h : undefined })} />
          </div>
        </>
      )}
      {checkMode === 'dns' && endpoint.dns_check && (
        <>
          <EntryRow title="Resolver" value={endpoint.dns_check.resolver ?? ''} onChange={(e) => setDNS({ resolver: e.target.value })} placeholder="8.8.8.8" className="font-mono" />
          <EntryRow title="Record Type" value={endpoint.dns_check.type ?? ''} onChange={(e) => setDNS({ type: e.target.value })} placeholder="A" className="font-mono" />
          <EntryRow title="Query" value={endpoint.dns_check.query ?? ''} onChange={(e) => setDNS({ query: e.target.value })} placeholder="example.com" className="font-mono" />
          <EntryRow title="Expected Answer" value={endpoint.dns_check.expected ?? ''} onChange={(e) => setDNS({ expected: e.target.value })} placeholder="1.2.3.4" className="font-mono" />
          <EntryRow title="Timeout (s)" value={endpoint.dns_check.timeout ?? ''} onChange={(e) => setDNS({ timeout: num(e.target.value) })} placeholder="5" className="font-mono" />
        </>
      )}
    </PreferencesGroup>
  )
}
