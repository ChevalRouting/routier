import { resultBadge } from '@/components/system-config/shared'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Badge, Button, Dialog, EmptyState, Pagination, Spinner, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, usePagination } from 'cheval-ui'
import { History, RefreshCw } from 'lucide-react'
import { useEffect, useState } from 'react'
import { CodeBlock } from '@/components/ui/CodeBlock'
import type { TypesSnapshotInfo } from '@/api'
import { SnapshotBrowser } from './SnapshotBrowser'

export function ApplyHistoryTab() {
  const { data, isLoading, reload } = useFetch(() => api.apiApplyLogsGet())
  const { data: snapshotData, error: snapshotError } = useFetch(() => api.apiSnapshotsGet())
  const [openId, setOpenId] = useState<string | null>(null)
  const [logText, setLogText] = useState('')
  const [browsing, setBrowsing] = useState<TypesSnapshotInfo | null>(null)

  const snapshots = new Map((snapshotData?.snapshots ?? []).map((snapshot) => [snapshot.id, snapshot]))
  const openRecord = data?.records?.find((record) => record.id === openId)
  const linkedSnapshot = openRecord?.snap_id ? snapshots.get(openRecord.snap_id) : undefined

  const browseSnapshot = (snapshot: TypesSnapshotInfo) => {
    setOpenId(null)
    setBrowsing(snapshot)
  }

  const browseLinkedSnapshot = (id: string | undefined) => {
    const snapshot = id ? snapshots.get(id) : undefined
    if (snapshot) browseSnapshot(snapshot)
  }

  const view = (id: string) => setOpenId(id)

  useEffect(() => {
    if (!openId) return
    let active = true
    setLogText('Loading…')
    api.apiApplyLogsIdGet({ id: openId })
      .then((text) => { if (active) setLogText(text) })
      .catch(() => { if (active) setLogText('Failed to load log.') })
    return () => { active = false }
  }, [openId])

  const records = data?.records ?? []
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(records, 15)
  if (isLoading) return <Spinner />

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Button variant="outline" size="sm" onClick={() => reload()} className="gap-2"><RefreshCw className="h-3.5 w-3.5" />Refresh</Button>
      </div>
      {snapshotError && <p role="alert" className="text-sm text-danger">Could not check linked snapshots: {snapshotError}</p>}
      {records.length === 0 ? (
        <EmptyState
          icon={<History />}
          title="No Applies"
        />
      ) : (
        <>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Started</TableHead><TableHead>Source</TableHead>
              <TableHead>Status</TableHead><TableHead>Snapshot</TableHead><TableHead></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {pageItems.map((r) => (
              <TableRow key={r.id}>
                <TableCell className="font-mono text-xs">{new Date(r.started_at).toLocaleString()}</TableCell>
                <TableCell><Badge variant="outline">{r.source}</Badge></TableCell>
                <TableCell>{resultBadge(r)}</TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {r.snap_id && snapshots.has(r.snap_id)
                    ? <Button variant="ghost" size="sm" className="font-mono text-xs" onClick={() => browseLinkedSnapshot(r.snap_id)}>{r.snap_id}</Button>
                    : r.snap_id ? <span>{r.snap_id}{snapshotData ? ' (unavailable)' : ''}</span> : '-'}
                </TableCell>
                <TableCell className="text-right">
                  <Button variant="ghost" size="sm" onClick={() => view(r.id)}>View Log</Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="applies" />
        </>
      )}

      {browsing && <SnapshotBrowser key={browsing.id} snapshot={browsing} onClose={() => setBrowsing(null)} />}

      <Dialog open={!!openId} onClose={() => setOpenId(null)} title={`Apply Log · ${openId ?? ''}`} className="max-w-3xl" footer={linkedSnapshot && <Button variant="outline" onClick={() => browseSnapshot(linkedSnapshot)}>Browse Snapshot</Button>}>
        <CodeBlock className="max-h-[60vh]">{logText}</CodeBlock>
      </Dialog>
    </div>
  )
}
