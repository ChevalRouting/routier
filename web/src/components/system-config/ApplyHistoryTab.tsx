import { resultBadge } from '@/components/system-config/shared'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Badge, Button, Dialog, EmptyState, Pagination, Spinner, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, usePagination } from 'cheval-ui'
import { History, RefreshCw } from 'lucide-react'
import { useState } from 'react'

export function ApplyHistoryTab() {
  const { data, isLoading, reload } = useFetch(() => api.apiApplyLogsGet())
  const [openId, setOpenId] = useState<string | null>(null)
  const [logText, setLogText] = useState('')

  const view = (id: string) => {
    setOpenId(id); setLogText('Loading…')
    api.apiApplyLogsIdGet({ id }).then(setLogText).catch(() => setLogText('Failed to load log.'))
  }

  const records = data?.records ?? []
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(records, 15)
  if (isLoading) return <Spinner />

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">Recent configuration applies, restores and rollbacks.</p>
        <Button variant="outline" size="sm" onClick={() => reload()} className="gap-2"><RefreshCw className="h-3.5 w-3.5" />Refresh</Button>
      </div>
      {records.length === 0 ? (
        <EmptyState
          icon={<History />}
          title="No applies yet"
          message="Configuration applies, restores and rollbacks will show up here."
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
                <TableCell className="font-mono text-xs text-muted-foreground">{r.snap_id || '-'}</TableCell>
                <TableCell className="text-right">
                  <Button variant="ghost" size="sm" onClick={() => view(r.id)}>View log</Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="applies" />
        </>
      )}

      <Dialog open={!!openId} onClose={() => setOpenId(null)} title={`Apply log · ${openId ?? ''}`} className="max-w-3xl">
        <pre className="m-0 overflow-auto whitespace-pre-wrap rounded-md bg-[#09090b] p-4 font-mono text-xs text-zinc-200">{logText}</pre>
      </Dialog>
    </div>
  )
}
