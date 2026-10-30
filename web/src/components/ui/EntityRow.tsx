import { Button, cn } from 'cheval-ui'
import { ChevronRight, Trash2 } from 'lucide-react'

interface EntityRowProps {
  title?: React.ReactNode
  placeholder?: string
  mono?: boolean
  children?: React.ReactNode
  onOpen: () => void
  onRemove?: () => void
  removeLabel?: string
}

export function EntityRow({ title, placeholder = 'Unnamed', mono = true, children, onOpen, onRemove, removeLabel = 'Remove' }: EntityRowProps) {
  return (
    <div className="flex items-center gap-1 pr-2 transition-colors hover:bg-accent/50">
      <button
        type="button"
        onClick={onOpen}
        className="flex min-h-[3.25rem] min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1.5 px-4 py-2.5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
      >
        <span className={cn('w-40 max-w-full shrink-0 truncate text-sm font-semibold', mono && 'font-mono')}>
          {title || <span className="font-normal text-muted-foreground">{placeholder}</span>}
        </span>
        <span className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">{children}</span>
        <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
      </button>
      {onRemove && (
        <Button variant="ghost" size="icon" onClick={onRemove} aria-label={removeLabel} title={removeLabel} className="shrink-0 text-muted-foreground hover:text-destructive">
          <Trash2 className="h-4 w-4" />
        </Button>
      )}
    </div>
  )
}
