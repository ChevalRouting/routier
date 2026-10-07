import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { ComboRow, EntryRow, PageHeader, PreferencesGroup, SaveButton, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Spinner } from 'cheval-ui'
import { useEffect, useState } from 'react'

interface LoggingConfig {
  target?: string
  file?: string
  level?: string
}

const LOG_LEVELS = ['debug', 'info', 'warn', 'error']
const LOG_TARGETS = ['', 'syslog', 'file', 'stdout', 'stderr']

export default function Logging() {
  const { data, isLoading, reload } = useFetch<LoggingConfig>(() => api.apiConfigSectionGet({ section: 'logging' }) as Promise<LoggingConfig>)
  const [target, setTarget] = useState('')
  const [file, setFile] = useState('')
  const [level, setLevel] = useState('')
  const [initialized, setInitialized] = useState(false)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('logging')
  useDataRefresh(() => setInitialized(false))

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

  const handleCancel = () => { setInitialized(false); reset(); reload(true) }

  return (
    <div className="space-y-6">
      <PageHeader title="Logging" description="Configure log output target and verbosity" action={
        <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(buildPayload(target, file, level))} onCancel={handleCancel} />
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
