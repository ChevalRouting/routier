import { ApplyStreamModalProps, armed, DEFAULT_WATCHDOG, lastApplyFailed, Phase, StatusBar } from '@/components/ApplyStreamModal'
import { api } from '@/lib/client'
import { getToken } from '@/lib/utils'
import { TerminalPane } from 'cheval-ui/terminal'
import { X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'

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
