import { useEffect, useState } from 'react'
import { RefreshCw, RotateCw, Search, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from 'cheval-ui'
import { Input } from 'cheval-ui'
import { Card, CardHeader, CardTitle, CardContent } from 'cheval-ui'
import { Spinner } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import {
  dnsSummary, dnsStats, dnsReloadZone, dnsFlush, dnsRestart, dnsQuery,
  type DnsSummary, type DnsStats, type DnsQueryResult,
} from '@/lib/dnsApi'

function StatTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md bg-card px-3 py-2 shadow-[var(--card-shadow)]">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="font-mono text-sm">{value}</div>
    </div>
  )
}

function statOf(stats: DnsStats | null, name: string): number {
  return (stats?.stats ?? []).find((s) => s.name === name)?.value ?? 0
}

function hitRatio(stats: DnsStats | null): string {
  const hits = statOf(stats, 'cache hits')
  const misses = statOf(stats, 'cache misses')
  if (hits + misses === 0) return '—'

  return `${Math.round((hits / (hits + misses)) * 100)}%`
}

function QueryTool() {
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

export function DnsPanel() {
  const [summary, setSummary] = useState<DnsSummary | null>(null)
  const [stats, setStats] = useState<DnsStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)

  const load = async () => {
    try {
      const [s, st] = await Promise.all([dnsSummary().catch(() => null), dnsStats().catch(() => null)])
      setSummary(s)
      setStats(st)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    const id = setInterval(() => void load(), 10000)

    return () => clearInterval(id)
  }, [])

  const reload = async (zone: string) => {
    setBusy(true)
    try {
      await dnsReloadZone(zone)
      toast.success(`Reloaded ${zone}`)
      await load()
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const flush = async () => {
    setBusy(true)
    try {
      await dnsFlush()
      toast.success('Cache flushed')
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const restart = async () => {
    setBusy(true)
    try {
      await dnsRestart()
      toast.success('DNS server restarted')
      await load()
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <Spinner />

  if (!summary) {
    return <EmptyState title="No DNS server" message="Enable dns.server to run a local name server." />
  }

  const running = summary.overview?.running ?? false
  const zones = summary.overview?.zones ?? []

  return (
    <div className="space-y-5">
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
        <StatTile label="named" value={running ? 'running' : 'stopped'} />
        <StatTile label="Mode" value={summary.mode} />
        <StatTile label="Recursion" value={summary.recurses ? 'enabled' : 'refused'} />
        <StatTile label="Port" value={String(summary.port)} />
        <StatTile label="Queries" value={String(statOf(stats, 'QUERY'))} />
        <StatTile label="Cache hit ratio" value={hitRatio(stats)} />
        <StatTile label="NXDOMAIN" value={String(statOf(stats, 'queries resulted in NXDOMAIN'))} />
        <StatTile label="Recursion caused" value={String(statOf(stats, 'queries caused recursion'))} />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" onClick={() => void load()}>
          <RefreshCw className="mr-1 h-3 w-3" /> Refresh
        </Button>
        <Button variant="outline" size="sm" onClick={() => void flush()} disabled={busy || !running}>
          <Trash2 className="mr-1 h-3 w-3" /> Flush cache
        </Button>
        <Button variant="outline" size="sm" onClick={() => void restart()} disabled={busy || !running}>
          <RotateCw className="mr-1 h-3 w-3" /> Restart
        </Button>
      </div>

      <Card>
        <CardHeader><CardTitle className="text-sm">Listening on</CardTitle></CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            {(summary.overview?.listen ?? []).map((a) => (
              <span key={a} className="rounded-md bg-muted/40 px-2 py-1 font-mono text-xs">{a}</span>
            ))}
            {(summary.overview?.listen ?? []).length === 0 && (
              <span className="text-xs text-muted-foreground">No listen address resolved</span>
            )}
          </div>
          {(summary.upstreams ?? []).length > 0 && (
            <div className="mt-3">
              <div className="mb-1 text-xs text-muted-foreground">Upstreams</div>
              <div className="flex flex-wrap gap-2">
                {(summary.upstreams ?? []).map((u) => (
                  <span key={u} className="rounded-md bg-muted/40 px-2 py-1 font-mono text-xs">{u}</span>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle className="text-sm">Authoritative zones</CardTitle></CardHeader>
        <CardContent>
          {zones.length === 0 ? (
            <p className="text-xs text-muted-foreground">No authoritative zones configured.</p>
          ) : (
            <table className="w-full text-xs">
              <thead className="text-muted-foreground">
                <tr className="border-b border-border">
                  <th className="py-1 text-left font-normal">Zone</th>
                  <th className="py-1 text-left font-normal">Serial</th>
                  <th className="py-1 text-left font-normal">Records</th>
                  <th className="py-1 text-left font-normal">Answering</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {zones.map((z) => (
                  <tr key={z.name} className="border-b border-border last:border-0">
                    <td className="py-1 font-mono">{z.name}</td>
                    <td className="py-1 font-mono">{z.serial || '—'}</td>
                    <td className="py-1">{(z.records ?? []).length}</td>
                    <td className="py-1">
                      {z.answered
                        ? <span className="text-emerald-600 dark:text-emerald-500">yes</span>
                        : <span className="text-amber-600 dark:text-amber-500">no</span>}
                    </td>
                    <td className="py-1 text-right">
                      <Button variant="ghost" size="sm" disabled={busy || !running} onClick={() => void reload(z.name)}>
                        Reload
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          {zones.some((z) => !z.answered) && running && (
            <p className="mt-2 text-xs text-amber-600 dark:text-amber-500">
              A declared zone that is not answering usually failed to load. Check the named log.
            </p>
          )}
        </CardContent>
      </Card>

      <QueryTool />
    </div>
  )
}
