import { api } from '@/lib/client'
import { Button, cn } from 'cheval-ui'
import { AlertTriangle } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

type WatchdogConfirmBarShape = {
  pollKey?: number
  onKept?: () => void
  onRolledBack?: () => void
  onPendingChange?: (pending: boolean) => void
  className?: string
}

type Done = 'kept' | 'gone' | null

export function WatchdogConfirmBar({
  pollKey = 0,
  onKept,
  onRolledBack,
  onPendingChange,
  className,
}: WatchdogConfirmBarShape) {
  const [armed, setArmed] = useState(false)
  const [remaining, setRemaining] = useState(0)
  const [done, setDone] = useState<Done>(null)
  const [error, setError] = useState<string | null>(null)
  const [confirming, setConfirming] = useState(false)
  const wasPending = useRef(false)
  const confirmationVersion = useRef(0)
  const onClosed = useRef(onRolledBack)
  onClosed.current = onRolledBack
  const onPending = useRef(onPendingChange)
  onPending.current = onPendingChange

  useEffect(() => {
    let cancelled = false
    let fetching = false
    const check = async () => {
      if (fetching) return
      fetching = true
      const version = confirmationVersion.current
      try {
        const p = await api.apiApplyPendingGet()
        if (cancelled || version !== confirmationVersion.current) return
        setError(null)
        setArmed(p.pending)
        onPending.current?.(p.pending)
        if (p.pending) {
          wasPending.current = true
          setDone(null)
          setRemaining(p.remaining ?? 60)
        } else if (wasPending.current) {
          wasPending.current = false
          setDone('gone')
          onClosed.current?.()
        }
      } catch (err: unknown) {
        if (!cancelled && version === confirmationVersion.current) setError((err as Error).message || 'Could not check confirmation status; retrying…')
      } finally { fetching = false }
    }
    void check()
    const interval = setInterval(() => { void check() }, 3000)
    return () => { cancelled = true; clearInterval(interval) }
  }, [pollKey])

  useEffect(() => {
    if (!armed || done) return
    if (remaining <= 0) return
    const id = setTimeout(() => setRemaining((r) => r - 1), 1000)
    return () => clearTimeout(id)
  }, [armed, remaining, done, onRolledBack])

  if ((!armed || done) && !error) return null

  const keep = async () => {
    setConfirming(true)
    try {
      await api.apiApplyConfirmPost()
      confirmationVersion.current += 1
      wasPending.current = false
      setDone('kept')
      setArmed(false)
      onPending.current?.(false)
      setError(null)
      toast.success('Changes kept.')
      onKept?.()
    } catch (err: unknown) {
      setError((err as Error).message || 'Could not confirm changes. Please retry.')
    } finally { setConfirming(false) }
  }

  return (
    <div role="status" className={cn('flex items-center gap-3 rounded-md border border-warning/30 bg-warning/8 px-3 py-2 text-sm', className)}>
      <AlertTriangle className="h-4 w-4 shrink-0 text-warning" />
      <div className="min-w-0 flex-1">
        <p className="flex min-w-0 gap-1 font-medium text-warning">
          <span className="truncate">{armed ? 'Rolling back in' : 'Checking apply confirmation'}</span>
          {armed && <span className="shrink-0 font-mono tabular-nums">{remaining}s</span>}
        </p>
        {error && <p className="line-clamp-2 break-words text-xs text-muted-foreground">{error}</p>}
      </div>
      {armed && <Button size="sm" className="shrink-0" onClick={keep} disabled={confirming}>{confirming ? 'Confirming…' : 'Keep Changes'}</Button>}
    </div>
  )
}
