import { LOG_SUBTABS, LogSettingsPanel, LogSource, LogSubTab } from '@/components/monitor/LogsPanel'
import { getToken } from '@/lib/utils'
import { Button, Tabs, useTabState } from 'cheval-ui'
import { ChevronsDown, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'

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

  const startStream = useCallback(async (source: LogSource) => {
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
  }, [])

  const stopStream = () => { abortRef.current?.abort(); abortRef.current = null; setStreaming(false) }

  const switchTab = (tab: LogSubTab) => {
    setActiveTab(tab)
  }

  useEffect(() => {
    if (activeTab === 'settings') return
    setLines([])
    startStream(activeTab)
    return () => { abortRef.current?.abort() }
  }, [activeTab, startStream])

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
