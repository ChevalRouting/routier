import { LOG_LEVELS, LOG_TARGETS, LoggingConfig } from '@/components/monitor/LogsPanel'
import { api } from '@/lib/client'
import { Button, Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

export function LogSettingsPanel() {
  const [data, setData] = useState<LoggingConfig | null>(null)
  const [target, setTarget] = useState('')
  const [file, setFile] = useState('')
  const [level, setLevel] = useState('')
  const [initialized, setInitialized] = useState(false)
  const [isDirty, setIsDirty] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    api.apiConfigSectionGet({ section: 'logging' }).then((d) => setData(d as LoggingConfig)).catch(() => {})
  }, [])

  useEffect(() => {
    if (data && !initialized) {
      setTarget(data.target ?? ''); setFile(data.file ?? ''); setLevel(data.level ?? '')
      setInitialized(true)
    }
  }, [data, initialized])

  const handleSave = async () => {
    setSaving(true)
    try {
      const payload: LoggingConfig = {}
      if (target) payload.target = target
      if (file)   payload.file   = file
      if (level)  payload.level  = level
      await api.apiConfigSectionPut({ section: 'logging', body: payload })
      setIsDirty(false)
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save')
    } finally { setSaving(false) }
  }

  return (
    <div className="space-y-4 max-w-xs">
      <div className="space-y-1.5">
        <Label>Target</Label>
        <Select value={target || '_none'} onValueChange={(v) => { setTarget(v === '_none' ? '' : v); setIsDirty(true) }}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            {LOG_TARGETS.map((t) => <SelectItem key={t || '_none'} value={t || '_none'}>{t || '(not set)'}</SelectItem>)}
          </SelectContent>
        </Select>
      </div>
      {target === 'file' && (
        <div className="space-y-1.5">
          <Label>File path</Label>
          <Input value={file} onChange={(e) => { setFile(e.target.value); setIsDirty(true) }} placeholder="/var/log/routier.log" className="font-mono" />
        </div>
      )}
      <div className="space-y-1.5">
        <Label>Level</Label>
        <Select value={level || '_none'} onValueChange={(v) => { setLevel(v === '_none' ? '' : v); setIsDirty(true) }}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="_none">(not set)</SelectItem>
            {LOG_LEVELS.map((l) => <SelectItem key={l} value={l}>{l}</SelectItem>)}
          </SelectContent>
        </Select>
      </div>
      <Button onClick={handleSave} disabled={!isDirty || saving} size="sm">{saving ? 'Saving…' : 'Save'}</Button>
    </div>
  )
}
