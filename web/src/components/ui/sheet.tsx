import * as Dialog from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface SheetProps {
  open: boolean
  onClose: () => void
  title: React.ReactNode
  children: React.ReactNode
  className?: string
}

export function Sheet({ open, onClose, title, children, className }: SheetProps) {
  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40" />
        <Dialog.Content
          aria-describedby={undefined}
          className={cn(
            'fixed right-0 top-0 z-40 flex h-full w-full max-w-xl flex-col bg-background border-l shadow-2xl focus:outline-none',
            className,
          )}
          onEscapeKeyDown={onClose}
          onInteractOutside={onClose}
        >
          <div className="flex items-center gap-3 px-5 py-4 border-b shrink-0">
            <Dialog.Title className="flex-1 font-mono font-semibold text-base truncate">
              {title}
            </Dialog.Title>
            <Dialog.Close asChild>
              <button type="button" className="text-muted-foreground hover:text-foreground shrink-0">
                <X className="h-5 w-5" />
              </button>
            </Dialog.Close>
          </div>
          <div className="flex-1 overflow-y-auto px-5 py-5 space-y-4">
            {children}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
