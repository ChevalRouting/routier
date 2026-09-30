import { useState, useEffect, useRef } from 'react'
import { toast } from 'sonner'
import { useTabState } from 'cheval-ui'
import { api } from '@/lib/client'
import { getToken } from '@/lib/utils'
import { Tabs } from 'cheval-ui'
import { Input } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { Trash2, ChevronsDown } from 'lucide-react'

type LogSource = 'messages' | 'dmesg'
type LogSubTab = LogSource | 'settings'

export const LOG_LEVELS = ['debug', 'info', 'warn', 'error']
export const LOG_TARGETS = ['', 'syslog', 'file', 'stdout', 'stderr']

export interface LoggingConfig { target?: string; file?: string; level?: string }

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

export const LOG_SUBTABS: { key: LogSubTab; label: string }[] = [
  { key: 'messages', label: 'System Log' },
  { key: 'dmesg',    label: 'Kernel Log' },
  { key: 'settings', label: 'Log Settings' },
]

export function LogsPanel() {
  const [activeTab, setActiveTab] = useTabState<LogSubTab>('monitor.logs', 'messages')
  const [lines, setLines] = useState<string[]>([])
  const [streaming, setStreaming] = useState(false)
  const [autoScroll, setAutoScroll] = useState(true)
  const bottomRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)
  const scrollBoxRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (autoScroll) bottomRef.current?.scrollIntoView({ behavior: 'instant' })
  }, [lines, autoScroll])

  const handleScroll = () => {
    const el = scrollBoxRef.current
    if (!el) return
    setAutoScroll(el.scrollHeight - el.scrollTop - el.clientHeight < 32)
  }

  const startStream = async (source: LogSource) => {
    if (abortRef.current) abortRef.current.abort()
    const controller = new AbortController()
    abortRef.current = controller
    setStreaming(true); setAutoScroll(true)
    try {
      const res = await fetch(`/api/logs/stream?source=${source}`, {
        headers: { Authorization: `Bearer ${getToken() ?? ''}` },
        signal: controller.signal,
      })
      if (!res.ok || !res.body) { setStreaming(false); return }
      const reader = res.body.getReader(); const decoder = new TextDecoder(); let buffer = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const parts = buffer.split('\n\n'); buffer = parts.pop() ?? ''
        for (const part of parts) {
          const line = part.replace(/^data: /, '').trim()
          if (line) setLines(prev => { const next = [...prev, line]; return next.length > 2000 ? next.slice(-2000) : next })
        }
      }
    } catch (err: unknown) {
      if (err instanceof Error && err.name !== 'AbortError') console.error(err)
    } finally { setStreaming(false) }
  }

  const stopStream = () => { abortRef.current?.abort(); abortRef.current = null; setStreaming(false) }

  const switchTab = (tab: LogSubTab) => {
    setActiveTab(tab)
    if (tab === 'settings') { stopStream(); return }
    setLines([]); startStream(tab)
  }

  useEffect(() => {
    startStream(activeTab as LogSource)
    return () => { abortRef.current?.abort() }
  }, [])

  const isLogTab = activeTab === 'messages' || activeTab === 'dmesg'

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <Tabs tabs={LOG_SUBTABS} active={activeTab} onChange={switchTab} variant="pills" />
        {isLogTab && (
          <div className="flex items-center gap-2">
            <Button size="sm" variant={streaming ? 'default' : 'outline'} onClick={streaming ? stopStream : () => startStream(activeTab as LogSource)}
              className={`gap-1.5 ${streaming ? 'bg-green-600 hover:bg-green-700 border-green-600 dark:bg-green-700 dark:hover:bg-green-600' : ''}`}>
              <span className={`h-1.5 w-1.5 rounded-full inline-block ${streaming ? 'bg-white animate-pulse' : 'bg-muted-foreground'}`} />
              Live
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setLines([])} className="gap-1.5 text-muted-foreground">
              <Trash2 className="h-3.5 w-3.5" />Clear
            </Button>
            <span className="text-xs text-muted-foreground">{lines.length} lines</span>
          </div>
        )}
      </div>

      {activeTab === 'settings' ? <LogSettingsPanel /> : (
        <div className="relative">
          <div ref={scrollBoxRef} onScroll={handleScroll} className="bg-zinc-950 text-zinc-100 font-mono text-xs p-3 rounded-md overflow-auto h-[520px]">
            {lines.length === 0
              ? <span className="text-zinc-500">{streaming ? 'Waiting for log output…' : 'Press Live to begin streaming logs.'}</span>
              : lines.map((line, i) => <div key={i}>{line}</div>)
            }
            <div ref={bottomRef} />
          </div>
          {!autoScroll && lines.length > 0 && (
            <button type="button" onClick={() => { setAutoScroll(true); bottomRef.current?.scrollIntoView({ behavior: 'smooth' }) }}
              className="absolute bottom-3 right-5 flex items-center gap-1 rounded-full bg-zinc-700 hover:bg-zinc-600 text-zinc-100 text-xs px-2.5 py-1 shadow">
              <ChevronsDown className="h-3 w-3" />Tail
            </button>
          )}
        </div>
      )}
    </div>
  )
}

