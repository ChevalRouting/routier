import type { TypesArtifactContent, TypesFailureRecord } from '@/api'
import type { ApplyFailure } from '@/lib/applyFailure'
import { api, configLayerRequest } from '@/lib/client'
import { Button, ComboRow, Dialog, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { useEffect, useState } from 'react'

type ApplyFailureModalShape = { failure: ApplyFailure; onClose: () => void }

export default function ApplyFailureModal({ failure, onClose }: ApplyFailureModalShape) {
  const [bundle, setBundle] = useState<TypesFailureRecord | null>(null)
  const [logs, setLogs] = useState<string | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [artifact, setArtifact] = useState<string>('')
  const [content, setContent] = useState<string | null>(null)
  const [artifactError, setArtifactError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let active = true
    setBundle(null)
    setLogs(null)
    setArtifact('')
    setContent(null)
    setLoadError(null)
    setLoading(true)
    const load = async () => {
      const [metadata, log] = await Promise.allSettled([
        failure.bundleID
          ? configLayerRequest<TypesFailureRecord>(`/api/failures/${encodeURIComponent(failure.bundleID)}`)
          : Promise.resolve(null),
        failure.logID
          ? api.apiApplyLogsIdGet({ id: failure.logID })
          : Promise.resolve(null),
      ])
      if (!active) return
      const errors: string[] = []
      if (metadata.status === 'fulfilled') {
        setBundle(metadata.value)
        const dest = failure.errors?.find((e) => metadata.value?.artifacts?.includes(e.dest ?? ''))?.dest
        setArtifact(dest ?? metadata.value?.artifacts?.[0] ?? '')
      } else errors.push(`Could not load artifacts: ${String(metadata.reason)}`)
      if (log.status === 'fulfilled') setLogs(log.value)
      else errors.push(`Could not load apply log: ${String(log.reason)}`)
      setLoadError(errors.length ? errors.join('\n') : null)
      setLoading(false)
    }
    void load()
    return () => { active = false }
  }, [failure])

  useEffect(() => {
    if (!artifact || !failure.bundleID) return
    let active = true
    setContent(null)
    setArtifactError(null)
    configLayerRequest<TypesArtifactContent>(`/api/failures/${encodeURIComponent(failure.bundleID)}/artifact?dest=${encodeURIComponent(artifact)}`)
      .then((result) => { if (active) setContent(result.content) })
      .catch((err: unknown) => { if (active) setArtifactError((err as Error).message) })
    return () => { active = false }
  }, [artifact, failure.bundleID])

  const errors = failure.errors?.length ? failure.errors : bundle?.errors ?? []
  const errorLines = new Set(errors.filter((e) => e.dest === artifact && e.line > 0).map((e) => e.line))

  return (
    <Dialog
      open
      onClose={onClose}
      title="Apply failure"
      description="Review the failure, apply log, and rendered artifacts from this attempt"
      className="max-w-4xl"
      footer={<Button variant="outline" onClick={onClose}>Close</Button>}
    >
      <div className="space-y-4">
        <div role="alert" className="rounded border border-danger/30 bg-danger/10 p-3 text-sm text-danger">
          <p className="break-words font-medium">{failure.message}</p>
          {errors.length > 0 && <ul className="mt-2 space-y-1 text-xs">
            {errors.map((error, index) => <li key={index} className="break-words">
              {error.tool && <span className="font-medium">{error.tool}: </span>}
              {error.dest && <span className="font-mono">{error.dest}{error.line > 0 ? `:${error.line}` : ''}: </span>}
              {error.message}
            </li>)}
          </ul>}
        </div>
        {loading && <p role="status" className="text-sm text-muted-foreground">Loading diagnostics…</p>}
        {loadError && <p role="alert" className="whitespace-pre-wrap text-sm text-danger">{loadError}</p>}
        <div className="space-y-2">
          <h3 className="text-sm font-semibold">Apply log</h3>
          {failure.logID && <p className="font-mono text-xs text-muted-foreground">{failure.logID}</p>}
          {logs !== null ? <pre className="max-h-64 overflow-auto rounded bg-muted/40 p-3 text-xs">{logs || 'No log output was recorded.'}</pre>
            : !loading && !failure.logID && <p className="text-xs text-muted-foreground">No apply log was created for this failure.</p>}
        </div>
        <div className="space-y-2">
          <h3 className="text-sm font-semibold">Rendered artifacts</h3>
          {bundle?.artifacts?.length ? <>
            <ComboRow title="Artifact">
              <Select value={artifact} onValueChange={setArtifact}>
                <SelectTrigger className="font-mono text-xs">
                  <SelectValue placeholder="Select an artifact" />
                </SelectTrigger>
                <SelectContent>
                  {bundle.artifacts.map((dest) => <SelectItem key={dest} value={dest}>{dest}</SelectItem>)}
                </SelectContent>
              </Select>
            </ComboRow>
            {artifactError && <p role="alert" className="text-xs text-danger">{artifactError}</p>}
            {content !== null ? <pre className="max-h-80 overflow-auto rounded bg-muted/40 p-3 text-xs">{content.split('\n').map((line, index) =>
              <span key={index} className={`block ${errorLines.has(index + 1) ? 'bg-danger/15 text-danger' : ''}`}><span className="mr-4 inline-block w-8 select-none text-right text-muted-foreground">{index + 1}</span>{line || ' '}</span>)}</pre>
              : !artifactError && <p role="status" className="text-xs text-muted-foreground">Loading artifact…</p>}
          </> : !loading && <p className="text-xs text-muted-foreground">No rendered artifacts are available for this attempt.</p>}
        </div>
      </div>
    </Dialog>
  )
}
