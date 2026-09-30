import { useEffect, useState } from 'react'
import { api } from '@/lib/client'
import type { TypesDiffLine as DiffLine, TypesFileDiff as FileDiff } from '@/api'
import { Button } from 'cheval-ui'
import { DiffCard } from 'cheval-ui'
import { Dialog } from 'cheval-ui'
import { RefreshCw } from 'lucide-react'

interface DiffModalProps {
  onApply: () => void
  onClose: () => void
  applying: boolean
}

export default function DiffModal({ onApply, onClose, applying }: DiffModalProps) {
  const [configLines, setConfigLines] = useState<DiffLine[] | null>(null)
  const [files, setFiles] = useState<FileDiff[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([api.apiConfigDiffGet(), api.apiConfigRenderDiffGet()])
      .then(([cfg, rendered]) => {
        setConfigLines(cfg ?? [])
        setFiles(rendered ?? [])
      })
      .catch((err: unknown) => setError((err as Error).message || 'Failed to load diff'))
  }, [])

  const configChanged = !!configLines && configLines.some((l) => l.type !== 'same')
  const loaded = configLines !== null && files !== null
  const noChanges = loaded && !configChanged && files!.length === 0

  return (
    <Dialog
      open
      onClose={onClose}
      title="Pending changes"
      description="Review the config and the files that will change on apply"
      className="max-w-3xl"
      footer={
        <>
          <Button variant="outline" onClick={onClose} disabled={applying}>
            Cancel
          </Button>
          <Button onClick={onApply} disabled={applying || !!error} className="gap-2">
            {applying && <RefreshCw className="h-3.5 w-3.5 animate-spin" />}
            {applying ? 'Applying…' : 'Apply'}
          </Button>
        </>
      }
    >
      {error && <p className="text-sm text-destructive">{error}</p>}

      {!error && !loaded && (
        <div className="flex h-32 items-center justify-center">
          <RefreshCw className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      )}

      {noChanges && (
        <p className="py-10 text-center text-sm text-muted-foreground">
          No differences detected between current and staged config.
        </p>
      )}

      {loaded && !noChanges && (
        <div className="space-y-2">
          {configChanged && configLines && (
            <DiffCard file="config.yml" status="modified" lines={configLines} defaultOpen />
          )}
          {files?.map((f) => (
            <DiffCard key={f.file} file={f.file} status={f.status} lines={f.lines ?? []} />
          ))}
        </div>
      )}
    </Dialog>
  )
}
