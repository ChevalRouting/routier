import { DraftControls } from '@/components/DraftControls'
import { LOG_LEVELS, LOG_TARGETS, LoggingConfig } from '@/components/monitor/LogsPanel'
import { api } from '@/lib/client'
import { EntryRow, PreferencesGroup } from 'cheval-ui'
import { SelectRow } from '@/components/ui/SelectRow'
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

  const changeTarget = (value: string | undefined) => { setTarget(value ?? ''); setIsDirty(true) }
  const changeLevel = (value: string | undefined) => { setLevel(value ?? ''); setIsDirty(true) }

  return (
    <PreferencesGroup>
      <SelectRow title="Target" value={target} onChange={changeTarget} options={LOG_TARGETS.filter(Boolean)} noneLabel="Not set" mono={false} />
      {target === 'file' && (
        <EntryRow title="File Path" value={file} onChange={(e) => { setFile(e.target.value); setIsDirty(true) }} placeholder="/var/log/routier.log" className="font-mono" />
      )}
      <SelectRow title="Level" value={level} onChange={changeLevel} options={LOG_LEVELS} noneLabel="Not set" mono={false} />
      <DraftControls isDirty={isDirty} saving={saving} onClick={handleSave} />
    </PreferencesGroup>
  )
}
