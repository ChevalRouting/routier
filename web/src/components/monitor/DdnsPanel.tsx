import { dnsDdns, dnsDdnsDelete, type DnsDdnsZoneView } from '@/lib/dnsApi'
import { AlertDialog, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, ReloadButton, Spinner, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'
import { Trash2 } from 'lucide-react'
import { plural } from '@/lib/utils'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

type RecShape = { name: string; type: string; value: string }

type PendingDelete = { zone: string; rec: RecShape }

export function DdnsPanel() {
  const [zones, setZones] = useState<DnsDdnsZoneView[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState('')
  const [pending, setPending] = useState<PendingDelete | null>(null)
  const [loadError, setLoadError] = useState('')

  const load = async () => {
    try {
      setZones(await dnsDdns())
      setLoadError('')
    } catch (err) {
      setLoadError((err as Error).message)
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

  const remove = async () => {
    if (!pending) return
    const { zone, rec } = pending
    const key = `${zone}|${rec.name}|${rec.type}|${rec.value}`
    setBusy(key)
    try {
      await dnsDdnsDelete(zone, rec)
      toast.success(`Deleted ${rec.name}`)
      setPending(null)
      await load()
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setBusy('')
    }
  }

  if (loading) return <Spinner />

  if (!zones || zones.length === 0) {
    return <EmptyState title="No Dynamic DNS" message={loadError ? `${loadError.charAt(0).toUpperCase()}${loadError.slice(1)}` : 'Enable dhcp.ddns alongside a local dns.server'} />
  }

  const total = zones.reduce((n, z) => n + (z.records ?? []).length, 0)

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm text-muted-foreground">{plural(total, 'dynamic record')}</span>
        <div className="ml-auto"><ReloadButton onClick={() => void load()} /></div>
      </div>

      {zones.map((z) => (
        <Card key={z.name}>
          <CardHeader><CardTitle className="text-sm font-mono">{z.name}</CardTitle></CardHeader>
          <CardContent>
            {z.error ? (
              <p className="text-sm text-warning">{z.error}</p>
            ) : (z.records ?? []).length === 0 ? (
              <p className="text-sm text-muted-foreground">No dynamic records</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Value</TableHead>
                    <TableHead>TTL</TableHead>
                    <TableHead className="w-10" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(z.records ?? []).map((r, i) => (
                    <TableRow key={i}>
                      <TableCell className="font-mono">{r.name}</TableCell>
                      <TableCell className="font-mono">{r.type}</TableCell>
                      <TableCell className="font-mono">{r.value}</TableCell>
                      <TableCell className="text-muted-foreground">{r.ttl ?? '-'}</TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label="Delete Record"
                          title="Delete Record"
                          disabled={busy === `${z.name}|${r.name}|${r.type}|${r.value}`}
                          onClick={() => setPending({ zone: z.name, rec: { name: r.name, type: r.type, value: r.value } })}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      ))}

      <AlertDialog
        open={pending !== null}
        onCancel={() => setPending(null)}
        onConfirm={remove}
        title="Delete Record?"
        description={pending ? `${pending.rec.type} ${pending.rec.name} → ${pending.rec.value}` : undefined}
        confirmLabel="Delete"
        destructive
        busy={busy !== ''}
      />
    </div>
  )
}
