import { DiffView, withContext, type ViewLine } from '@/components/DiffView'
import { ApplyDialogProps } from '@/components/macros/shared'
import { api } from '@/lib/client'
import { Button, Dialog } from 'cheval-ui'
import { RefreshCw } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

export function ApplyDialog({ macro, onClose, onApplied }: ApplyDialogProps) {
  const [diffLines, setDiffLines] = useState<ViewLine[] | null>(null)
  const [diffError, setDiffError] = useState<string | null>(null)
  const [applying, setApplying] = useState(false)
  const [loadingDiff, setLoadingDiff] = useState(true)

  useEffect(() => {
    api.apiMacrosIdDiffGet({ id: macro.id })
      .then((raw = []) => {
        setLoadingDiff(false)
        if (!raw.length || raw.every((l) => l.type === 'same')) {
          setDiffLines([])
        } else {
          setDiffLines(withContext(raw))
        }
      })
      .catch((err: unknown) => {
        setLoadingDiff(false)
        setDiffError((err as Error).message || 'Failed to load diff')
      })
  }, [macro.id])

  const doApply = async () => {
    setApplying(true)
    try {
      const result = await api.apiMacrosIdApplyPost({ id: macro.id, TypesMacroApplyRequest: { force: true } })
      if (result.warning) {
        toast.warning(`Applied with warning: ${result.warning}`)
      } else {
        toast.success(`Macro "${macro.name}" applied`)
      }
      onApplied()
      onClose()
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to apply macro')
      setApplying(false)
    }
  }

  return (
    <Dialog
      open
      onClose={onClose}
      title={`Apply "${macro.name}"`}
      description={macro.description || `Sections: ${(macro.sections ?? []).join(', ')}`}
      className="max-w-3xl"
      footer={
        <>
          <Button variant="outline" onClick={onClose} disabled={applying}>
            Cancel
          </Button>
          <Button
            onClick={doApply}
            disabled={applying || !!diffError}
            className="gap-2"
          >
            {applying && <RefreshCw className="h-3.5 w-3.5 animate-spin" />}
            {applying ? 'Applying...' : 'Apply'}
          </Button>
        </>
      }
    >
      {diffError && (
        <p className="text-sm text-destructive">{diffError}</p>
      )}

      {!diffError && loadingDiff && (
        <div className="flex items-center justify-center h-32">
          <RefreshCw className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      )}

      {!diffError && !loadingDiff && diffLines !== null && (
        diffLines.length === 0
          ? (
            <p className="text-sm text-muted-foreground text-center py-10">
              No changes, macro result is identical to current config.
            </p>
          )
          : (
            <div className="overflow-auto max-h-[50vh] rounded bg-muted/30 p-3">
              <DiffView lines={diffLines} />
            </div>
          )
      )}
    </Dialog>
  )
}
