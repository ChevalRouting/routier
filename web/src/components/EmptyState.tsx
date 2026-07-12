import * as React from 'react'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  icon?: React.ReactNode
  title?: string
  message: string
  action?: React.ReactNode
  className?: string
}

export function EmptyState({ icon, title, message, action, className }: EmptyStateProps) {
  return (
    <div className={cn(
      'flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border/60 px-6 py-16 text-center',
      className,
    )}>
      {icon && (
        <div className="text-muted-foreground/40 [&_svg]:h-12 [&_svg]:w-12">{icon}</div>
      )}
      {title && <p className="text-base font-semibold text-foreground">{title}</p>}
      <p className="max-w-sm text-sm text-muted-foreground">{message}</p>
      {action}
    </div>
  )
}
