import { readEventStream } from '@/lib/eventStream'
import { Button } from 'cheval-ui'
import { Trash2 } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

const MAX_LINES = 2000

function appendLine(lines: string[], line: string): string[] {
  const next = [...lines, line]
  return next.length > MAX_LINES ? next.slice(-MAX_LINES) : next
}

export function useLiveLog(path: string | null) {
  const [lines, setLines] = useState<string[]>([])
  const [streaming, setStreaming] = useState(false)
  const abortRef = useRef<AbortController | null>(null)

  const stop = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
    setStreaming(false)
  }, [])

  const start = useCallback(async () => {
    if (!path) return
    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller
    setStreaming(true)
    try {
      await readEventStream(path, controller.signal, (line) => setLines((prev) => appendLine(prev, line)))
    } catch (err: unknown) {
      if (!controller.signal.aborted && err instanceof Error) toast.error(err.message)
    } finally {
      if (abortRef.current === controller) setStreaming(false)
    }
  }, [path])

  useEffect(() => {
    setLines([])
    void start()
    return () => { abortRef.current?.abort() }
  }, [start])

  const clear = () => setLines([])

  return { lines, streaming, start, stop, clear }
}

type LiveLogControlsShape = { log: ReturnType<typeof useLiveLog> }

export function LiveLogControls({ log }: LiveLogControlsShape) {
  const toggle = () => { if (log.streaming) log.stop(); else void log.start() }

  return (
    <div className="flex items-center gap-2">
      <Button size="sm" variant={log.streaming ? 'secondary' : 'outline'} aria-pressed={log.streaming} onClick={toggle} className="gap-1.5">
        <span className={`h-1.5 w-1.5 rounded-full inline-block ${log.streaming ? 'bg-success' : 'bg-muted-foreground'}`} />
        Live
      </Button>
      <Button size="sm" variant="ghost" onClick={log.clear} className="gap-1.5 text-muted-foreground">
        <Trash2 className="h-3.5 w-3.5" />Clear
      </Button>
      <span className="text-xs text-muted-foreground">{log.lines.length} lines</span>
    </div>
  )
}

type LiveLogViewShape = { log: ReturnType<typeof useLiveLog>; waitingLabel: string }

export function LiveLogView({ log, waitingLabel }: LiveLogViewShape) {
  const [autoScroll, setAutoScroll] = useState(true)
  const scrollBoxRef = useRef<HTMLDivElement>(null)

  const stickToBottom = () => {
    const el = scrollBoxRef.current
    if (el) el.scrollTop = el.scrollHeight
  }

  useEffect(() => {
    if (autoScroll) stickToBottom()
  }, [log.lines, autoScroll])

  const handleScroll = () => {
    const el = scrollBoxRef.current
    if (el) setAutoScroll(el.scrollHeight - el.scrollTop - el.clientHeight < 32)
  }

  const jumpToEnd = () => {
    setAutoScroll(true)
    stickToBottom()
  }

  return (
    <div className="relative">
      <div ref={scrollBoxRef} onScroll={handleScroll} className="h-[32rem] overflow-auto rounded-md bg-muted p-3 font-mono text-xs text-foreground">
        {log.lines.length === 0
          ? <span className="text-muted-foreground">{log.streaming ? waitingLabel : 'Not streaming'}</span>
          : log.lines.map((line, i) => <div key={i}>{line}</div>)}
      </div>
      {!autoScroll && log.lines.length > 0 && (
        <Button size="sm" variant="secondary" onClick={jumpToEnd} className="absolute bottom-3 right-5 shadow">Jump to End</Button>
      )}
    </div>
  )
}
