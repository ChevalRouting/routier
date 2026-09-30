import { useEffect, useState } from 'react'
import { AlertTriangle } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from 'cheval-ui'
import { api } from '@/lib/client'

type Done = 'kept' | 'gone' | null

export function WatchdogConfirmBar({
  pollKey = 0,
  onKept,
  onRolledBack,
}: {
  pollKey?: number
  onKept?: () => void
  onRolledBack?: () => void
}) {
  const [armed, setArmed] = useState(false)
  const [remaining, setRemaining] = useState(0)
  const [done, setDone] = useState<Done>(null)

  useEffect(() => {
    let cancelled = false
    api.apiApplyPendingGet()
      .then((p) => {
        if (cancelled || !p.pending) return
        setArmed(true)
        setDone(null)
        setRemaining(p.remaining ?? 60)
      })
      .catch(() => {})
    return () => { cancelled = true }
  }, [pollKey])

  useEffect(() => {
    if (!armed || done) return
    if (remaining <= 0) {
      setDone('gone')
      toast.error('Not confirmed in time, configuration was rolled back.')
      onRolledBack?.()
      return
    }
    const id = setTimeout(() => setRemaining((r) => r - 1), 1000)
    return () => clearTimeout(id)
  }, [armed, remaining, done, onRolledBack])

  if (!armed || done) return null

  const keep = () =>
    api.apiApplyConfirmPost()
      .then(() => {
        setDone('kept')
        toast.success('Changes kept.')
        onKept?.()
      })
      .catch(() => {})

  return (
    <div className="flex items-center gap-3 rounded-md border border-warning/30 bg-warning/8 px-4 py-2.5 text-sm">
      <AlertTriangle className="h-4 w-4 text-warning shrink-0" />
      <span className="text-warning font-medium">Apply pending confirmation</span>
      <span className="text-muted-foreground">Keep the changes to disarm the automatic rollback.</span>
      <div className="ml-auto flex items-center gap-3">
        <span className="font-mono tabular-nums font-semibold">{remaining}s</span>
        <Button size="sm" onClick={keep}>Keep changes</Button>
      </div>
    </div>
  )
}
