import type { RenderImportedPrefixList } from '@/api'
import { Button, Dialog, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'

type ImportedPrefixListDialogShape = {
  list: RenderImportedPrefixList
  onClose: () => void
}

export function ImportedPrefixListDialog({ list, onClose }: ImportedPrefixListDialogShape) {
  const entries = [...list.entries].sort((a, b) => a.seq - b.seq)

  return (
    <Dialog
      open
      onClose={onClose}
      title={`Prefix List ${list.name}`}
      description={`Imported from ${list.sources.join(', ')}, edit it on the friend that shares it`}
      className="max-w-2xl"
      footer={<Button variant="outline" onClick={onClose}>Close</Button>}
    >
      {entries.length === 0 ? (
        <p className="text-sm text-muted-foreground">No entries</p>
      ) : (
        <div className="overflow-x-auto rounded bg-card shadow-[var(--card-shadow)]">
          <Table className="w-full text-xs">
            <TableHeader className="bg-muted/50">
              <TableRow>
                <TableHead className="w-16 px-3 py-1.5 text-left font-medium text-muted-foreground">Seq</TableHead>
                <TableHead className="w-24 px-3 py-1.5 text-left font-medium text-muted-foreground">Action</TableHead>
                <TableHead className="px-3 py-1.5 text-left font-medium text-muted-foreground">Prefix</TableHead>
                <TableHead className="w-16 px-3 py-1.5 text-left font-medium text-muted-foreground">ge</TableHead>
                <TableHead className="w-16 px-3 py-1.5 text-left font-medium text-muted-foreground">le</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody className="divide-y font-mono">
              {entries.map((entry) => (
                <TableRow key={entry.seq}>
                  <TableCell className="px-3 py-1.5">{entry.seq}</TableCell>
                  <TableCell className={entry.action === 'deny' ? 'px-3 py-1.5 text-danger' : 'px-3 py-1.5 text-success'}>{entry.action}</TableCell>
                  <TableCell className="px-3 py-1.5">{entry.prefix}</TableCell>
                  <TableCell className="px-3 py-1.5 text-muted-foreground">{entry.ge || '-'}</TableCell>
                  <TableCell className="px-3 py-1.5 text-muted-foreground">{entry.le || '-'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </Dialog>
  )
}
