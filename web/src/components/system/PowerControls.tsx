import { useState } from 'react'
import { toast } from 'sonner'
import { Button, AlertDialog } from 'cheval-ui'
import { Power, RotateCcw } from 'lucide-react'
import { configLayerRequest } from '@/lib/client'

type PendingAction = 'reboot' | 'shutdown' | null

export function PowerControls() {
  const [pending, setPending] = useState<PendingAction>(null)
  const [busy, setBusy] = useState(false)

  const run = async () => {
    if (!pending) return
    setBusy(true)
    try {
      await configLayerRequest(`/api/system/${pending}`, { method: 'POST' })
      toast.message(pending === 'reboot'
        ? 'Rebooting, the appliance will be unreachable for a moment'
        : 'Shutting down, the appliance is powering off')
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setBusy(false)
      setPending(null)
    }
  }

  return (
    <>
      <Button variant="outline" size="sm" onClick={() => setPending('reboot')} className="gap-2">
        <RotateCcw className="h-4 w-4" />Reboot
      </Button>
      <Button variant="outline" size="sm" onClick={() => setPending('shutdown')} className="gap-2">
        <Power className="h-4 w-4" />Shutdown
      </Button>

      <AlertDialog
        open={pending !== null}
        onCancel={() => setPending(null)}
        onConfirm={run}
        title={pending === 'shutdown' ? 'Power off the appliance?' : 'Reboot the appliance?'}
        description={pending === 'shutdown'
          ? 'The router will power off and stop forwarding traffic until it is switched back on.'
          : 'The router will restart. Connectivity will drop for a moment while it comes back up.'}
        confirmLabel={pending === 'shutdown' ? 'Power off' : 'Reboot'}
        cancelLabel="Cancel"
        destructive
        busy={busy}
      />
    </>
  )
}
