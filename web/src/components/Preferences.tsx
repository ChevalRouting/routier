import * as React from 'react'
import { ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Switch } from '@/components/ui/switch'

export function PreferencesGroup({
  title,
  description,
  header,
  children,
  className,
}: {
  title?: string
  description?: string
  header?: React.ReactNode
  children: React.ReactNode
  className?: string
}) {
  return (
    <section className={cn('space-y-2', className)}>
      {(title || description || header) && (
        <div className="flex items-end justify-between gap-3 px-1">
          <div className="min-w-0">
            {title && <h2 className="text-sm font-semibold text-foreground">{title}</h2>}
            {description && <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>}
          </div>
          {header && <div className="shrink-0">{header}</div>}
        </div>
      )}
      <div className="overflow-hidden rounded-xl border border-border bg-card divide-y divide-border">
        {children}
      </div>
    </section>
  )
}

export function PreferencesGroups({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return (
    <div className={cn('[column-gap:1.5rem] lg:columns-2 [&>*]:mb-6 [&>*]:break-inside-avoid', className)}>
      {children}
    </div>
  )
}

export function PreferencesColumns({
  title,
  description,
  header,
  children,
  className,
}: {
  title?: string
  description?: string
  header?: React.ReactNode
  children: React.ReactNode
  className?: string
}) {
  return (
    <section className="space-y-2">
      {(title || description || header) && (
        <div className="flex items-end justify-between gap-3 px-1">
          <div className="min-w-0">
            {title && <h2 className="text-sm font-semibold text-foreground">{title}</h2>}
            {description && <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>}
          </div>
          {header && <div className="shrink-0">{header}</div>}
        </div>
      )}
      <div
        className={cn(
          'grid gap-3 grid-cols-1 lg:grid-cols-2 items-start [&>*]:overflow-hidden [&>*]:rounded-xl [&>*]:border [&>*]:border-border [&>*]:bg-card',
          className,
        )}
      >
        {children}
      </div>
    </section>
  )
}

interface RowProps {
  title?: React.ReactNode
  subtitle?: React.ReactNode
  prefix?: React.ReactNode
  children?: React.ReactNode
  onClick?: () => void
  className?: string
}

export function Row({ title, subtitle, prefix, children, onClick, className }: RowProps) {
  const body = (
    <>
      {prefix && <span className="shrink-0 text-muted-foreground">{prefix}</span>}
      <div className="min-w-0 flex-1">
        {title != null && <div className="text-sm font-medium text-foreground break-words">{title}</div>}
        {subtitle != null && <div className="mt-0.5 text-xs text-muted-foreground break-words">{subtitle}</div>}
      </div>
      {children != null && (
        <div className="flex shrink-0 flex-wrap items-center gap-2 max-sm:w-full">{children}</div>
      )}
    </>
  )
  const base = cn('flex min-h-[3.25rem] gap-x-3 gap-y-1.5 px-4 py-2.5', className)
  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        className={cn(base, 'w-full items-center text-left transition-colors hover:bg-accent/50')}
      >
        {body}
        <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
      </button>
    )
  }
  return <div className={cn(base, 'flex-col items-start sm:flex-row sm:items-center')}>{body}</div>
}

export function ComboRow({ title, subtitle, children }: { title: React.ReactNode; subtitle?: React.ReactNode; children: React.ReactNode }) {
  return (
    <Row title={title} subtitle={subtitle}>
      <div className="w-full sm:w-44">{children}</div>
    </Row>
  )
}

export const EntryRow = React.forwardRef<
  HTMLInputElement,
  { title: string; suffix?: React.ReactNode; error?: string | null } & React.InputHTMLAttributes<HTMLInputElement>
>(({ title, suffix, error, className, ...props }, ref) => (
  <div className="flex flex-col gap-0.5 px-4 py-2">
    <label htmlFor={props.id} className={cn('text-xs font-medium', error ? 'text-destructive' : 'text-muted-foreground')}>
      {title}
    </label>
    <div className="flex items-center gap-2">
      <input
        ref={ref}
        aria-invalid={error ? true : undefined}
        className={cn(
          'w-full bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground/50',
          className,
        )}
        {...props}
      />
      {suffix && <div className="flex shrink-0 items-center gap-1">{suffix}</div>}
    </div>
    {error && <p className="text-xs text-destructive">{error}</p>}
  </div>
))
EntryRow.displayName = 'EntryRow'

export function SwitchRow({
  title,
  subtitle,
  checked,
  onCheckedChange,
  disabled,
}: {
  title: React.ReactNode
  subtitle?: React.ReactNode
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => onCheckedChange(!checked)}
      className="flex min-h-[3.25rem] w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-accent/50 disabled:opacity-50"
    >
      <div className="min-w-0 flex-1">
        <div className="text-sm font-medium text-foreground break-words">{title}</div>
        {subtitle != null && <div className="mt-0.5 text-xs text-muted-foreground break-words">{subtitle}</div>}
      </div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} disabled={disabled} />
    </button>
  )
}
