import { Phase } from '@/components/ApplyStreamModal'
import { Button, Spinner } from 'cheval-ui'
import { AlertTriangle, CheckCircle2 } from 'lucide-react'

type StatusBarShape = {
  phase: Phase; remaining: number; onKeep: () => void; onClose: () => void
}

export function StatusBar({ phase, remaining, onKeep, onClose }: StatusBarShape) {
  if (phase === 'streaming') return null

  if (phase === 'kept' || phase === 'done') {
    return (
      <div className="flex items-center gap-2 w-full text-sm text-success">
        <CheckCircle2 className="h-4 w-4" />
        {phase === 'kept' ? 'Changes kept' : 'Applied'}
        <Button size="sm" variant="outline" className="ml-auto" onClick={onClose}>Close</Button>
      </div>
    )
  }

  if (phase === 'rolledback') {
    return (
      <div className="flex items-center gap-2 w-full text-sm text-danger">
        <AlertTriangle className="h-4 w-4" /> Not confirmed in time, rolled back
        <Button size="sm" variant="outline" className="ml-auto" onClick={onClose}>Close</Button>
      </div>
    )
  }

  if (phase === 'failed') {
    return (
      <div className="flex items-center gap-2 w-full text-sm text-danger">
        <AlertTriangle className="h-4 w-4" /> Apply failed, see the log above
        <Button size="sm" variant="outline" className="ml-auto" onClick={onClose}>Close</Button>
      </div>
    )
  }

  if (phase === 'confirming') {
    return (
      <div className="flex items-center gap-2 w-full text-sm text-muted-foreground">
        <Spinner size="sm" /> Confirming changes…
      </div>
    )
  }

  const interrupted = phase === 'interrupted'
  const reconnected = phase === 'reconnected'
  return (
    <div className="flex items-center gap-3 w-full text-sm">
      <AlertTriangle className="h-4 w-4 text-warning shrink-0" />
      <div className="min-w-0">
        <div className="font-medium text-warning">
          {interrupted ? 'Connection Interrupted' : 'Confirm Changes'}
        </div>
        <div className="text-muted-foreground">
          {interrupted
            ? 'Waiting to reconnect; rolls back unless confirmed'
            : reconnected
              ? 'Reconnected; keep the changes to stop the rollback'
              : 'Keep the changes to stop the automatic rollback'}
        </div>
      </div>
      <div className="ml-auto flex items-center gap-3">
        <span className="font-mono tabular-nums text-lg font-semibold">{remaining}s</span>
        {interrupted
          ? <Spinner size="sm" />
          : <Button size="sm" onClick={onKeep}>Keep Changes</Button>}
      </div>
    </div>
  )
}
