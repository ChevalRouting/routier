import { dnsDdns, dnsDdnsDelete, type DnsDdnsZoneView } from '@/lib/dnsApi'
import { Button, Card, CardContent, CardHeader, CardTitle, EmptyState, Spinner } from 'cheval-ui'
import { RefreshCw, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

type RecShape = { name: string; type: string; value: string }

export function DdnsPanel() {
  const [zones, setZones] = useState<DnsDdnsZoneView[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState('')

  const load = async () => {
    try {
      setZones(await dnsDdns())
    } catch (err) {
      toast.error((err as Error).message)
      setZones([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    const id = setInterval(() => void load(), 15000)

    return () => clearInterval(id)
  }, [])

  const remove = async (zone: string, rec: RecShape) => {
    if (!window.confirm(`Delete ${rec.type} record ${rec.name} → ${rec.value}?`)) return

    const key = `${zone}|${rec.name}|${rec.type}|${rec.value}`
    setBusy(key)
    try {
      await dnsDdnsDelete(zone, rec)
      toast.success(`Deleted ${rec.name}`)
      await load()
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setBusy('')
    }
  }

  if (loading) return <Spinner />

  if (!zones || zones.length === 0) {
    return <EmptyState title="No dynamic DNS" message="Enable dhcp.ddns alongside a local dns.server to see leased records here." />
  }

  const total = zones.reduce((n, z) => n + (z.records ?? []).length, 0)

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" onClick={() => void load()}>
          <RefreshCw className="mr-1 h-3 w-3" /> Refresh
        </Button>
        <span className="text-xs text-muted-foreground">{total} dynamic record{total === 1 ? '' : 's'}</span>
      </div>

      {zones.map((z) => (
        <Card key={z.name}>
          <CardHeader><CardTitle className="text-sm font-mono">{z.name}</CardTitle></CardHeader>
          <CardContent>
            {z.error ? (
              <p className="text-xs text-amber-600 dark:text-amber-500">{z.error}</p>
            ) : (z.records ?? []).length === 0 ? (
              <p className="text-xs text-muted-foreground">No dynamic records.</p>
            ) : (
              <table className="w-full text-xs">
                <thead className="text-muted-foreground">
                  <tr className="border-b border-border">
                    <th className="py-1 text-left font-normal">Name</th>
                    <th className="py-1 text-left font-normal">Type</th>
                    <th className="py-1 text-left font-normal">Value</th>
                    <th className="py-1 text-left font-normal">TTL</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {(z.records ?? []).map((r, i) => {
                    const key = `${z.name}|${r.name}|${r.type}|${r.value}`

                    return (
                      <tr key={i} className="border-b border-border last:border-0">
                        <td className="py-1 pr-3 font-mono">{r.name}</td>
                        <td className="py-1 pr-3 font-mono">{r.type}</td>
                        <td className="py-1 pr-3 font-mono">{r.value}</td>
                        <td className="py-1 pr-3 text-muted-foreground">{r.ttl ?? '-'}</td>
                        <td className="py-1 text-right">
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={busy === key}
                            onClick={() => void remove(z.name, { name: r.name, type: r.type, value: r.value })}
                          >
                            <Trash2 className="h-3 w-3" />
                          </Button>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
