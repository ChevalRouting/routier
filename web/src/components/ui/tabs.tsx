import { cn } from '@/lib/utils'

interface TabItem<T extends string> {
  key: T
  label: string
  badge?: string | number
}

interface TabsProps<T extends string> {
  tabs: TabItem<T>[]
  active: T
  onChange: (key: T) => void
  variant?: 'underline' | 'pills'
  className?: string
}

export function Tabs<T extends string>({
  tabs,
  active,
  onChange,
  variant = 'underline',
  className,
}: TabsProps<T>) {
  if (variant === 'pills') {
    return (
      <div className={cn('flex gap-0.5 bg-muted rounded-md p-0.5 overflow-x-auto scrollbar-none', className)}>
        {tabs.map((t) => (
          <button
            key={t.key}
            type="button"
            onClick={() => onChange(t.key)}
            className={cn(
              'flex shrink-0 items-center gap-1.5 whitespace-nowrap px-3 py-1.5 rounded text-xs font-medium transition-colors',
              active === t.key
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            {t.label}
            {t.badge != null && (
              <span className={cn(
                'inline-flex items-center justify-center rounded-full px-1.5 py-0.5 text-[10px] font-semibold leading-none min-w-[1.125rem]',
                active === t.key ? 'bg-muted text-muted-foreground' : 'bg-muted-foreground/20 text-muted-foreground',
              )}>
                {t.badge}
              </span>
            )}
          </button>
        ))}
      </div>
    )
  }

  return (
    <div className={cn('flex gap-1 border-b border-border overflow-x-auto scrollbar-none', className)}>
      {tabs.map((t) => (
        <button
          key={t.key}
          type="button"
          onClick={() => onChange(t.key)}
          className={cn(
            'flex items-center gap-1.5 whitespace-nowrap px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors shrink-0',
            active === t.key
              ? 'border-primary text-foreground'
              : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border',
          )}
        >
          {t.label}
          {t.badge != null && (
            <span className={cn(
              'inline-flex items-center justify-center rounded-full px-1.5 py-0.5 text-[10px] font-semibold leading-none min-w-[1.125rem]',
              active === t.key ? 'bg-primary/15 text-primary' : 'bg-muted text-muted-foreground',
            )}>
              {t.badge}
            </span>
          )}
        </button>
      ))}
    </div>
  )
}
