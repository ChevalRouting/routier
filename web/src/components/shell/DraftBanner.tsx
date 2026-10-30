import { AlertDialog, Button, cn } from 'cheval-ui'
import { useState } from 'react'
import type { ApplyFailure } from '@/lib/applyFailure'

interface DraftBannerProps {
  editing: boolean
  savingDraft: boolean
  saveDisabled: boolean
  staging: boolean
  applying: boolean
  applyBlocked: boolean
  applyFailure: ApplyFailure | null
  canSaveMacro: boolean
  onSaveDraft: () => void
  onDiscardEdits: () => void
  onDiscardDraft: () => Promise<void>
  onApply: () => void
  onSaveMacro: () => void
  onShowLogs: () => void
}

function bannerTitle({ editing, staging, applyFailure }: DraftBannerProps): string {
  if (editing) return 'Unsaved edits'
  if (applyFailure) return 'Apply failed'
  if (staging) return 'Draft not applied'
  return ''
}

export function DraftBanner(props: DraftBannerProps) {
  const { editing, savingDraft, saveDisabled, staging, applying, applyFailure, canSaveMacro } = props
  const [confirmDiscard, setConfirmDiscard] = useState(false)
  const [discarding, setDiscarding] = useState(false)

  if (!editing && !staging && !applyFailure) return null

  const discardDraft = async () => {
    setDiscarding(true)
    try {
      await props.onDiscardDraft()
      setConfirmDiscard(false)
    } finally {
      setDiscarding(false)
    }
  }

  const failed = !editing && !!applyFailure

  return (
    <div
      role={failed ? 'alert' : 'status'}
      className={cn(
        'flex shrink-0 flex-wrap items-center gap-x-3 gap-y-2 border-b px-4 py-2 text-sm',
        failed ? 'border-danger/30 bg-danger/10 text-danger' : 'border-warning/20 bg-warning/10 text-warning',
      )}
    >
      <div className="min-w-0 flex-1">
        <p className="font-medium">{bannerTitle(props)}</p>
        {failed && <p className="line-clamp-2 whitespace-pre-wrap break-words text-xs">{applyFailure.message}</p>}
      </div>
      {editing && (
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={props.onDiscardEdits} disabled={savingDraft || applying}>Discard Edits</Button>
          <Button variant="suggested" onClick={props.onSaveDraft} disabled={savingDraft || applying || saveDisabled}>
            {savingDraft ? 'Saving…' : 'Save Draft'}
          </Button>
        </div>
      )}
      {!editing && (staging || applyFailure) && (
        <div className="flex flex-wrap items-center gap-2">
          {failed && <Button variant="outline" onClick={props.onShowLogs}>Check Logs</Button>}
          {staging && <Button variant="outline" onClick={() => setConfirmDiscard(true)} disabled={applying}>Discard Draft…</Button>}
          {staging && canSaveMacro && (
            <Button variant="outline" className="hidden sm:inline-flex" onClick={props.onSaveMacro} disabled={applying}>Save as Macro…</Button>
          )}
          <Button variant="suggested" onClick={props.onApply} disabled={applying || props.applyBlocked}>
            {applying ? 'Applying…' : failed ? 'Retry Apply…' : 'Apply…'}
          </Button>
        </div>
      )}
      <AlertDialog
        open={confirmDiscard}
        onCancel={() => setConfirmDiscard(false)}
        onConfirm={discardDraft}
        title="Discard Draft?"
        description="All saved changes that are not applied are lost"
        confirmLabel="Discard Draft"
        destructive
        busy={discarding}
      />
    </div>
  )
}
