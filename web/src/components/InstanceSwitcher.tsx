import {
  getActiveInstance, subscribeInstance,
  type Instance
} from '@/lib/instance'
import { cn } from 'cheval-ui'
import { Check } from 'lucide-react'
import { useEffect, useState } from 'react'

export type Tone = 'sidebar' | 'sheet'

export function useActiveInstance(): Instance {
  const [inst, setInst] = useState(getActiveInstance)
  useEffect(() => subscribeInstance(setInst), [])
  return inst
}

export function dotColor(reachable?: boolean): string {
  if (reachable === undefined) return 'bg-muted-foreground/40'
  return reachable ? 'bg-success' : 'bg-danger'
}

export interface InstanceRowProps {
  tone: Tone
  icon: React.ReactNode
  label: string
  sub?: string
  active: boolean
  onClick: () => void
}

export function InstanceRow({ tone, icon, label, sub, active, onClick }: InstanceRowProps) {
  const tones = tone === 'sheet'
    ? {
        active: 'text-primary',
        idle: 'text-foreground/80 hover:bg-muted/60 hover:text-foreground',
      }
    : {
        active: 'text-sidebar-accent-foreground',
        idle: 'text-sidebar-foreground/70 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground',
      }

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex items-center gap-2 px-2.5 py-2 text-[13px] w-full text-left transition-colors',
        active ? tones.active : tones.idle,
      )}
    >
      {icon}
      <span className="truncate">{label}</span>
      {sub && <span className="truncate text-[11px] text-muted-foreground font-mono">{sub}</span>}
      {active && <Check className="ml-auto h-3.5 w-3.5 shrink-0" />}
    </button>
  )
}

export { InstanceMenu } from './InstanceSwitcherParts/InstanceMenu'
export { InstanceSwitcher } from './InstanceSwitcherParts/InstanceSwitcher'
