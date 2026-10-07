import { Phase } from '@/components/ApplyStreamModal'
import { Button } from 'cheval-ui'
import { AlertTriangle, CheckCircle2, RefreshCw } from 'lucide-react'

type StatusBarShape = {
  phase: Phase; remaining: number; onKeep: () => void; onClose: () => void
}

export function StatusBar({ phase, remaining, onKeep, onClose }: StatusBarShape) {
  if (phase === 'streaming') return null

  if (phase === 'kept' || phase === 'done') {
    return (
      <div className="flex items-center gap-2 px-5 pb-4 pt-2 shrink-0 text-sm text-success">
        <CheckCircle2 className="h-4 w-4" />
        {phase === 'kept' ? 'Changes applied and kept.' : 'Apply finished.'}
        <Button size="sm" variant="outline" className="ml-auto" onClick={onClose}>Close</Button>
      </div>
    )
  }

  if (phase === 'rolledback') {
    return (
      <div className="flex items-center gap-2 px-5 pb-4 pt-2 shrink-0 text-sm text-danger">
        <AlertTriangle className="h-4 w-4" /> Not confirmed in time, configuration was rolled back.
        <Button size="sm" variant="outline" className="ml-auto" onClick={onClose}>Close</Button>
      </div>
    )
  }

  if (phase === 'failed') {
    return (
      <div className="flex items-center gap-2 px-5 pb-4 pt-2 shrink-0 text-sm text-danger">
        <AlertTriangle className="h-4 w-4" /> Configuration failed to apply. Check the log above for details.
        <Button size="sm" variant="outline" className="ml-auto" onClick={onClose}>Close</Button>
      </div>
    )
  }

  if (phase === 'confirming') {
    return (
      <div className="flex items-center gap-2 px-5 pb-4 pt-2 shrink-0 text-sm text-muted-foreground">
        <RefreshCw className="h-4 w-4 animate-spin" /> Confirming changes…
      </div>
    )
  }

  const interrupted = phase === 'interrupted'
  const reconnected = phase === 'reconnected'
  return (
    <div className="flex items-center gap-3 px-5 pb-4 pt-2 shrink-0 text-sm">
      <AlertTriangle className="h-4 w-4 text-warning shrink-0" />
      <div className="min-w-0">
        <div className="font-medium text-warning">
          {interrupted ? 'Connection interrupted' : 'Confirm to keep changes'}
        </div>
        <div className="text-muted-foreground">
          {interrupted
            ? 'Waiting to reconnect, automatic rollback if not confirmed.'
            : reconnected
              ? 'Reconnected. Keep the changes to disarm the automatic rollback.'
              : 'Verify connectivity, then keep the changes to disarm the automatic rollback.'}
        </div>
      </div>
      <div className="ml-auto flex items-center gap-3">
        <span className="font-mono tabular-nums text-lg font-semibold">{remaining}s</span>
        {interrupted
          ? <RefreshCw className="h-4 w-4 animate-spin text-muted-foreground" />
          : <Button size="sm" onClick={onKeep}>Keep changes</Button>}
      </div>
    </div>
  )
}
