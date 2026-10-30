import { apiRequest } from '@/lib/client'
import { AlertDialog, MenuButton, type MenuItem } from 'cheval-ui'
import { EllipsisVertical, Power, RefreshCw, RotateCcw } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

type PendingAction = 'reboot' | 'shutdown' | null

interface PowerControlsProps {
  onReapply: () => void
}

export function PowerControls({ onReapply }: PowerControlsProps) {
  const [pending, setPending] = useState<PendingAction>(null)
  const [busy, setBusy] = useState(false)

  const run = async () => {
    if (!pending) return
    setBusy(true)
    try {
      await apiRequest(`/api/system/${pending}`, { method: 'POST' })
      toast.message(pending === 'reboot' ? 'Rebooting' : 'Shutting down')
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setBusy(false)
      setPending(null)
    }
  }

  const sections: MenuItem[][] = [
    [{ key: 'reapply', label: 'Force Re-Apply…', icon: RefreshCw, onSelect: onReapply }],
    [
      { key: 'reboot', label: 'Reboot…', icon: RotateCcw, onSelect: () => setPending('reboot') },
      { key: 'shutdown', label: 'Shut Down…', icon: Power, destructive: true, onSelect: () => setPending('shutdown') },
    ],
  ]

  return (
    <>
      <MenuButton label="System Actions" sections={sections} className="h-8 w-8">
        <EllipsisVertical className="h-4 w-4" />
      </MenuButton>

      <AlertDialog
        open={pending !== null}
        onCancel={() => setPending(null)}
        onConfirm={run}
        title={pending === 'shutdown' ? 'Shut Down Router?' : 'Reboot Router?'}
        description={pending === 'shutdown'
          ? 'Forwarding stops until the router is powered on again'
          : 'Connectivity drops until the router is back up'}
        confirmLabel={pending === 'shutdown' ? 'Shut Down' : 'Reboot'}
        destructive
        busy={busy}
      />
    </>
  )
}
