import { OnStateChange, SysctlEntry, parseEntries, toRecord } from '@/components/system-config/shared'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { Input, Button, EmptyState, PreferencesColumns, Spinner } from 'cheval-ui'
import { Plus, SlidersHorizontal, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'

type SysctlTabShape = { onStateChange: OnStateChange }

export function SysctlTab({ onStateChange }: SysctlTabShape) {
  const { data, isLoading } = useFetch(() => api.apiConfigSectionGet({ section: 'sysctl' }))
  const [entries, setEntries] = useState<SysctlEntry[] | null>(null)
  const { isDirty, markDirty, save, saving } = usePageSave('sysctl')

  const parsed = useMemo(() => parseEntries(data), [data])
  const current = entries ?? parsed

  const handleSave = useCallback(() => save(toRecord(current)), [save, current])
  useEffect(() => { onStateChange({ isDirty, saving, save: handleSave }) }, [isDirty, saving, handleSave, onStateChange])

  const handleChange = (index: number, field: 'key' | 'value', val: string) => {
    setEntries(current.map((e, i) => (i === index ? { ...e, [field]: val } : e))); markDirty()
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
          title="No parameters"
          message="Written to /etc/sysctl.d/99-routier.conf and applied on each config apply."
          action={<Button variant="outline" onClick={handleAdd} className="gap-2"><Plus className="h-4 w-4" />Add parameter</Button>}
        />
      ) : (
        <PreferencesColumns
          title="Parameters"
          description="Written to /etc/sysctl.d/99-routier.conf and applied on each config apply."
          header={
            <Button variant="outline" size="sm" onClick={handleAdd} className="gap-1.5">
              <Plus className="h-4 w-4" />Add
            </Button>
          }
        >
          {current.map((entry, index) => (
            <div key={index} className="flex items-center gap-2 px-4 py-2">
              <Input
                value={entry.key}
                onChange={(e) => handleChange(index, 'key', e.target.value)}
                placeholder="net.ipv4.ip_forward"
                className="min-w-0 flex-1 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50"
              />
              <span className="shrink-0 text-muted-foreground">=</span>
              <Input
                value={entry.value}
                onChange={(e) => handleChange(index, 'value', e.target.value)}
                placeholder="1"
                className="w-32 shrink-0 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50"
              />
              <Button
                variant="ghost"
                size="icon"
                onClick={() => handleRemove(index)}
                className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))}
        </PreferencesColumns>
      )}
    </>
  )
}
