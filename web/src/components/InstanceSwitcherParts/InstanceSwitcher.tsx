import { InstanceMenu, useActiveInstance } from '@/components/InstanceSwitcher'
import { cn } from 'cheval-ui'
import { ChevronsUpDown, Handshake, Server } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

type InstanceSwitcherShape = { collapsed: boolean }

export function InstanceSwitcher({ collapsed }: InstanceSwitcherShape) {
  const active = useActiveInstance()
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false) }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const isFriend = active.id !== 'self'
  const Icon = isFriend ? Handshake : Server

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        title={collapsed ? active.label : undefined}
        className={cn(
          'flex items-center rounded-md text-[13px] font-medium transition-colors w-full',
          'text-sidebar-foreground/80 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground',
          isFriend && 'bg-info/15 text-info hover:bg-info/20 hover:text-info',
          collapsed ? 'justify-center py-2 h-9' : 'gap-2 px-2.5 py-1.5',
        )}
      >
        <Icon className="h-3.5 w-3.5 shrink-0" />
        {!collapsed && (
          <>
            <span className="truncate">{active.label}</span>
            <ChevronsUpDown className="ml-auto h-3 w-3 opacity-50" />
          </>
        )}
      </button>

      {open && (
        <div
          className={cn(
            'absolute z-50 mt-1 rounded-md border border-sidebar-border bg-sidebar shadow-lg',
            collapsed ? 'left-full ml-1 top-0 w-48' : 'left-0 right-0',
          )}
        >
          <InstanceMenu onDone={() => setOpen(false)} />
        </div>
      )}
    </div>
  )
}
