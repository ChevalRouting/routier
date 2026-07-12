import { useState } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

interface SectionItem<T extends string> {
  key: T
  label: string
  badge?: string | number
}

interface SectionNavProps<T extends string> {
  items: SectionItem<T>[]
  active: T
  onChange: (key: T) => void
  children: React.ReactNode
  className?: string
}

export function SectionNav<T extends string>({ items, active, onChange, children, className }: SectionNavProps<T>) {
  const [entered, setEntered] = useState(false)
  const activeLabel = items.find((i) => i.key === active)?.label

  return (
    <div className={cn('flex flex-col sm:flex-row sm:gap-6', className)}>
      <nav
        className={cn(
          'flex flex-col gap-0.5 sm:w-48 sm:shrink-0 sm:border-r sm:border-border sm:pr-3',
          entered && 'hidden sm:flex',
        )}
      >
        {items.map((it) => (
          <button
            key={it.key}
            type="button"
            onClick={() => { onChange(it.key); setEntered(true) }}
            className={cn(
              'flex items-center gap-2 rounded-md px-3 py-2 text-left text-sm font-medium transition-colors',
              active === it.key ? 'bg-accent text-foreground' : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground',
            )}
          >
            <span className="min-w-0 flex-1 truncate">{it.label}</span>
            {it.badge != null && (
              <span className="inline-flex items-center justify-center rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-semibold leading-none text-muted-foreground">
                {it.badge}
              </span>
            )}
            <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground sm:hidden" />
          </button>
        ))}
      </nav>

      <div className={cn('min-w-0 flex-1 pt-4 sm:pt-0', !entered && 'hidden sm:block')}>
        <button
          type="button"
          onClick={() => setEntered(false)}
          className="mb-3 flex items-center gap-1 text-sm font-medium text-muted-foreground hover:text-foreground sm:hidden"
        >
          <ChevronLeft className="h-4 w-4" />
          {activeLabel ?? 'Back'}
        </button>
        {children}
      </div>
    </div>
  )
}
