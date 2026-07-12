import { useState, useEffect, useCallback, useMemo } from 'react'
import { api } from '@/lib/client'
import type { TypesProcessInfo as ProcessInfo } from '@/api'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Pagination, usePagination } from '@/components/Pagination'
import { Button } from '@/components/ui/button'
import { RefreshCw, Pause, Play, Search, ArrowUp, ArrowDown } from 'lucide-react'
import { Spinner } from '@/components/Spinner'
import { fmtBytes } from '@/lib/fmt'
import { cn } from '@/lib/utils'
import { OnActionChange } from './shared'

export function procStateLabel(s: string): string {
  const m: Record<string, string> = { R: 'Running', S: 'Sleeping', D: 'Waiting', Z: 'Zombie', T: 'Stopped', I: 'Idle' }
  return m[s] ?? s
}

function SortIcon({ field, sortField, sortDir }: { field: 'cpu_pct' | 'mem_rss'; sortField: string; sortDir: string }) {
  if (sortField !== field) return <ArrowUp className="h-3 w-3 opacity-30" />
  return sortDir === 'desc' ? <ArrowDown className="h-3 w-3" /> : <ArrowUp className="h-3 w-3" />
}

export function ProcessesPanel({ onActionChange }: { onActionChange: OnActionChange }) {
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
        <Button variant="outline" size="icon" className="h-8 w-8" onClick={fetchProcs}>
          <RefreshCw className="h-3.5 w-3.5" />
        </Button>
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
          className="max-w-xs h-8 text-sm"
        />
        {sorted && <span className="text-xs text-muted-foreground">{sorted.length} processes</span>}
      </div>
      {!sorted ? <Spinner /> : (
        <Card>
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left">
                    <th className="px-4 py-2.5 font-medium text-muted-foreground w-16">PID</th>
                    <th className="px-4 py-2.5 font-medium text-muted-foreground">Name</th>
                    <th className="px-4 py-2.5 font-medium text-muted-foreground">Command</th>
                    <th className="px-4 py-2.5 font-medium text-muted-foreground text-right w-24">
                      <button className="inline-flex items-center gap-1 ml-auto hover:text-foreground transition-colors" onClick={() => toggleSort('cpu_pct')}>
                        CPU% <SortIcon field="cpu_pct" sortField={sortField} sortDir={sortDir} />
                      </button>
                    </th>
                    <th className="px-4 py-2.5 font-medium text-muted-foreground text-right w-28">
                      <button className="inline-flex items-center gap-1 ml-auto hover:text-foreground transition-colors" onClick={() => toggleSort('mem_rss')}>
                        Memory <SortIcon field="mem_rss" sortField={sortField} sortDir={sortDir} />
                      </button>
                    </th>
                    <th className="px-4 py-2.5 font-medium text-muted-foreground">State</th>
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((p) => (
                    <tr key={p.pid} className="border-b last:border-b-0 hover:bg-muted/30">
                      <td className="px-4 py-2 font-mono text-xs text-muted-foreground">{p.pid}</td>
                      <td className="px-4 py-2 font-mono font-medium">{p.name}</td>
                      <td className="px-4 py-2 font-mono text-xs text-muted-foreground max-w-xs truncate">{p.cmd || '–'}</td>
                      <td className={cn(
                        'px-4 py-2 text-right font-mono tabular-nums text-xs',
                        p.cpu_pct > 50 ? 'text-danger' : p.cpu_pct > 20 ? 'text-warning' : '',
                      )}>
                        {p.cpu_pct.toFixed(1)}%
                      </td>
                      <td className="px-4 py-2 text-right font-mono tabular-nums text-xs">{fmtBytes(p.mem_rss)}</td>
                      <td className="px-4 py-2">
                        <span className={cn(
                          'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                          p.state === 'R' ? 'bg-success/15 text-success' :
                          p.state === 'Z' ? 'bg-danger/15 text-danger' :
                          'bg-muted text-muted-foreground',
                        )}>{procStateLabel(p.state)}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
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

