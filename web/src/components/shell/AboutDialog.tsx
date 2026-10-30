import { Dialog, PreferencesGroup, Row, Spinner } from 'cheval-ui'
import { useEffect, useState } from 'react'
import Logo from '@/components/Logo'
import { api } from '@/lib/client'

interface VersionInfo {
  version?: string
  kernel?: string
  flavor?: string
  live?: boolean
}

interface AboutDialogProps {
  open: boolean
  onClose: () => void
}

export function AboutDialog({ open, onClose }: AboutDialogProps) {
  const [info, setInfo] = useState<VersionInfo | null>(null)

  useEffect(() => {
    if (!open) return
    api.apiSystemVersionGet().then((value) => setInfo(value as VersionInfo)).catch(() => setInfo({}))
  }, [open])

  return (
    <Dialog open={open} onClose={onClose} title="About Routier" className="max-w-sm">
      <div className="space-y-5">
        <div className="flex flex-col items-center gap-2 text-center">
          <Logo className="h-16 w-16" />
          <p className="text-base font-semibold">Routier</p>
          {info?.version && <p className="text-sm text-muted-foreground">{info.version}</p>}
        </div>
        {info === null ? <Spinner /> : (
          <PreferencesGroup>
            <Row title="Kernel">
              <span className="font-mono text-sm">{info.kernel || 'Unknown'}</span>
            </Row>
            <Row title="Flavor">
              <span className="text-sm">{info.live ? `${info.flavor || 'Unknown'}, live` : info.flavor || 'Unknown'}</span>
            </Row>
          </PreferencesGroup>
        )}
      </div>
    </Dialog>
  )
}
