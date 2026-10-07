import type { IpcalcRangeCIDRs } from '@/api'
import { errorMessage, ErrorNote, ToolInput, useDebounced } from '@/components/ip-tools/shared'
import { api } from '@/lib/client'
import { Badge, Card, CardContent, CopyButton, Label } from 'cheval-ui'
import { useEffect, useState } from 'react'

export function RangeTool() {
  const [start, setStart] = useState('192.0.2.5')
  const [end, setEnd] = useState('192.0.2.20')
  const qStart = useDebounced(start.trim())
  const qEnd = useDebounced(end.trim())
  const [result, setResult] = useState<IpcalcRangeCIDRs | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!qStart || !qEnd) {
      setResult(null)
      setError(null)
      return
    }
    let live = true
    api.apiToolsRangeGet({ start: qStart, end: qEnd })
      .then((res) => { if (live) { setResult(res as IpcalcRangeCIDRs); setError(null) } })
      .catch((err) => { if (live) { setResult(null); setError(errorMessage(err)) } })
    return () => { live = false }
  }, [qStart, qEnd])

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <ToolInput label="Start" value={start} onChange={(e) => setStart(e.target.value)} placeholder="192.0.2.5" />
        <ToolInput label="End" value={end} onChange={(e) => setEnd(e.target.value)} placeholder="192.0.2.20" />
      </div>
      {error && <ErrorNote message={error} />}
      {result && (
        <Card>
          <CardContent className="space-y-2">
            <div className="flex items-center justify-between">
              <Label className="text-xs">{result.cidrs.length} block{result.cidrs.length !== 1 ? 's' : ''}</Label>
              <CopyButton text={result.cidrs.join('\n')} />
            </div>
            <div className="flex flex-wrap gap-1.5">
              {result.cidrs.map((c) => (
                <Badge key={c} variant="outline" className="font-mono">{c}</Badge>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
