import { cn } from '@/lib/utils'

export type StatusIntent = 'success' | 'warning' | 'danger' | 'info' | 'neutral'

interface StatusChipProps {
  intent: StatusIntent
  label: string
  pulse?: boolean
  className?: string
}

const CHIP_STYLES: Record<StatusIntent, string> = {
  success: 'bg-success/12 text-success border-success/25',
  warning: 'bg-warning/12 text-warning border-warning/25',
  danger:  'bg-danger/12  text-danger  border-danger/25',
  info:    'bg-info/12    text-info    border-info/25',
  neutral: 'bg-muted/60   text-muted-foreground border-border',
}

const DOT_STYLES: Record<StatusIntent, string> = {
  success: 'bg-success',
  warning: 'bg-warning',
  danger:  'bg-danger',
  info:    'bg-info',
  neutral: 'bg-muted-foreground/60',
}

export function StatusChip({ intent, label, pulse = false, className }: StatusChipProps) {
  return (
    <span className={cn(
      'inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full border text-[11px] font-semibold tracking-wide',
      CHIP_STYLES[intent],
      className,
    )}>
      <span className={cn(
        'h-1.5 w-1.5 rounded-full shrink-0',
        DOT_STYLES[intent],
        pulse && 'animate-pulse',
      )} />
      {label}
    </span>
  )
}

export function intentForState(state: string): StatusIntent {
  switch (state.toUpperCase()) {
    case 'REACHABLE':
    case 'MASTER':
    case 'ESTABLISHED':
    case 'UP':
    case 'RUNNING':
    case 'ACTIVE':
    case 'PERMANENT':
    case 'NOARP':
      return 'success'

    case 'STALE':
    case 'DELAY':
    case 'PROBE':
    case 'BACKUP':
    case 'DEGRADED':
    case 'WARN':
      return 'warning'

    case 'FAILED':
    case 'FAULT':
    case 'DOWN':
    case 'STOPPED':
    case 'ERROR':
    case 'INCOMPLETE':
      return 'danger'

    case 'INFO':
    case 'CONNECTING':
      return 'info'

    default:
      return 'neutral'
  }
}

export function StateChip({ state, pulse }: { state: string; pulse?: boolean }) {
  const intent = intentForState(state)
  const shouldPulse = pulse ?? (intent === 'success' && ['MASTER', 'REACHABLE', 'ESTABLISHED', 'UP', 'RUNNING', 'ACTIVE'].includes(state.toUpperCase()))
  return <StatusChip intent={intent} label={state} pulse={shouldPulse} />
}
