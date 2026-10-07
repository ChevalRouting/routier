import type { TypesMacroInfo as MacroInfo } from '@/api'
import { ApplyDialog } from '@/components/macros/ApplyDialog'
import { api } from '@/lib/client'
import { useDataVersion } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { AlertDialog, Badge, Button, EmptyState, PageHeader, ReloadButton, Spinner } from 'cheval-ui'
import { Boxes, Play, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

export default function Macros() {
  const { data: macros, isLoading, error, reload } = useFetch(() => api.apiMacrosGet())
  const { bump } = useDataVersion()
  const [applyTarget, setApplyTarget] = useState<MacroInfo | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<MacroInfo | null>(null)
  const [deleting, setDeleting] = useState(false)

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await api.apiMacrosIdDelete({ id: deleteTarget.id })
      toast.success(`Macro "${deleteTarget.name}" deleted`)
      setDeleteTarget(null)
      reload()
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to delete macro')
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Macros"
        description="Named, reusable config changes you can apply repeatedly"
        action={<ReloadButton onClick={reload} />}
      />

      {isLoading && <Spinner />}
      {error && <p className="text-sm text-destructive">{error}</p>}

      {!isLoading && !error && macros !== null && (
        macros.length === 0
          ? (
            <EmptyState
              className="max-w-2xl mx-auto"
              icon={<Boxes />}
              title="No macros saved"
              message='Make changes, then click "Save as Macro" in the staging banner to record them.'
            />
          )
          : (
            <div className="space-y-2">
              {macros.map((m) => (
                <div
                  key={m.id}
                  className="flex items-center gap-3 px-4 py-3 rounded-lg bg-card shadow-[var(--card-shadow)]"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-medium text-sm">{m.name}</span>
                      {(m.sections ?? []).map((s) => (
                        <Badge key={s} variant="secondary" className="text-xs font-mono">{s}</Badge>
                      ))}
                    </div>
                    {m.description && (
                      <p className="text-xs text-muted-foreground mt-0.5">{m.description}</p>
                    )}
                    <p className="text-xs text-muted-foreground mt-0.5">
                      Created by {m.created_by} · {new Date(m.created_at).toLocaleDateString()}
                      {m.apply_count > 0 && ` · Applied ${m.apply_count}×`}
                    </p>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    <Button
                      size="sm"
                      variant="outline"
                      className="gap-1.5 h-7 text-xs"
                      onClick={() => setApplyTarget(m)}
                    >
                      <Play className="h-3 w-3" />
                      Apply
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive"
                      onClick={() => setDeleteTarget(m)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )
      )}

      {applyTarget && (
        <ApplyDialog
          macro={applyTarget}
          onClose={() => setApplyTarget(null)}
          onApplied={() => { reload(); bump() }}
        />
      )}

      {deleteTarget && (
        <AlertDialog
          open
          onCancel={() => setDeleteTarget(null)}
          onConfirm={handleDelete}
          title={`Delete "${deleteTarget.name}"?`}
          description="This macro will be permanently deleted. Applied configs are not affected."
          confirmLabel={deleting ? 'Deleting…' : 'Delete'}
          destructive
          busy={deleting}
        />
      )}
    </div>
  )
}
