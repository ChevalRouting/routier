import { SortableHead } from '@/components/ui/SortableHead'
import type { TypesProcessInfo as ProcessInfo } from '@/api'
import { api } from '@/lib/client'
import { plural } from '@/lib/utils'
import { Button, Card, CardContent, cn, fmtBytes, Input, Pagination, Spinner, usePagination, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, StatusChip, ReloadButton } from 'cheval-ui'
import { Pause, Play, Search } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { OnActionChange } from './shared'

type ProcessesPanelShape = { onActionChange: OnActionChange }

export function procStateLabel(s: string): string {
  const m: Record<string, string> = { R: 'Running', S: 'Sleeping', D: 'Waiting', Z: 'Zombie', T: 'Stopped', I: 'Idle' }
  return m[s] ?? s
}


export function ProcessesPanel({ onActionChange }: ProcessesPanelShape) {
  const [procs, setProcs] = useState<ProcessInfo[] | null>(null)
  const [paused, setPaused] = useState(false)
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null)
  const [q, setQ] = useState('')
  const [sortField, setSortField] = useState<'cpu_pct' | 'mem_rss'>('cpu_pct')
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')

  const fetchProcs = useCallback(async () => {
    try {
      const data = await api.apiStatsProcessesGet()
      setProcs(data.processes ?? [])
      setLastUpdate(new Date())
    } catch {}
  }, [])

  useEffect(() => { fetchProcs() }, [fetchProcs])
  useEffect(() => {
    if (paused) return
    const id = setInterval(fetchProcs, 5000)
    return () => clearInterval(id)
  }, [paused, fetchProcs])

  useEffect(() => {
    onActionChange(
      <div className="flex items-center gap-2">
        {lastUpdate && <span className="text-xs text-muted-foreground hidden sm:inline">{lastUpdate.toLocaleTimeString()}</span>}
        <Button variant="outline" size="sm" onClick={() => setPaused((p) => !p)} className="gap-1.5">
          {paused ? <Play className="h-3.5 w-3.5" /> : <Pause className="h-3.5 w-3.5" />}
          {paused ? 'Resume' : 'Pause'}
        </Button>
        <ReloadButton onClick={fetchProcs} />
      </div>
    )
  }, [paused, lastUpdate, fetchProcs, onActionChange])

  const sorted = useMemo(() => {
    if (!procs) return null
    let list = procs
    if (q) {
      const lq = q.toLowerCase()
      list = list.filter((p) => p.name.toLowerCase().includes(lq) || (p.cmd ?? '').toLowerCase().includes(lq))
    }
    return [...list].sort((a, b) => {
      const diff = a[sortField] - b[sortField]
      return sortDir === 'desc' ? -diff : diff
    })
  }, [procs, q, sortField, sortDir])

  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(sorted ?? [], 25)

  const toggleSort = (field: 'cpu_pct' | 'mem_rss') => {
    if (sortField === field) setSortDir((d) => (d === 'desc' ? 'asc' : 'desc'))
    else { setSortField(field); setSortDir('desc') }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Search className="h-4 w-4 text-muted-foreground shrink-0" />
        <Input
          placeholder="Filter processes…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          className="h-8 min-w-0 flex-1 text-sm sm:max-w-xs"
        />
        {sorted && <span className="shrink-0 whitespace-nowrap text-xs text-muted-foreground">{plural(sorted.length, 'Process', 'Processes')}</span>}
      </div>
      {!sorted ? <Spinner /> : (
        <Card>
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <Table className="w-full text-sm">
                <TableHeader>
                  <TableRow className="border-b border-border text-left">
                    <TableHead className="px-4 py-2.5 font-medium text-muted-foreground w-16">PID</TableHead>
                    <TableHead className="px-4 py-2.5 font-medium text-muted-foreground">Name</TableHead>
                    <TableHead className="px-4 py-2.5 font-medium text-muted-foreground">Command</TableHead>
                    <SortableHead label="CPU %" active={sortField === 'cpu_pct'} direction={sortDir} onSort={() => toggleSort('cpu_pct')} className="w-24 px-4 py-2.5 text-right" />
                    <SortableHead label="Memory" active={sortField === 'mem_rss'} direction={sortDir} onSort={() => toggleSort('mem_rss')} className="w-28 px-4 py-2.5 text-right" />
                    <TableHead className="px-4 py-2.5 font-medium text-muted-foreground">State</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pageItems.map((p) => (
                    <TableRow key={p.pid} className="border-b border-border last:border-b-0 hover:bg-muted/30">
                      <TableCell className="px-4 py-2 font-mono text-xs text-muted-foreground">{p.pid}</TableCell>
                      <TableCell data-title className="px-4 py-2 font-mono font-medium">{p.name}</TableCell>
                      <TableCell data-wide className="px-4 py-2 font-mono text-xs text-muted-foreground max-w-xs truncate">{p.cmd || '–'}</TableCell>
                      <TableCell className={cn(
                        'px-4 py-2 text-right font-mono tabular-nums text-xs',
                        p.cpu_pct > 50 ? 'text-danger' : p.cpu_pct > 20 ? 'text-warning' : '',
                      )}>
                        {p.cpu_pct.toFixed(1)}%
                      </TableCell>
                      <TableCell className="px-4 py-2 text-right font-mono tabular-nums text-xs">{fmtBytes(p.mem_rss)}</TableCell>
                      <TableCell className="px-4 py-2">
                        <StatusChip intent={p.state === 'R' ? 'success' : p.state === 'Z' ? 'danger' : 'neutral'} label={procStateLabel(p.state)} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      )}
      {sorted && totalPages > 1 && (
        <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="processes" />
      )}
    </div>
  )
}

