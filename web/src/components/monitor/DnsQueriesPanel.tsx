import { readEventStream } from '@/lib/eventStream'
import { Button, Card, CardContent, cn, Input, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'
import { ChevronDown, ChevronRight, Search, Trash2 } from 'lucide-react'
import { Fragment, useEffect, useMemo, useRef, useState } from 'react'

type RowShape = { id: number }

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

type Row = DnsQuery & RowShape

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

function parseQuery(data: string): DnsQuery | null {
  try {
    return JSON.parse(data) as DnsQuery
  } catch {
    return null
  }
}

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
    const addQuery = (data: string) => {
      const parsed = parseQuery(data)
      if (parsed) setRows((prev) => [{ ...parsed, id: counter++ }, ...prev].slice(0, 500))
    }

    const markLive = () => {
      connectedRef.current = true
      setStatus('Live')
    }

    const run = async () => {
      setStatus('Connecting…')
      const history = connectedRef.current ? '?history=0' : ''
      try {
        await readEventStream(`/api/dns/queries/stream${history}`, controller.signal, addQuery, markLive)
        throw new Error('Query stream disconnected')
      } catch (err) {
        if (!controller.signal.aborted) {
          setStatus(`${err instanceof Error ? err.message : 'Connection failed'}, retrying…`)
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
            <span className={cn('h-1.5 w-1.5 rounded-full inline-block', live && status === 'Live' ? 'bg-success' : 'bg-muted-foreground')} />
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
            <Table className="w-full text-sm">
              <TableHeader>
                <TableRow className="border-b border-border text-left">
                  <TableHead className="w-8" />
                  <TableHead className="px-3 py-2.5 font-medium text-muted-foreground w-28">Time</TableHead>
                  <TableHead className="px-3 py-2.5 font-medium text-muted-foreground">Client</TableHead>
                  <TableHead className="px-3 py-2.5 font-medium text-muted-foreground">Name</TableHead>
                  <TableHead className="px-3 py-2.5 font-medium text-muted-foreground w-20">Type</TableHead>
                  <TableHead className="px-3 py-2.5 font-medium text-muted-foreground w-16">Class</TableHead>
                  <TableHead className="px-3 py-2.5 font-medium text-muted-foreground w-24">Flags</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="px-4 py-8 text-center text-sm text-muted-foreground">
                      {q && rows.length > 0 ? 'No Results' : live ? 'Waiting for Queries' : 'Paused'}
                    </TableCell>
                  </TableRow>
                ) : filtered.map((r) => {
                  const open = expanded === r.id
                  return (
                    <Fragment key={r.id}>
                      <TableRow
                        className="border-b border-border last:border-b-0 hover:bg-muted/30 cursor-pointer"
                        onClick={() => setExpanded(open ? null : r.id)}
                      >
                        <TableCell className="pl-3 text-muted-foreground">
                          {open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
                        </TableCell>
                        <TableCell className="px-3 py-2 font-mono text-xs text-muted-foreground tabular-nums">{shortTime(r.time)}</TableCell>
                        <TableCell className="px-3 py-2 font-mono text-xs">{r.client}</TableCell>
                        <TableCell className="px-3 py-2 font-mono truncate max-w-xs">{r.name}</TableCell>
                        <TableCell className="px-3 py-2 font-mono text-xs">
                          <span className="inline-flex items-center rounded-full bg-primary/10 text-primary px-2 py-0.5">{r.type}</span>
                        </TableCell>
                        <TableCell className="px-3 py-2 font-mono text-xs text-muted-foreground">{r.class}</TableCell>
                        <TableCell className="px-3 py-2 font-mono text-xs text-muted-foreground">{r.flags}</TableCell>
                      </TableRow>
                      {open && (
                        <TableRow className="border-b border-border bg-muted/20">
                          <TableCell />
                          <TableCell colSpan={6} className="px-3 py-3">
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
                          </TableCell>
                        </TableRow>
                      )}
                    </Fragment>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
