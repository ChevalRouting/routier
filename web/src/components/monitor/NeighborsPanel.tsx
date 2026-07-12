import { useState, useEffect, useCallback, useMemo } from 'react'
import { api } from '@/lib/client'
import type { TypesNeighbor as Neighbor } from '@/api'
import { Input } from '@/components/ui/input'
import { StateChip } from '@/components/ui/status-chip'
import { SectionLabel } from '@/components/SectionLabel'
import { Pagination, usePagination } from '@/components/Pagination'
import { Button } from '@/components/ui/button'
import { RefreshCw, Search } from 'lucide-react'
import { Spinner } from '@/components/Spinner'
import { EmptyState } from '@/components/EmptyState'

export function NeighborsPanel() {
  const [neighbors, setNeighbors] = useState<Neighbor[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const [query, setQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')

  useEffect(() => { const id = setTimeout(() => setDebouncedQuery(query), 300); return () => clearTimeout(id) }, [query])

  const refresh = useCallback(async () => {
    try {
      const result = await api.apiRoutingNeighborsGet({ q: debouncedQuery || undefined })
      setNeighbors(result.neighbors ?? []); setError(null); setLastUpdated(new Date())
    } catch (e) { setError((e as Error).message) }
    finally { setLoading(false) }
  }, [debouncedQuery])

  useEffect(() => { refresh(); const id = setInterval(refresh, 10000); return () => clearInterval(id) }, [refresh])

  const split = useMemo(() => {
    if (!neighbors) return { v4: [], v6: [] }
    return { v4: neighbors.filter((n) => n.family === 'ipv4'), v6: neighbors.filter((n) => n.family === 'ipv6') }
  }, [neighbors])

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/50" />
          <Input placeholder="Filter by IP, MAC, interface, state…" value={query} onChange={(e) => setQuery(e.target.value)} className="pl-8 h-8 text-xs" />
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {lastUpdated && <span className="text-xs text-muted-foreground hidden sm:inline">{lastUpdated.toLocaleTimeString()}</span>}
          <Button variant="outline" size="sm" onClick={refresh} disabled={loading} className="gap-1.5">
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />Refresh
          </Button>
        </div>
      </div>

      {error && <div className="rounded-md border border-danger/30 bg-danger/8 px-4 py-3 text-sm text-danger">{error}</div>}

      {loading && !neighbors ? <Spinner /> : (
        <div className="space-y-5">
          <NeighborTable label={`IPv4 - ARP (${split.v4.length})`} rows={split.v4} />
          <NeighborTable label={`IPv6 - NDP (${split.v6.length})`} rows={split.v6} />
        </div>
      )}
    </div>
  )
}

export function NeighborTable({ label, rows }: { label: string; rows: Neighbor[] }) {
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(rows, 25)
  return (
    <div className="space-y-2">
      <SectionLabel>{label}</SectionLabel>
      {rows.length === 0
        ? <EmptyState className="py-8" title="No neighbors" message="Discovered L2/L3 neighbors will show up here." />
        : (
          <>
            <div className="rounded-xl border border-border overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border/60">
                    {['Address', 'Interface', 'MAC / Link-layer', 'State'].map((h) => (
                      <th key={h} className="px-4 py-2.5 text-left text-xs font-semibold text-muted-foreground">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((n, i) => (
                    <tr key={`${n.dst}-${n.dev}-${i}`} className="border-b border-border/40 last:border-0 hover:bg-accent/20">
                      <td className="px-4 py-2 font-mono text-xs">{n.dst}</td>
                      <td className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.dev}</td>
                      <td className="px-4 py-2 font-mono text-xs text-muted-foreground">{n.lladdr ?? '-'}</td>
                      <td className="px-4 py-2"><StateChip state={n.state} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="entries" />
          </>
        )}
    </div>
  )
}

