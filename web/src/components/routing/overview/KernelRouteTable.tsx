import { PROTO_FILTER } from './graph'
import { useState, useEffect } from 'react'
import { api } from '@/lib/client'
import type { TypesKernelRoute as KernelRoute } from '@/api'
import { protoColor } from '@/lib/palette'
import { cn } from '@/lib/utils'
import { Pagination } from '@/components/Pagination'
import '@xyflow/react/dist/style.css'

export const ROUTE_PAGE_SIZE = 100

export function KernelRouteTable({ proto }: { proto: string }) {
  const [page, setPage] = useState(0)
  const [data, setData] = useState<{ routes: KernelRoute[]; total: number }>({ routes: [], total: 0 })
  const [loading, setLoading] = useState(true)
  const protoParam = (PROTO_FILTER[proto] ?? []).join(',')

  useEffect(() => { setPage(0) }, [protoParam])
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    api.apiRoutingRoutesGet({ proto: protoParam || undefined, limit: ROUTE_PAGE_SIZE, offset: page * ROUTE_PAGE_SIZE })
      .then((d) => { if (!cancelled) setData({ routes: d.routes ?? [], total: d.total ?? 0 }) })
      .catch(() => { if (!cancelled) setData({ routes: [], total: 0 }) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [protoParam, page])

  const { routes, total } = data
  const totalPages = Math.max(1, Math.ceil(total / ROUTE_PAGE_SIZE))

  if (!loading && total === 0) {
    return <p className="px-2 py-4 text-xs text-muted-foreground">No kernel routes for this selection.</p>
  }

  return (
    <div className="space-y-2">
      <div className="max-h-96 overflow-auto rounded-xl border border-border">
        <table className="w-full text-xs font-mono">
          <thead className="sticky top-0 z-10 bg-card">
            <tr className="border-b border-border text-muted-foreground">
              {['Destination', 'Gateway', 'Dev', 'Protocol', 'Metric', 'Family'].map((h) => (
                <th key={h} className="px-3 py-2 text-left font-medium">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody className={cn('transition-opacity', loading && 'opacity-50')}>
            {routes.map((r, i) => (
              <tr key={i} className="border-b border-border/40 last:border-0 hover:bg-muted/30">
                <td className="px-3 py-1">{r.dst}</td>
                <td className="px-3 py-1 text-muted-foreground">{r.gateway || '-'}</td>
                <td className="px-3 py-1 text-muted-foreground">{r.dev || '-'}</td>
                <td className="px-3 py-1">
                  <span className="inline-block rounded px-1.5 py-0.5 text-[10px] text-white" style={{ background: protoColor(r.protocol) }}>
                    {r.protocol}
                  </span>
                </td>
                <td className="px-3 py-1">{r.metric || '-'}</td>
                <td className="px-3 py-1 text-muted-foreground">{r.family}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {totalPages > 1 ? (
        <Pagination page={page} totalPages={totalPages} total={total} pageSize={ROUTE_PAGE_SIZE} onPage={setPage} unit="routes" />
      ) : (
        <p className="text-xs text-muted-foreground">{total.toLocaleString()} route{total !== 1 ? 's' : ''}</p>
      )}
    </div>
  )
}

