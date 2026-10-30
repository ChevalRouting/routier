import { cn, TableHead } from 'cheval-ui'
import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react'

interface SortableHeadProps {
  label: string
  active: boolean
  direction: 'asc' | 'desc'
  onSort: () => void
  className?: string
}

export function SortableHead({ label, active, direction, onSort, className }: SortableHeadProps) {
  const Icon = !active ? ArrowUpDown : direction === 'asc' ? ArrowDown : ArrowUp
  return (
    <TableHead className={className} aria-sort={active ? (direction === 'asc' ? 'ascending' : 'descending') : 'none'}>
      <button
        type="button"
        onClick={onSort}
        className={cn('inline-flex items-center gap-1 rounded hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring', active && 'text-foreground')}
      >
        {label}
        <Icon className={cn('h-3.5 w-3.5', !active && 'opacity-40')} />
      </button>
    </TableHead>
  )
}
