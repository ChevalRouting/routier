import {
  dnsQuery,
  type DnsQueryResult
} from '@/lib/dnsApi'
import { Button, Card, CardContent, CardHeader, CardTitle, Input } from 'cheval-ui'
import { Search } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

export function QueryTool() {
  const [name, setName] = useState('')
  const [type, setType] = useState('A')
  const [result, setResult] = useState<DnsQueryResult | null>(null)
  const [busy, setBusy] = useState(false)

  const run = async () => {
    if (!name.trim()) return
    setBusy(true)
    try {
      setResult(await dnsQuery(name.trim(), type.trim()))
    } catch (err) {
      toast.error((err as Error).message)
      setResult(null)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Card>
      <CardHeader><CardTitle className="text-sm">Resolve a name</CardTitle></CardHeader>
      <CardContent className="space-y-3">
        <div className="flex gap-2">
          <Input
            className="h-8 flex-1 font-mono text-xs"
            placeholder="nas.home.arpa"
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') void run() }}
          />
          <Input
            className="h-8 w-20 font-mono text-xs"
            value={type}
            onChange={(e) => setType(e.target.value)}
          />
          <Button size="sm" onClick={() => void run()} disabled={busy}>
            <Search className="mr-1 h-3 w-3" /> Query
          </Button>
        </div>

        {result && (
          <div className="space-y-1.5">
            <div className="flex gap-4 text-xs text-muted-foreground">
              <span>rcode <span className="font-mono text-foreground">{result.rcode}</span></span>
              {result.query_ms > 0 && <span>{result.query_ms} ms</span>}
              {result.server && <span className="font-mono">{result.server}</span>}
            </div>
            {(result.answers ?? []).length === 0 ? (
              <p className="text-xs text-muted-foreground">No answer records.</p>
            ) : (
              <table className="w-full text-xs">
                <tbody>
                  {(result.answers ?? []).map((a, i) => (
                    <tr key={i} className="border-t border-border">
                      <td className="py-1 pr-3 font-mono">{a.name}</td>
                      <td className="py-1 pr-3 text-muted-foreground">{a.ttl}</td>
                      <td className="py-1 pr-3 font-mono">{a.type}</td>
                      <td className="py-1 font-mono">{a.data}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
