import { useEffect, useMemo, useState } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export function usePagination<T>(items: T[], pageSize = 12) {
  const [page, setPage] = useState(0)
  const totalPages = Math.max(1, Math.ceil(items.length / pageSize))
  const safePage = Math.min(page, totalPages - 1)

  useEffect(() => {
    if (page > totalPages - 1) setPage(totalPages - 1)
  }, [page, totalPages])

  const pageItems = useMemo(
    () => items.slice(safePage * pageSize, safePage * pageSize + pageSize),
    [items, safePage, pageSize],
  )

  return { page: safePage, setPage, totalPages, pageItems, total: items.length, pageSize }
}

interface PaginationProps {
  page: number
  totalPages: number
  total: number
  pageSize: number
  onPage: (page: number) => void
  unit?: string
  className?: string
}

export function Pagination({ page, totalPages, total, pageSize, onPage, unit = 'items', className }: PaginationProps) {
  if (totalPages <= 1) return null

  const from = page * pageSize + 1
  const to = Math.min((page + 1) * pageSize, total)

  return (
    <div className={cn('flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground', className)}>
      <span>
        {from.toLocaleString()}–{to.toLocaleString()} of {total.toLocaleString()} {unit}
      </span>
      <div className="flex items-center gap-2">
        <Button variant="outline" size="icon" className="h-7 w-7" disabled={page === 0} onClick={() => onPage(page - 1)} title="Previous page">
          <ChevronLeft className="h-4 w-4" />
        </Button>
        <span className="tabular-nums">
          Page {page + 1} / {totalPages}
        </span>
        <Button variant="outline" size="icon" className="h-7 w-7" disabled={page >= totalPages - 1} onClick={() => onPage(page + 1)} title="Next page">
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
