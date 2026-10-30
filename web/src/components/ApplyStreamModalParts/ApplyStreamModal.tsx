import { ApplyStreamModalProps, armed, DEFAULT_WATCHDOG, lastApplyFailed, Phase, StatusBar } from '@/components/ApplyStreamModal'
import { api } from '@/lib/client'
import { getToken } from '@/lib/utils'
import { TerminalPane } from 'cheval-ui/terminal'
import { useEffect, useRef, useState } from 'react'
import { Dialog } from 'cheval-ui'

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

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Applying Configuration"
      className="max-w-3xl"
      footer={<StatusBar phase={phase} remaining={remaining} onKeep={keepChanges} onClose={onClose} />}
    >
      <div className="-mx-5 -my-5 h-80">
        <TerminalPane wsPath="/api/ws/exec?name=reapply" token={getToken() ?? undefined} className="h-full" onClose={handleClose} quiet />
      </div>
    </Dialog>
  )
}
