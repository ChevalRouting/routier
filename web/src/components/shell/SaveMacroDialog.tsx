import { Button, Dialog, EntryRow, PreferencesGroup, Spinner } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { DiffView, withContext, type ViewLine } from '@/components/DiffView'
import { api } from '@/lib/client'

interface SaveMacroDialogProps {
  open: boolean
  onClose: () => void
}

export function SaveMacroDialog({ open, onClose }: SaveMacroDialogProps) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [diff, setDiff] = useState<ViewLine[] | null>(null)

  useEffect(() => {
    if (!open) return
    setName('')
    setDescription('')
    setError(null)
    setDiff(null)
    api.apiConfigDiffGet()
      .then((raw = []) => setDiff(raw.every((line) => line.type === 'same') ? [] : withContext(raw)))
      .catch(() => setDiff([]))
  }, [open])

  const save = async () => {
    const trimmed = name.trim()
    if (!trimmed) return
    setSaving(true)
    setError(null)
    try {
      await api.apiMacrosPost({ TypesCreateMacroRequest: { name: trimmed, description: description.trim() } })
      toast.success(`Macro ${trimmed} saved`)
      onClose()
    } catch (err: unknown) {
      setError((err as Error).message || 'Failed to save macro')
    } finally {
      setSaving(false)
    }
  }

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault()
    void save()
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Save as Macro"
      className="max-w-2xl"
      footer={(
        <>
          {error && <p className="mr-auto text-xs text-danger">{error}</p>}
          <Button variant="outline" onClick={onClose} disabled={saving}>Cancel</Button>
          <Button variant="suggested" onClick={save} disabled={saving || !name.trim()}>
            {saving ? 'Saving…' : 'Save'}
          </Button>
        </>
      )}
    >
      <form onSubmit={handleSubmit} className="space-y-5">
        <PreferencesGroup>
          <EntryRow title="Name" value={name} onChange={(event) => setName(event.target.value)} placeholder="enable-wireguard" autoFocus />
          <EntryRow title="Description" value={description} onChange={(event) => setDescription(event.target.value)} />
        </PreferencesGroup>
        <PreferencesGroup title="Changes">
          {diff === null && <Spinner />}
          {diff?.length === 0 && <p className="px-4 py-3 text-sm text-muted-foreground">No changes</p>}
          {diff && diff.length > 0 && (
            <div className="max-h-64 overflow-auto p-3">
              <DiffView lines={diff} />
            </div>
          )}
        </PreferencesGroup>
      </form>
    </Dialog>
  )
}
