import { DHCP_LOG_TABS, DhcpLogSource } from '@/components/monitor/DhcpPanel'
import { getToken } from '@/lib/utils'
import { Button, Tabs, useTabState } from 'cheval-ui'
import { ChevronsDown, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

export function DhcpLogsPanel() {
  const [source, setSource] = useTabState<DhcpLogSource>('monitor.dhcp.logs', 'kea-dhcp4')
  const [lines, setLines] = useState<string[]>([])
  const [streaming, setStreaming] = useState(false)
  const [autoScroll, setAutoScroll] = useState(true)
  const scrollBoxRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  const stickToBottom = () => {
    const el = scrollBoxRef.current
    if (el) el.scrollTop = el.scrollHeight
  }

  useEffect(() => {
    if (autoScroll) stickToBottom()
  }, [lines, autoScroll])

  const handleScroll = () => {
    const el = scrollBoxRef.current
    if (!el) return
    setAutoScroll(el.scrollHeight - el.scrollTop - el.clientHeight < 32)
  }

  const startStream = useCallback(async (selectedSource: DhcpLogSource) => {
    if (abortRef.current) abortRef.current.abort()
    const controller = new AbortController()
    abortRef.current = controller
    setStreaming(true); setAutoScroll(true)
    try {
      const res = await fetch(`/api/dhcp/leases/stream?source=${encodeURIComponent(selectedSource)}`, {
        headers: { Authorization: `Bearer ${getToken() ?? ''}` },
        signal: controller.signal,
      })
      if (!res.ok || !res.body) throw new Error(`Unable to stream ${selectedSource} logs (${res.status})`)
      const reader = res.body.getReader(); const decoder = new TextDecoder(); let buffer = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done || controller.signal.aborted) break
        buffer += decoder.decode(value, { stream: true })
        const parts = buffer.split('\n\n'); buffer = parts.pop() ?? ''
        for (const part of parts) {
          const line = part.split('\n').filter((value) => value.startsWith('data:')).map((value) => value.slice(5).trimStart()).join('\n')
          if (line) setLines((prev) => { const next = [...prev, line]; return next.length > 2000 ? next.slice(-2000) : next })
        }
      }
    } catch (err: unknown) {
      if (!controller.signal.aborted && err instanceof Error) toast.error(err.message)
    } finally { if (abortRef.current === controller) setStreaming(false) }
  }, [])

  const stopStream = () => { abortRef.current?.abort(); abortRef.current = null; setStreaming(false) }

  useEffect(() => {
    setLines([])
    startStream(source)
    return () => { abortRef.current?.abort() }
  }, [source, startStream])

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Tabs tabs={DHCP_LOG_TABS} active={source} onChange={setSource} variant="pills" />
        <div className="flex items-center gap-2">
          <Button size="sm" variant={streaming ? 'default' : 'outline'} onClick={streaming ? stopStream : () => startStream(source)}
            className={`gap-1.5 ${streaming ? 'bg-green-600 hover:bg-green-700 border-green-600 dark:bg-green-700 dark:hover:bg-green-600' : ''}`}>
            <span className={`h-1.5 w-1.5 rounded-full inline-block ${streaming ? 'bg-white animate-pulse' : 'bg-muted-foreground'}`} />
            Live
          </Button>
          <Button size="sm" variant="ghost" onClick={() => setLines([])} className="gap-1.5 text-muted-foreground">
            <Trash2 className="h-3.5 w-3.5" />Clear
          </Button>
          <span className="text-xs text-muted-foreground">{lines.length} lines</span>
        </div>
      </div>

      <div className="relative">
        <div ref={scrollBoxRef} onScroll={handleScroll} className="bg-zinc-950 text-zinc-100 font-mono text-xs p-3 rounded-md overflow-auto h-[520px]">
          {lines.length === 0
            ? <span className="text-zinc-500">{streaming ? `Waiting for ${source} log output…` : 'Press Live to begin streaming logs.'}</span>
            : lines.map((line, i) => <div key={i}>{line}</div>)
          }
        </div>
        {!autoScroll && lines.length > 0 && (
          <button type="button" onClick={() => { setAutoScroll(true); stickToBottom() }}
            className="absolute bottom-3 right-5 flex items-center gap-1 rounded-full bg-zinc-700 hover:bg-zinc-600 text-zinc-100 text-xs px-2.5 py-1 shadow">
            <ChevronsDown className="h-3 w-3" />Tail
          </button>
        )}
      </div>
    </div>
  )
}
