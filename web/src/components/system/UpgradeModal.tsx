import { apiRequest } from '@/lib/client'
import { getToken } from '@/lib/utils'
import { Button, Spinner, Dialog } from 'cheval-ui'
import { TerminalPane } from 'cheval-ui/terminal'
import { AlertTriangle, CheckCircle2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

interface UpgradeStatus {
  state: 'idle' | 'running' | 'done' | 'failed'
  reboot_required: boolean
  self_update: boolean
  error?: string
}

interface UpgradeModalProps {
  open: boolean
  selfUpdate: boolean
  onClose: () => void
}

export function UpgradeModal({ open, selfUpdate, onClose }: UpgradeModalProps) {
  const [status, setStatus] = useState<UpgradeStatus | null>(null)
  const [connKey, setConnKey] = useState(0)
  const [rebooting, setRebooting] = useState(false)
  const startedRef = useRef(false)
  const statusRef = useRef<UpgradeStatus | null>(null)
  statusRef.current = status

  useEffect(() => {
    if (!open) {
      startedRef.current = false
      setStatus(null)
      return
    }
    if (startedRef.current) return
    startedRef.current = true
    apiRequest('/api/system/upgrade', { method: 'POST' }).catch(() => {})
  }, [open])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    const poll = async () => {
      try {
        const s = await apiRequest<UpgradeStatus>('/api/system/upgrade')
        if (!cancelled) setStatus(s)
      } catch {}
    }
    poll()
    const id = setInterval(poll, 2000)
    return () => { cancelled = true; clearInterval(id) }
  }, [open])

  const done = status?.state === 'done'
  const failed = status?.state === 'failed'
  const running = !done && !failed

  const onStreamClose = () => {
    const state = statusRef.current?.state
    if (state === 'done' || state === 'failed') return
    setTimeout(() => setConnKey((k) => k + 1), 1500)
  }

  const reboot = async () => {
    setRebooting(true)
    try {
      await apiRequest('/api/system/reboot', { method: 'POST' })
      toast.message('Rebooting, the appliance will be unreachable for a moment')
    } catch (e) {
      toast.error((e as Error).message)
      setRebooting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Upgrading System"
      description={selfUpdate && running ? 'The web UI may disconnect briefly while it updates itself' : undefined}
      className="max-w-3xl"
      footer={(
        <div className="flex w-full items-center gap-2 text-sm">
          {running && (
            <>
              <Spinner size="sm" />
              <span className="text-muted-foreground">Upgrading…</span>
            </>
          )}
          {done && (
            <>
              <CheckCircle2 className="h-4 w-4 text-success" />
              <span className="text-success">Upgrade complete</span>
            </>
          )}
          {failed && (
            <>
              <AlertTriangle className="h-4 w-4 text-danger" />
              <span className="text-danger">Upgrade failed, see the log above</span>
            </>
          )}
          <div className="ml-auto flex items-center gap-2">
            {(done || failed) && <Button variant="outline" onClick={onClose}>Close</Button>}
            {done && status?.reboot_required && (
              <Button variant="destructive" onClick={reboot} disabled={rebooting}>
                {rebooting ? 'Rebooting…' : 'Reboot Now'}
              </Button>
            )}
          </div>
        </div>
      )}
    >
      <div className="-mx-5 -my-5 h-80">
        <TerminalPane
          key={connKey}
          wsPath="/api/ws/exec?name=upgrade"
          token={getToken() ?? undefined}
          className="h-full"
          onClose={onStreamClose}
          quiet
        />
      </div>
    </Dialog>
  )
}
