import { useState, useEffect } from 'react'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { SaveButton } from 'cheval-ui'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from 'cheval-ui'
import { PageHeader } from 'cheval-ui'
import { PreferencesGroup, ComboRow, EntryRow } from 'cheval-ui'
import { Spinner } from 'cheval-ui'

interface LoggingConfig {
  target?: string
  file?: string
  level?: string
}

const LOG_LEVELS = ['debug', 'info', 'warn', 'error']
const LOG_TARGETS = ['', 'syslog', 'file', 'stdout', 'stderr']

export default function Logging() {
  const { data, isLoading } = useFetch<LoggingConfig>(() => api.apiConfigSectionGet({ section: 'logging' }) as Promise<LoggingConfig>)
  const [target, setTarget] = useState('')
  const [file, setFile] = useState('')
  const [level, setLevel] = useState('')
  const [initialized, setInitialized] = useState(false)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('logging')

  useEffect(() => {
    if (data && !initialized) {
      setTarget(data.target ?? '')
      setFile(data.file ?? '')
      setLevel(data.level ?? '')
      setInitialized(true)
    }
  }, [data, initialized])

  const buildPayload = (t: string, f: string, l: string): LoggingConfig => {
    const payload: LoggingConfig = {}
    if (t) payload.target = t
    if (f) payload.file = f
    if (l) payload.level = l
    return payload
  }

  if (isLoading) {
    return (
      <Spinner />
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Logging" description="Configure log output target and verbosity" action={
        <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(buildPayload(target, file, level))} onCancel={() => { setInitialized(false); reset() }} />
      } />

      <div className="max-w-xl">
        <PreferencesGroup title="Output">
          <ComboRow title="Target" subtitle="Where log lines are written">
            <Select value={target || '_none'} onValueChange={(v) => { setTarget(v === '_none' ? '' : v); markDirty() }}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {LOG_TARGETS.map((t) => (
                  <SelectItem key={t || '_none'} value={t || '_none'}>{t || '(not set)'}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </ComboRow>

          {target === 'file' && (
            <EntryRow
              id="log-file"
              title="File path"
              value={file}
              onChange={(e) => { setFile(e.target.value); markDirty() }}
              placeholder="/var/log/routier.log"
              className="font-mono"
            />
          )}

          <ComboRow title="Level" subtitle="Minimum verbosity to emit">
            <Select value={level || '_none'} onValueChange={(v) => { setLevel(v === '_none' ? '' : v); markDirty() }}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="_none">(not set)</SelectItem>
                {LOG_LEVELS.map((l) => (
                  <SelectItem key={l} value={l}>{l}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </ComboRow>
        </PreferencesGroup>
      </div>
    </div>
  )
}
