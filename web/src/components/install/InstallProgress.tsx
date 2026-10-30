import type { TypesInstallStatus as InstallStatus } from '@/api'
import { Button, NoticeBanner, Spinner } from 'cheval-ui'
import { CircleCheck } from 'lucide-react'
import { useEffect, useRef } from 'react'

type InstallProgressShape = {
  status: InstallStatus
  rebooting: boolean
  onReboot: () => void
  onRetry: () => void
}

export function InstallProgress({ status, rebooting, onReboot, onRetry }: InstallProgressShape) {
  const logRef = useRef<HTMLPreElement>(null)

  useEffect(() => {
    const el = logRef.current
    if (el) el.scrollTop = el.scrollHeight
  }, [status.log])

  return (
    <div className="space-y-4">
      {status.state === 'running' && (
        <div className="flex items-center justify-center gap-3 text-sm text-muted-foreground">
          <Spinner className="h-4 w-4" />
          <span>Installing to <span className="font-mono">{status.disk}</span>, this takes a few minutes</span>
        </div>
      )}

      {status.state === 'done' && (
        <div className="space-y-2 text-center">
          <CircleCheck className="mx-auto h-10 w-10 text-success" />
          <p className="text-base font-semibold">Routier Is Installed</p>
          <p className="text-sm text-muted-foreground">
            Remove the installation media, then restart. Every interface uses DHCP after the restart; sign in as routier with the password you chose.
          </p>
        </div>
      )}

      {status.state === 'failed' && (
        <NoticeBanner intent="danger">Installation failed: {status.error}</NoticeBanner>
      )}

      {status.log.length > 0 && (
        <pre
          ref={logRef}
          className="max-h-72 overflow-y-auto whitespace-pre-wrap break-words rounded-xl bg-muted/50 p-3 font-mono text-xs leading-relaxed text-muted-foreground"
        >
          {status.log.join('\n')}
        </pre>
      )}

      {status.state === 'done' && (
        <div className="flex justify-end">
          <Button onClick={onReboot} disabled={rebooting}>{rebooting ? 'Restarting…' : 'Restart'}</Button>
        </div>
      )}

      {status.state === 'failed' && (
        <div className="flex justify-end">
          <Button variant="outline" onClick={onRetry}>Back</Button>
        </div>
      )}
    </div>
  )
}
