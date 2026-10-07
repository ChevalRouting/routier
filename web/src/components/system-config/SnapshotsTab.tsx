import type { TypesSnapshotInfo as SnapshotInfo } from '@/api'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { AlertDialog, Button, EmptyState, Pagination, Spinner, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, usePagination } from 'cheval-ui'
import { RotateCcw } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

type SnapshotsTabShape = { onChanged: () => void }

export function SnapshotsTab({ onChanged }: SnapshotsTabShape) {
  const { data, isLoading, reload } = useFetch(() => api.apiSnapshotsGet())
  const [busy, setBusy] = useState<string | null>(null)
  const [confirm, setConfirm] = useState<SnapshotInfo | null>(null)

  const restore = (s: SnapshotInfo) => setConfirm(s)

  const doRestore = () => {
    const s = confirm
    if (!s) return
    setConfirm(null)
    setBusy(s.id)
    api.apiSnapshotsIdRestorePost({ id: s.id })
      .then(() => { toast.success('Snapshot restored'); reload(); onChanged() })
      .catch((e) => toast.error(`Restore failed: ${e.message}`))
      .finally(() => setBusy(null))
  }

  const snapshots = data?.snapshots ?? []
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(snapshots, 15)
  if (isLoading) return <Spinner />

  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">Each apply records a snapshot of the files it changed. Restore reverts to a previous state.</p>
      {snapshots.length === 0 ? (
        <EmptyState
          icon={<RotateCcw />}
          title="No snapshots"
          message="Every apply snapshots the previous state so it can be restored."
        />
      ) : (
        <>
        <Table>
          <TableHeader>
            <TableRow><TableHead>Time</TableHead><TableHead>ID</TableHead><TableHead>Files</TableHead><TableHead></TableHead></TableRow>
          </TableHeader>
          <TableBody>
            {pageItems.map((s) => (
              <TableRow key={s.id}>
                <TableCell className="font-mono text-xs">{new Date(s.time).toLocaleString()}</TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">{s.id}</TableCell>
                <TableCell>{(s.files ?? []).length}</TableCell>
                <TableCell className="text-right">
                  <Button variant="outline" size="sm" className="gap-1.5" disabled={busy === s.id} onClick={() => restore(s)}>
                    <RotateCcw className="h-3.5 w-3.5" />Restore
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="snapshots" />
        </>
      )}

      <AlertDialog
        open={!!confirm}
        onCancel={() => setConfirm(null)}
        onConfirm={doRestore}
        title="Restore this snapshot?"
        description={confirm ? `Reverts the managed files to their state before apply ${confirm.id}. The current files will be overwritten.` : undefined}
        confirmLabel="Restore"
        cancelLabel="Cancel"
        destructive
      />
    </div>
  )
}
