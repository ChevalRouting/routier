import type { TypesSnapshotInfo } from '@/api'
import { api } from '@/lib/client'
import { Button, ComboRow, Dialog, EmptyState, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Spinner } from 'cheval-ui'
import { FileText } from 'lucide-react'
import { useEffect, useState } from 'react'
import { CodeBlock } from '@/components/ui/CodeBlock'

interface SnapshotBrowserProps {
  snapshot: TypesSnapshotInfo
  onClose: () => void
}

interface ArtifactState {
  content: string | null
  error: string | null
}

export function SnapshotBrowser({ snapshot, onClose }: SnapshotBrowserProps) {
  const files = snapshot.files ?? []
  const [dest, setDest] = useState(files[0] ?? '')
  const [state, setState] = useState<ArtifactState>({ content: null, error: null })

  useEffect(() => {
    if (!dest) return
    let active = true
    setState({ content: null, error: null })
    api.apiSnapshotsIdArtifactGet({ id: snapshot.id, dest })
      .then((artifact) => { if (active) setState({ content: artifact.content, error: null }) })
      .catch((error: unknown) => { if (active) setState({ content: null, error: (error as Error).message }) })
    return () => { active = false }
  }, [snapshot.id, dest])

  return (
    <Dialog
      open
      onClose={onClose}
      title="Snapshot Artifacts"
      description={`Files saved before apply ${snapshot.id}, ${new Date(snapshot.time).toLocaleString()}`}
      className="max-w-4xl"
      footer={<Button variant="outline" onClick={onClose}>Close</Button>}
    >
      <div className="space-y-4">
        {files.length === 0 ? <EmptyState icon={<FileText />} title="No Saved Artifacts" /> : (
          <>
            <ComboRow title="Artifact">
              <Select value={dest} onValueChange={setDest}>
                <SelectTrigger className="font-mono text-xs"><SelectValue /></SelectTrigger>
                <SelectContent>{files.map((file) => <SelectItem key={file} value={file}>{file}</SelectItem>)}</SelectContent>
              </Select>
            </ComboRow>
            {state.error ? <p role="alert" className="text-sm text-danger">{state.error}</p>
              : state.content === null ? <Spinner />
                : state.content === '' ? <p className="text-sm text-muted-foreground">No saved content. This file may not have existed before the apply.</p>
                  : <CodeBlock className="max-h-[60vh]">{state.content}</CodeBlock>}
          </>
        )}
      </div>
    </Dialog>
  )
}
