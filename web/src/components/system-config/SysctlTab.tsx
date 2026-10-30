import { OnStateChange, SysctlEntry, parseEntries, toRecord } from '@/components/system-config/shared'
import { AutocompleteInput, type AutocompleteOption } from '@/components/ui/AutocompleteInput'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { Input, Button, EmptyState, Spinner, PreferencesGroup } from 'cheval-ui'
import { Plus, SlidersHorizontal, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'

type SysctlTabShape = { onStateChange: OnStateChange }

export function SysctlTab({ onStateChange }: SysctlTabShape) {
  const { data, isLoading } = useFetch(() => api.apiConfigSectionGet({ section: 'sysctl' }))
  const [entries, setEntries] = useState<SysctlEntry[] | null>(null)
  const { isDirty, markDirty, save, saving } = usePageSave('sysctl')
  const { data: available } = useFetch(() => api.apiSystemSysctlGet())

  const currentValues = useMemo(() => new Map((available ?? []).map((p) => [p.key, p.value])), [available])
  const options = useMemo<AutocompleteOption[]>(
    () => (available ?? []).map((p) => ({ value: p.key, label: p.key, meta: p.value.length > 24 ? `${p.value.slice(0, 24)}…` : p.value })),
    [available],
  )

  const parsed = useMemo(() => parseEntries(data), [data])
  const current = entries ?? parsed

  const handleSave = useCallback(() => save(toRecord(current)), [save, current])
  useEffect(() => { onStateChange({ isDirty, saving, save: handleSave }) }, [isDirty, saving, handleSave, onStateChange])

  const handleChange = (index: number, field: 'key' | 'value', val: string) => {
    setEntries(current.map((e, i) => (i === index ? { ...e, [field]: val } : e))); markDirty()
  }
  const handleKey = (index: number, key: string) => {
    setEntries(current.map((e, i) => (i === index ? { key, value: e.value || (currentValues.get(key) ?? '') } : e))); markDirty()
  }
  const handleAdd = () => { setEntries([...current, { key: '', value: '' }]); markDirty() }
  const handleRemove = (index: number) => { setEntries(current.filter((_, i) => i !== index)); markDirty() }

  if (isLoading) return <Spinner />

  return (
    <>
      {current.length === 0 ? (
        <EmptyState
          className="max-w-2xl mx-auto"
          icon={<SlidersHorizontal />}
          title="No Parameters"
          action={<Button variant="outline" onClick={handleAdd} className="gap-2"><Plus className="h-4 w-4" />Add Parameter</Button>}
        />
      ) : (
        <PreferencesGroup
          title="Parameters"
          description="Written to /etc/sysctl.d/99-routier.conf on apply"
          header={
            <Button variant="outline" size="sm" onClick={handleAdd} className="gap-1.5">
              <Plus className="h-4 w-4" />Add
            </Button>
          }
        >
          {current.map((entry, index) => (
            <div key={index} className="flex flex-wrap items-center gap-2 px-4 py-2 sm:flex-nowrap">
              <div className="w-full min-w-0 sm:w-auto sm:flex-1">
                <AutocompleteInput
                  value={entry.key}
                  options={options}
                  onChange={(key) => handleKey(index, key)}
                  placeholder="net.ipv4.ip_forward"
                  emptyMessage="No matching parameter"
                  popupMinWidth={360}
                  className="font-mono text-sm"
                  inputProps={{ 'aria-label': 'Parameter' }}
                />
              </div>
              <span className="hidden shrink-0 text-muted-foreground sm:inline">=</span>
              <Input
                value={entry.value}
                onChange={(e) => handleChange(index, 'value', e.target.value)}
                placeholder={currentValues.get(entry.key) ?? '1'}
                aria-label="Value"
                className="min-w-0 flex-1 bg-transparent font-mono text-sm sm:w-32 sm:flex-none text-foreground outline-none placeholder:text-muted-foreground/50"
              />
              <Button
                variant="ghost"
                size="icon"
                onClick={() => handleRemove(index)}
                className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive"
               aria-label="Remove Parameter" title="Remove Parameter">
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))}
        </PreferencesGroup>
      )}
    </>
  )
}
