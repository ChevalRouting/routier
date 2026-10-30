import { DNSConfig, OnStateChange } from '@/components/system-config/shared'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { isIP } from '@/lib/validate'
import { Button, ComboRow, Input, PreferencesGroup, Row, Spinner } from 'cheval-ui'
import { Plus, X } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'

type DNSTabShape = { onStateChange: OnStateChange }

export function DNSTab({ onStateChange }: DNSTabShape) {
  const { data, isLoading } = useFetch<DNSConfig>(() => api.apiConfigSectionGet({ section: 'dns' }) as Promise<DNSConfig>)
  const [nameservers, setNameservers] = useState<string[]>([])
  const [search, setSearch] = useState<string[]>([])
  const [newNs, setNewNs] = useState('')
  const [newSearch, setNewSearch] = useState('')
  const [initialized, setInitialized] = useState(false)
  const { isDirty, markDirty, save, saving } = usePageSave('dns')
  useDataRefresh(() => setInitialized(false))

  useEffect(() => {
    if (data && !initialized) { setNameservers(data.nameservers ?? []); setSearch(data.search ?? []); setInitialized(true) }
  }, [data, initialized])

  const handleSave = useCallback(() => save({ nameservers, search }), [save, nameservers, search])
  useEffect(() => { onStateChange({ isDirty, saving, save: handleSave }) }, [isDirty, saving, handleSave, onStateChange])

  const addNs = () => {
    const t = newNs.trim()
    if (!t || !isIP(t) || nameservers.includes(t)) return
    setNameservers([...nameservers, t]); setNewNs(''); markDirty()
  }
  const addSearch = () => {
    const t = newSearch.trim()
    if (!t || search.includes(t)) return
    setSearch([...search, t]); setNewSearch(''); markDirty()
  }

  if (isLoading) return <Spinner />

  return (
    <div className="max-w-3xl space-y-6">
      <PreferencesGroup title="Nameservers">
        {nameservers.map((ns) => (
          <Row key={ns} title={<span className="font-mono">{ns}</span>}>
            <Button variant="ghost" size="icon" onClick={() => { setNameservers(nameservers.filter((n) => n !== ns)); markDirty() }} className="text-muted-foreground hover:text-destructive" aria-label={`Remove ${ns}`} title="Remove">
              <X className="h-4 w-4" />
            </Button>
          </Row>
        ))}
        <ComboRow title="Add Nameserver">
<div className="flex w-full items-center gap-2">
          <Input value={newNs} onChange={(e) => setNewNs(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addNs()} placeholder="8.8.8.8" className={`min-w-0 flex-1 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50 ${newNs.trim() && !isIP(newNs.trim()) ? 'text-danger' : ''}`} />
          <Button variant="ghost" size="sm" onClick={addNs} disabled={!!newNs.trim() && !isIP(newNs.trim())} className="shrink-0 gap-1"><Plus className="h-4 w-4" />Add</Button>
        </div>
</ComboRow>
      </PreferencesGroup>

      <PreferencesGroup title="Search Domains">
        {search.map((s) => (
          <Row key={s} title={<span className="font-mono">{s}</span>}>
            <Button variant="ghost" size="icon" onClick={() => { setSearch(search.filter((n) => n !== s)); markDirty() }} className="text-muted-foreground hover:text-destructive" aria-label={`Remove ${s}`} title="Remove">
              <X className="h-4 w-4" />
            </Button>
          </Row>
        ))}
        <ComboRow title="Add Search Domain">
<div className="flex w-full items-center gap-2">
          <Input value={newSearch} onChange={(e) => setNewSearch(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addSearch()} placeholder="example.com" className="min-w-0 flex-1 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50" />
          <Button variant="ghost" size="sm" onClick={addSearch} className="shrink-0 gap-1"><Plus className="h-4 w-4" />Add</Button>
        </div>
</ComboRow>
      </PreferencesGroup>
    </div>
  )
}
