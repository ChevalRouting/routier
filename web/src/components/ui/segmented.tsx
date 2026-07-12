import { cn } from '@/lib/utils'

interface SegmentedProps<T extends string> {
  value: T
  onChange: (value: T) => void
  options: { value: T; label: string; activeClass?: string }[]
  className?: string
}

export function Segmented<T extends string>({ value, onChange, options, className }: SegmentedProps<T>) {
  return (
    <div className={cn('inline-flex h-8 overflow-hidden rounded-md border border-input text-sm', className)}>
      {options.map((o, i) => (
        <button
          key={o.value}
          type="button"
          onClick={() => onChange(o.value)}
          className={cn(
            'flex items-center px-3 transition-colors',
            i > 0 && 'border-l border-input',
            value === o.value ? (o.activeClass ?? 'bg-primary text-primary-foreground') : 'hover:bg-muted',
          )}
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}
