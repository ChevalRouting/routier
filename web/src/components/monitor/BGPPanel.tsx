import type { TypesBGPPeerSummary as BGPPeerSummary, TypesStatsResponse as StatsResponse } from '@/api'
import { api } from '@/lib/client'
import { Button, EmptyState, Spinner, StateChip } from 'cheval-ui'
import { RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { OnActionChange } from './shared'

type BGPPanelShape = { onActionChange: OnActionChange }

export function BGPPanel({ onActionChange }: BGPPanelShape) {
  const [bgp, setBgp] = useState<StatsResponse['bgp'] | null>(null)
  const [loading, setLoading] = useState(true)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)

  const refresh = useCallback(async () => {
    try {
      const s = await api.apiStatsGet()
      setBgp(s.bgp ?? null)
      setLastUpdated(new Date())
    } catch {}
    finally { setLoading(false) }
  }, [])

  useEffect(() => { refresh(); const id = setInterval(refresh, 5000); return () => clearInterval(id) }, [refresh])

  useEffect(() => {
    onActionChange(
      <div className="flex items-center gap-2">
        {lastUpdated && <span className="text-xs text-muted-foreground hidden sm:inline">{lastUpdated.toLocaleTimeString()}</span>}
        <Button variant="outline" size="sm" onClick={refresh} disabled={loading} className="gap-1.5">
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />Refresh
        </Button>
      </div>
    )
  }, [loading, lastUpdated, refresh, onActionChange])

  if (loading) return <Spinner />

  const peers = bgp?.peers ?? []
  if (!bgp || peers.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center gap-2">
        <EmptyState className="py-10" title="No BGP data" message="Enable BGP under Network › Routing to see peer status here." />
        <p className="text-xs text-muted-foreground">BGP must be enabled and connected in Routing to see session data here.</p>
      </div>
    )
  }

  const established = peers.filter((p) => p.state === 'Established').length

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-6 flex-wrap">
        <div className="space-y-0.5">
          <p className="text-xs text-muted-foreground">Local AS</p>
          <p className="font-mono font-semibold text-sm">{bgp.local_asn}</p>
        </div>
        <div className="space-y-0.5">
          <p className="text-xs text-muted-foreground">Router ID</p>
          <p className="font-mono font-semibold text-sm">{bgp.router_id}</p>
        </div>
        <div className="space-y-0.5">
          <p className="text-xs text-muted-foreground">Sessions</p>
          <p className="font-mono font-semibold text-sm">
            <span className="text-success">{established}</span>
            <span className="text-muted-foreground">/{peers.length} up</span>
          </p>
        </div>
      </div>

      <div className="rounded-xl bg-card shadow-[var(--card-shadow)] overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-xs text-muted-foreground">
              <th className="text-left px-4 py-2.5">Neighbor</th>
              <th className="text-left px-4 py-2.5">Remote AS</th>
              <th className="text-left px-4 py-2.5">State</th>
              <th className="text-left px-4 py-2.5">Uptime</th>
              <th className="text-right px-4 py-2.5">Prefixes</th>
            </tr>
          </thead>
          <tbody>
            {[...peers]
              .sort((a, b) => {
                const ord = (s: string) => (s === 'Established' ? 0 : 1)
                const d = ord(a.state) - ord(b.state)
                return d !== 0 ? d : a.asn - b.asn
              })
              .map((p: BGPPeerSummary) => (
                <tr key={p.address} className="border-b border-border/50 last:border-0 hover:bg-muted/30">
                  <td className="px-4 py-2.5">
                    <div className="font-mono text-xs">{p.address}</div>
                    {p.description && <div className="text-xs text-muted-foreground mt-0.5">{p.description}</div>}
                  </td>
                  <td className="px-4 py-2.5 font-mono text-xs">AS{p.asn}</td>
                  <td className="px-4 py-2.5"><StateChip state={p.state} /></td>
                  <td className="px-4 py-2.5 font-mono text-xs text-muted-foreground">{p.uptime || '-'}</td>
                  <td className="px-4 py-2.5 text-right font-mono text-xs">{p.prefixes}</td>
                </tr>
              ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
