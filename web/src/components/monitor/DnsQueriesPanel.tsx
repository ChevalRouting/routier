import { Fragment, useEffect, useMemo, useRef, useState } from 'react'
import { Card, CardContent, Button, Input } from 'cheval-ui'
import { cn } from 'cheval-ui'
import { activeBaseUrl } from '@/lib/instance'
import { getToken } from '@/lib/utils'
import { Search, Trash2, ChevronRight, ChevronDown } from 'lucide-react'

interface DnsQuery {
  time: string
  client: string
  port: string
  name: string
  class: string
  type: string
  flags: string
  dest?: string
  raw: string
}

type Row = DnsQuery & { id: number }

const FLAG_MEANINGS: Record<string, string> = {
  '+': 'recursion desired',
  '-': 'recursion not desired',
  E: 'EDNS',
  T: 'over TCP',
  D: 'DNSSEC OK',
  C: 'checking disabled',
  S: 'signed (TSIG/SIG0)',
  V: 'valid cookie',
  K: 'DNS cookie',
}

function decodeFlags(flags: string): string[] {
  const out: string[] = []
  for (const ch of flags) {
    if (FLAG_MEANINGS[ch] && !out.includes(FLAG_MEANINGS[ch])) out.push(FLAG_MEANINGS[ch])
  }
  return out
}

function shortTime(time: string): string {
  const parts = time.split(' ')
  return parts.length > 1 ? parts[1] : time
}

let counter = 0

export function DnsQueriesPanel() {
  const [rows, setRows] = useState<Row[]>([])
  const [live, setLive] = useState(true)
  const [status, setStatus] = useState('Connecting…')
  const [q, setQ] = useState('')
  const [expanded, setExpanded] = useState<number | null>(null)
  const connectedRef = useRef(false)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    if (!live) {
      abortRef.current?.abort()
      abortRef.current = null
      return
    }

    const controller = new AbortController()
    abortRef.current = controller

    let retry: ReturnType<typeof setTimeout>
    const run = async () => {
      setStatus('Connecting…')
      try {
        const res = await fetch(`${activeBaseUrl()}/api/dns/queries/stream${connectedRef.current ? "?history=0" : ""}`, {
          headers: { Authorization: `Bearer ${getToken() ?? ''}` },
          signal: controller.signal,
        })
        if (!res.ok || !res.body) throw new Error(`Unable to stream queries (HTTP ${res.status})`)
        setStatus('Live')
        connectedRef.current = true
        const reader = res.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''
        while (true) {
          const { done, value } = await reader.read()
          if (done) throw new Error('Query stream disconnected')
          buffer += decoder.decode(value, { stream: true })
          const parts = buffer.split('\n\n')
          buffer = parts.pop() ?? ''
          for (const part of parts) {
            const line = part.split('\n').filter((line) => line.startsWith('data:')).map((line) => line.slice(5).trimStart()).join('\n')
            if (!line) continue
            try {
              const parsed = JSON.parse(line) as DnsQuery
              setRows((prev) => {
                const next = [{ ...parsed, id: counter++ }, ...prev]
                return next.length > 500 ? next.slice(0, 500) : next
              })
            } catch {
              /* ignore malformed frame */
            }
          }
        }
      } catch (err) {
        if (!controller.signal.aborted) {
          setStatus(`${err instanceof Error ? err.message : 'Connection failed'} — retrying…`)
          retry = setTimeout(run, 3000)
        }
      }
    }

    run()
    return () => { controller.abort(); clearTimeout(retry) }
  }, [live])

  const filtered = useMemo(() => {
    if (!q) return rows
    const lq = q.toLowerCase()
    return rows.filter((r) =>
      r.name.toLowerCase().includes(lq) || r.client.includes(lq) || r.type.toLowerCase().includes(lq),
    )
  }, [rows, q])

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <Search className="h-4 w-4 text-muted-foreground shrink-0" />
        <Input
          placeholder="Filter by name, client or type…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          className="max-w-xs h-8 text-sm"
        />
        <span className="text-xs text-muted-foreground">{filtered.length} queries</span>
        <span role="status" className="text-xs text-muted-foreground">{live ? status : 'Paused'}</span>
        <div className="ml-auto flex items-center gap-2">
          <Button
            size="sm"
            variant={live ? 'default' : 'outline'}
            onClick={() => setLive((v) => !v)}
            className="gap-1.5"
          >
            <span className={cn('h-1.5 w-1.5 rounded-full inline-block', live && status === 'Live' ? 'bg-white animate-pulse' : 'bg-muted-foreground')} />
            {live ? 'Pause' : 'Resume'}
          </Button>
          <Button size="sm" variant="ghost" onClick={() => setRows([])} className="gap-1.5 text-muted-foreground">
            <Trash2 className="h-3.5 w-3.5" />Clear
          </Button>
        </div>
      </div>

      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-left">
                  <th className="w-8" />
                  <th className="px-3 py-2.5 font-medium text-muted-foreground w-28">Time</th>
                  <th className="px-3 py-2.5 font-medium text-muted-foreground">Client</th>
                  <th className="px-3 py-2.5 font-medium text-muted-foreground">Name</th>
                  <th className="px-3 py-2.5 font-medium text-muted-foreground w-20">Type</th>
                  <th className="px-3 py-2.5 font-medium text-muted-foreground w-16">Class</th>
                  <th className="px-3 py-2.5 font-medium text-muted-foreground w-24">Flags</th>
                </tr>
              </thead>
              <tbody>
                {filtered.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-8 text-center text-sm text-muted-foreground">
                      {q && rows.length > 0 ? 'No queries match your filter.' : live ? 'Waiting for queries… (enable query logging in Resolver settings if none appear)' : 'Paused.'}
                    </td>
                  </tr>
                ) : filtered.map((r) => {
                  const open = expanded === r.id
                  return (
                    <Fragment key={r.id}>
                      <tr
                        className="border-b border-border last:border-b-0 hover:bg-muted/30 cursor-pointer"
                        onClick={() => setExpanded(open ? null : r.id)}
                      >
                        <td className="pl-3 text-muted-foreground">
                          {open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
                        </td>
                        <td className="px-3 py-2 font-mono text-xs text-muted-foreground tabular-nums">{shortTime(r.time)}</td>
                        <td className="px-3 py-2 font-mono text-xs">{r.client}</td>
                        <td className="px-3 py-2 font-mono truncate max-w-xs">{r.name}</td>
                        <td className="px-3 py-2 font-mono text-xs">
                          <span className="inline-flex items-center rounded-full bg-primary/10 text-primary px-2 py-0.5">{r.type}</span>
                        </td>
                        <td className="px-3 py-2 font-mono text-xs text-muted-foreground">{r.class}</td>
                        <td className="px-3 py-2 font-mono text-xs text-muted-foreground">{r.flags}</td>
                      </tr>
                      {open && (
                        <tr className="border-b border-border bg-muted/20">
                          <td />
                          <td colSpan={6} className="px-3 py-3">
                            <dl className="grid grid-cols-[7rem_1fr] gap-x-4 gap-y-1 text-xs">
                              <dt className="text-muted-foreground">Timestamp</dt>
                              <dd className="font-mono">{r.time || '–'}</dd>
                              <dt className="text-muted-foreground">Client</dt>
                              <dd className="font-mono">{r.client}#{r.port}</dd>
                              <dt className="text-muted-foreground">Query</dt>
                              <dd className="font-mono">{r.name} {r.class} {r.type}</dd>
                              {r.dest && (<><dt className="text-muted-foreground">Resolver</dt><dd className="font-mono">{r.dest}</dd></>)}
                              <dt className="text-muted-foreground">Flags</dt>
                              <dd className="font-mono">{r.flags}{decodeFlags(r.flags).length > 0 && <span className="text-muted-foreground"> ({decodeFlags(r.flags).join(', ')})</span>}</dd>
                              <dt className="text-muted-foreground">Raw</dt>
                              <dd className="font-mono break-all text-muted-foreground">{r.raw}</dd>
                            </dl>
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  )
                })}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
