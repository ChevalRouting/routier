import { configLayerRequest } from '@/lib/client'
import { getToken } from '@/lib/utils'
import { Button } from 'cheval-ui'
import { TerminalPane } from 'cheval-ui/terminal'
import { AlertTriangle, CheckCircle2, RefreshCw, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
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
    configLayerRequest('/api/system/upgrade', { method: 'POST' }).catch(() => {})
  }, [open])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    const poll = async () => {
      try {
        const s = await configLayerRequest<UpgradeStatus>('/api/system/upgrade')
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
      await configLayerRequest('/api/system/reboot', { method: 'POST' })
      toast.message('Rebooting, the appliance will be unreachable for a moment')
    } catch (e) {
      toast.error((e as Error).message)
      setRebooting(false)
    }
  }

  if (!open) return null

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative z-10 w-full max-w-3xl mx-4 bg-background rounded-xl shadow-2xl flex flex-col max-h-[80vh]">
        <div className="flex items-center justify-between px-5 py-3 shrink-0">
          <span className="font-semibold text-sm">Upgrading system…</span>
          <button type="button" onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>

        {selfUpdate && running && (
          <div className="px-5 pb-2 text-xs text-muted-foreground">
            The web UI may briefly disconnect while it updates itself. This is expected, the upgrade keeps running in the background.
          </div>
        )}

        <div className="flex-1 min-h-0 overflow-hidden" style={{ minHeight: 320 }}>
          <TerminalPane
            key={connKey}
            wsPath="/api/ws/exec?name=upgrade"
            token={getToken() ?? undefined}
            className="h-full"
            onClose={onStreamClose}
            quiet
          />
        </div>

        <div className="flex items-center gap-2 px-5 pb-4 pt-2 shrink-0 text-sm">
          {running && (
            <>
              <RefreshCw className="h-4 w-4 animate-spin text-muted-foreground" />
              <span className="text-muted-foreground">Upgrade in progress…</span>
            </>
          )}
          {done && (
            <>
              <CheckCircle2 className="h-4 w-4 text-success" />
              <span className="text-success">Upgrade complete.</span>
            </>
          )}
          {failed && (
            <>
              <AlertTriangle className="h-4 w-4 text-danger" />
              <span className="text-danger">Upgrade failed. Check the log above.</span>
            </>
          )}
          <div className="ml-auto flex items-center gap-2">
            {done && status?.reboot_required && (
              <Button size="sm" variant="destructive" onClick={reboot} disabled={rebooting}>
                {rebooting ? 'Rebooting…' : 'Reboot now'}
              </Button>
            )}
            {(done || failed) && <Button size="sm" variant="outline" onClick={onClose}>Close</Button>}
          </div>
        </div>
      </div>
    </div>,
    document.body,
  )
}
