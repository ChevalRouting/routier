import type { TypesLLDPNeighbor as LLDPNeighbor, TypesNeighbor as Neighbor } from '@/api'
import { LLDPTable, NeighborTable } from '@/components/monitor/NeighborsPanel'
import { api } from '@/lib/client'
import { Button, Input, Spinner } from 'cheval-ui'
import { RefreshCw, Search } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'

export function NeighborsPanel() {
  const [neighbors, setNeighbors] = useState<Neighbor[] | null>(null)
  const [lldp, setLldp] = useState<LLDPNeighbor[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const [query, setQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')

  useEffect(() => { const id = setTimeout(() => setDebouncedQuery(query), 300); return () => clearTimeout(id) }, [query])

  const refresh = useCallback(async () => {
    try {
      const [result, lldpResult] = await Promise.all([
        api.apiRoutingNeighborsGet({ q: debouncedQuery || undefined }),
        api.apiRoutingLldpGet({ q: debouncedQuery || undefined }),
      ])
      setNeighbors(result.neighbors ?? []); setLldp(lldpResult.neighbors ?? []); setError(null); setLastUpdated(new Date())
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
          <LLDPTable label={`LLDP / CDP (${(lldp ?? []).length})`} rows={lldp ?? []} />
        </div>
      )}
    </div>
  )
}
