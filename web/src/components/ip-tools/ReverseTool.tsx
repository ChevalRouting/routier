import type { IpcalcReverseDNS } from '@/api'
import { errorMessage, ErrorNote, ToolInput, useDebounced } from '@/components/ip-tools/shared'
import { api } from '@/lib/client'
import { Card, CardContent, CopyButton, Label } from 'cheval-ui'
import { useEffect, useState } from 'react'

export function ReverseTool() {
  const [input, setInput] = useState('100.64.12.192')
  const query = useDebounced(input.trim())
  const [result, setResult] = useState<IpcalcReverseDNS | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!query) {
      setResult(null)
      setError(null)
      return
    }
    let live = true
    api.apiToolsReverseGet({ ip: query })
      .then((res) => { if (live) { setResult(res as IpcalcReverseDNS); setError(null) } })
      .catch((err) => { if (live) { setResult(null); setError(errorMessage(err)) } })
    return () => { live = false }
  }, [query])

  return (
    <div className="space-y-4">
      <ToolInput
        label="Address or CIDR"
        hint="A CIDR aligned to an octet (IPv4) or nibble (IPv6) boundary also shows its delegation zone."
        value={input}
        onChange={(e) => setInput(e.target.value)}
        placeholder="100.64.12.192 or 2001:db8::/32"
      />
      {error && <ErrorNote message={error} />}
      {result && (
        <Card>
          <CardContent className="space-y-3">
            <div className="space-y-1.5">
              <Label className="text-xs">PTR name</Label>
              <div className="flex items-center gap-1.5 font-mono text-sm break-all">
                {result.name}
                <CopyButton text={result.name} />
              </div>
            </div>
            {result.zone && (
              <div className="space-y-1.5">
                <Label className="text-xs">Delegation zone</Label>
                <div className="flex items-center gap-1.5 font-mono text-sm break-all">
                  {result.zone}
                  <CopyButton text={result.zone} />
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
