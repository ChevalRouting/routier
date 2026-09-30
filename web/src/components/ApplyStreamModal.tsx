import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { X, AlertTriangle, CheckCircle2, RefreshCw } from 'lucide-react'
import { Button } from 'cheval-ui'
import { TerminalPane } from 'cheval-ui/terminal'
import { api } from '@/lib/client'
import { getToken } from '@/lib/utils'

const DEFAULT_WATCHDOG = 60

type Phase = 'streaming' | 'completed' | 'interrupted' | 'reconnected' | 'confirming' | 'kept' | 'done' | 'rolledback' | 'failed'

const armed = (p: Phase) => p === 'completed' || p === 'reconnected'

async function lastApplyFailed(): Promise<boolean> {
  const logs = await api.apiApplyLogsGet()
  const latest = logs.records?.find((r) => r.source === 'web')
  return latest?.result === 'failed'
}

interface ApplyStreamModalProps {
  open: boolean
  onClose: () => void
}

export function ApplyStreamModal({ open, onClose }: ApplyStreamModalProps) {
  const [phase, setPhase] = useState<Phase>('streaming')
  const [remaining, setRemaining] = useState(DEFAULT_WATCHDOG)
  const [prevOpen, setPrevOpen] = useState(open)
  const phaseRef = useRef(phase)
  phaseRef.current = phase

  if (open !== prevOpen) {
    setPrevOpen(open)
    if (open) {
      setPhase('streaming')
      setRemaining(DEFAULT_WATCHDOG)
    }
  }

  useEffect(() => {
    if (phase !== 'interrupted' && !armed(phase)) return
    const id = setInterval(() => {
      setRemaining((r) => {
        if (r <= 1) {
          clearInterval(id)
          if (phaseRef.current !== 'kept' && phaseRef.current !== 'confirming') setPhase('rolledback')
          return 0
        }
        return r - 1
      })
    }, 1000)
    return () => clearInterval(id)
  }, [phase])

  useEffect(() => {
    if (phase !== 'interrupted') return
    let cancelled = false
    const poll = async () => {
      try {
        const p = await api.apiApplyPendingGet()
        if (cancelled || !p.pending) return
        setRemaining(p.remaining ?? DEFAULT_WATCHDOG)
        setPhase('reconnected')
      } catch {}
    }
    poll()
    const id = setInterval(poll, 2000)
    return () => { cancelled = true; clearInterval(id) }
  }, [phase])

  const onCompleted = async () => {
    try {
      const p = await api.apiApplyPendingGet()
      if (p.pending) {
        setRemaining(p.remaining ?? DEFAULT_WATCHDOG)
        setPhase('completed')
        return
      }
    } catch {
      setPhase('interrupted')
      return
    }

    try {
      setPhase((await lastApplyFailed()) ? 'failed' : 'done')
    } catch {
      setPhase('done')
    }
  }

  const handleClose = (clean: boolean) => {
    if (phaseRef.current !== 'streaming') return
    if (clean) onCompleted()
    else setPhase('interrupted')
  }

  const keepChanges = () => {
    setPhase('confirming')
    api.apiApplyConfirmPost()
      .then(() => setPhase('kept'))
      .catch(() => setPhase(armed(phaseRef.current) ? phaseRef.current : 'reconnected'))
  }

  if (!open) return null

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-3xl mx-4 bg-background rounded-xl shadow-2xl flex flex-col max-h-[80vh]">
        <div className="flex items-center justify-between px-5 py-3 shrink-0">
          <span className="font-semibold text-sm">Applying configuration…</span>
          <button type="button" onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex-1 min-h-0 overflow-hidden" style={{ minHeight: 320 }}>
          <TerminalPane wsPath="/api/ws/exec?name=reapply" token={getToken() ?? undefined} className="h-full" onClose={handleClose} quiet />
        </div>

        <StatusBar phase={phase} remaining={remaining} onKeep={keepChanges} onClose={onClose} />
      </div>
    </div>,
    document.body,
  )
}

function StatusBar({ phase, remaining, onKeep, onClose }: {
  phase: Phase; remaining: number; onKeep: () => void; onClose: () => void
}) {
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
