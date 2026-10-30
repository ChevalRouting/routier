import { DraftControls } from '@/components/DraftControls'
import { SectionCard, orderedSections } from '@/components/ConfigView'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Textarea, Button, Card, CardContent, PageHeader, Spinner, useClipboard, Segmented, ReloadButton } from 'cheval-ui'
import { Check, Copy, Pencil } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

function toYAML(val: unknown, indent = 0): string {
  const pad = '  '.repeat(indent)
  if (val === null || val === undefined) return 'null'
  if (typeof val === 'boolean') return val ? 'true' : 'false'
  if (typeof val === 'number') return String(val)
  if (typeof val === 'string') {
    if (val === '') return "''"
    if (val.includes('\n') || val.includes(':') || val.includes('#') || val.startsWith(' '))
      return `"${val.replace(/"/g, '\\"')}"`
    return val
  }
  if (Array.isArray(val)) {
    if (val.length === 0) return '[]'
    return '\n' + val.map((v) => `${pad}- ${toYAML(v, indent + 1).trimStart()}`).join('\n')
  }
  if (typeof val === 'object') {
    const entries = Object.entries(val as Record<string, unknown>).filter(
      ([, v]) => v !== null && v !== undefined
    )
    if (entries.length === 0) return '{}'
    return (
      '\n' +
      entries
        .map(([k, v]) => {
          const rendered = toYAML(v, indent + 1)
          return `${pad}${k}:${rendered.startsWith('\n') ? rendered : ' ' + rendered}`
        })
        .join('\n')
    )
  }
  return String(val)
}

export default function ConfigBrowser() {
  const { data, isLoading, reload } = useFetch<Record<string, unknown>>(
    () => api.apiConfigGet() as unknown as Promise<Record<string, unknown>>
  )
  const [view, setView] = useState<'tree' | 'yaml'>('tree')
  const { copied, copy } = useClipboard(2000)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')
  const [saving, setSaving] = useState(false)

  const selectView = (next: 'tree' | 'yaml') => {
    setView(next)
    if (next === 'tree') setEditing(false)
  }

  const handleCopy = () => {
    if (!data) return
    const yaml = Object.entries(data)
      .filter(([, v]) => v !== null && v !== undefined)
      .map(([k, v]) => {
        const rendered = toYAML(v, 1)
        return `${k}:${rendered.startsWith('\n') ? rendered : ' ' + rendered}`
      })
      .join('\n')
    copy(yaml)
  }

  if (isLoading) {
    return (
      <Spinner />
    )
  }

  const sections = orderedSections(data)

  const yamlText = sections
    .filter(([, v]) => v !== null && v !== undefined)
    .map(([k, v]) => {
      const rendered = toYAML(v, 1)
      return `${k}:${rendered.startsWith('\n') ? rendered : ' ' + rendered}`
    })
    .join('\n')

  const startEdit = () => { setDraft(yamlText); setEditing(true) }
  const cancelEdit = () => setEditing(false)
  const saveEdit = async () => {
    setSaving(true)
    try {
      await api.apiConfigRawPut({ body: draft })
      toast.success('Config staged, apply to take effect')
      setEditing(false)
      reload()
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Config Browser" action={
        <div className="flex flex-wrap items-center gap-2">
          <Segmented value={view} onChange={selectView} options={[{ value: 'tree', label: 'Tree' }, { value: 'yaml', label: 'YAML' }]} />
          <Button variant="outline" onClick={handleCopy} className="gap-2">
            {copied ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
            {copied ? 'Copied' : 'Copy YAML'}
          </Button>
          <ReloadButton onClick={() => reload()} />
        </div>
      } />

      {view === 'yaml' ? (
        <Card>
          <div className="flex items-center justify-between border-b px-4 py-2">
            <span className="text-xs text-muted-foreground">{editing ? 'Editing config.yml, saving stages the change' : 'config.yml'}</span>
            {editing ? (
              <DraftControls isDirty={draft !== yamlText} saving={saving} onClick={saveEdit} onCancel={cancelEdit} />
            ) : (
              <Button variant="outline" size="sm" onClick={startEdit} className="gap-1.5">
                <Pencil className="h-3.5 w-3.5" />Edit
              </Button>
            )}
          </div>
          <CardContent className="p-0">
            {editing ? (
              <Textarea
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                spellCheck={false}
                autoCorrect="off"
                autoCapitalize="off"
                className="h-[60vh] w-full resize-y bg-transparent p-4 font-mono text-xs leading-5 text-foreground outline-none"
                onKeyDown={(e) => {
                  if (e.key === 'Tab') {
                    e.preventDefault()
                    const t = e.currentTarget
                    const s = t.selectionStart
                    const next = draft.slice(0, s) + '  ' + draft.slice(t.selectionEnd)
                    setDraft(next)
                    requestAnimationFrame(() => { t.selectionStart = t.selectionEnd = s + 2 })
                  }
                }}
              />
            ) : (
              <pre className="whitespace-pre-wrap break-all p-4 text-xs font-mono leading-5 text-foreground">
                {yamlText || '# empty config'}
              </pre>
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="md:columns-2 [column-gap:0.75rem]">
          {sections.map(([k, v]) => (
            <div key={k} className="mb-3 break-inside-avoid">
              <SectionCard name={k} value={v} />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
