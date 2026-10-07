import { api } from '@/lib/client'
import { Button } from 'cheval-ui'
import { AlertTriangle } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

type WatchdogConfirmBarShape = {
  pollKey?: number
  onKept?: () => void
  onRolledBack?: () => void
}

type Done = 'kept' | 'gone' | null

export function WatchdogConfirmBar({
  pollKey = 0,
  onKept,
  onRolledBack,
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
      setError(null)
      toast.success('Changes kept.')
      onKept?.()
    } catch (err: unknown) {
      setError((err as Error).message || 'Could not confirm changes. Please retry.')
    } finally { setConfirming(false) }
  }

  return (
    <div className="flex items-center gap-3 rounded-md border border-warning/30 bg-warning/8 px-4 py-2.5 text-sm">
      <AlertTriangle className="h-4 w-4 text-warning shrink-0" />
      <span className="text-warning font-medium">{armed ? 'Apply pending confirmation' : 'Checking apply confirmation'}</span>
      <span className="text-muted-foreground">{error || 'Keep the changes to disarm the automatic rollback.'}</span>
      {armed && <div className="ml-auto flex items-center gap-3">
        <span className="font-mono tabular-nums font-semibold">{remaining}s</span>
        <Button size="sm" onClick={keep} disabled={confirming}>{confirming ? 'Confirming…' : 'Keep changes'}</Button>
      </div>}
    </div>
  )
}
