import { RefreshCw, Check, X } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface SaveButtonProps {
  isDirty: boolean
  saving: boolean
  onClick: () => void
  onCancel?: () => void
}

export default function SaveButton({ isDirty, saving, onClick, onCancel }: SaveButtonProps) {
  if (saving) {
    return (
      <Button disabled size="sm" className="gap-2">
        <RefreshCw className="h-3.5 w-3.5 animate-spin" />
        Saving…
      </Button>
    )
  }

  if (!isDirty) {
    return (
      <Button variant="ghost" size="sm" disabled className="gap-2 text-muted-foreground">
        <Check className="h-3.5 w-3.5" />
        Saved
      </Button>
    )
  }

  return (
    <>
      {onCancel && (
        <Button variant="ghost" size="sm" onClick={onCancel} className="gap-1.5 text-muted-foreground">
          <X className="h-3.5 w-3.5" />
          Cancel
        </Button>
      )}
      <Button
        size="sm"
        onClick={onClick}
        className="gap-2 bg-warning hover:bg-warning/90 text-warning-foreground border-0"
      >
        <span className="inline-block h-2 w-2 rounded-full bg-warning-foreground animate-pulse" />
        Save
      </Button>
    </>
  )
}
