import type { TypesKernelRoute as KernelRoute } from '@/api'
import { api } from '@/lib/client'
import { protoColor } from '@/lib/palette'
import '@xyflow/react/dist/style.css'
import { cn, Pagination, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { PROTO_FILTER } from './graph'

type KernelRouteTableShape2 = { proto: string }

type KernelRouteTableShape = { routes: KernelRoute[]; total: number }

export const ROUTE_PAGE_SIZE = 100

export function KernelRouteTable({ proto }: KernelRouteTableShape2) {
  const [page, setPage] = useState(0)
  const [data, setData] = useState<KernelRouteTableShape>({ routes: [], total: 0 })
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
    return <p className="px-2 py-4 text-xs text-muted-foreground">No kernel routes</p>
  }

  return (
    <div className="space-y-2">
      <div className="max-h-96 overflow-auto rounded-xl bg-card shadow-[var(--card-shadow)]">
        <Table className="w-full text-xs font-mono">
          <TableHeader className="sticky top-0 z-10 bg-card">
            <TableRow className="border-b border-border text-muted-foreground">
              {['Destination', 'Gateway', 'Dev', 'Protocol', 'Metric', 'Family'].map((h) => (
                <TableHead key={h} className="px-3 py-2 text-left font-medium">{h}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody className={cn('transition-opacity', loading && 'opacity-50')}>
            {routes.map((r, i) => (
              <TableRow key={i} className="border-b border-border/40 last:border-0 hover:bg-muted/30">
                <TableCell className="px-3 py-1">{r.dst}</TableCell>
                <TableCell className="px-3 py-1 text-muted-foreground">{r.gateway || '-'}</TableCell>
                <TableCell className="px-3 py-1 text-muted-foreground">{r.dev || '-'}</TableCell>
                <TableCell className="px-3 py-1">
                  <span className="inline-flex items-center gap-1.5 text-xs">
                    <span className="h-2 w-2 rounded-full" style={{ background: protoColor(r.protocol) }} />
                    {r.protocol}
                  </span>
                </TableCell>
                <TableCell className="px-3 py-1">{r.metric || '-'}</TableCell>
                <TableCell className="px-3 py-1 text-muted-foreground">{r.family}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {totalPages > 1 ? (
        <Pagination page={page} totalPages={totalPages} total={total} pageSize={ROUTE_PAGE_SIZE} onPage={setPage} unit="routes" />
      ) : (
        <p className="text-xs text-muted-foreground">{total.toLocaleString()} route{total !== 1 ? 's' : ''}</p>
      )}
    </div>
  )
}
