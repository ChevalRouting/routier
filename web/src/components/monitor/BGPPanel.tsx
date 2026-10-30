import type { TypesBGPPeerSummary as BGPPeerSummary, TypesStatsResponse as StatsResponse } from '@/api'
import { BGPPeerGroup, groupBGPPeers, peerHealth } from '@/components/routing/overview/bgpPeers'
import { BGPNeighbor, RoutingConfig, WgIface } from '@/components/routing/overview/types'
import { api } from '@/lib/client'
import { EmptyState, Dialog, Spinner, ReloadButton } from 'cheval-ui'
import { useCallback, useEffect, useState } from 'react'
import { BGPSessionTable } from './BGPSessionTable'
import { OnActionChange } from './shared'

type BGPPanelShape = { onActionChange: OnActionChange }

function sessionNeighbors(peers: BGPPeerSummary[], configured: BGPNeighbor[]): BGPNeighbor[] {
  const byAddress = new Map(configured.map((n) => [n.address.split('%')[0], n]))
  return peers.map((p) => byAddress.get(p.address.split('%')[0]) ?? { address: p.address, remote_asn: p.asn, description: p.description })
}

function groupSummary(group: BGPPeerGroup): string {
  const asns = group.asns.map((a) => `AS${a}`).join(', ')
  const links = group.ifaces.length > 1 ? `${group.ifaces.length} tunnels` : group.ifaces[0]
  return [asns, links].filter(Boolean).join(' · ')
}

export function BGPPanel({ onActionChange }: BGPPanelShape) {
  const [bgp, setBgp] = useState<StatsResponse['bgp'] | null>(null)
  const [loading, setLoading] = useState(true)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const [selected, setSelected] = useState<string | null>(null)
  const [detail, setDetail] = useState('')
  const [detailLoading, setDetailLoading] = useState(false)
  const [neighbors, setNeighbors] = useState<BGPNeighbor[]>([])
  const [wireguard, setWireguard] = useState<Record<string, WgIface>>({})

  useEffect(() => {
    api.apiConfigSectionGet({ section: 'routing' })
      .then((r) => setNeighbors((r as RoutingConfig).bgp?.neighbors ?? []))
      .catch(() => {})
    api.apiConfigSectionGet({ section: 'vpn.wireguard' })
      .then((w) => setWireguard((w as Record<string, WgIface>) ?? {}))
      .catch(() => {})
  }, [])

  useEffect(() => {
    if (selected === null) return
    setDetailLoading(true)
    setDetail('')
    api.apiRoutingBgpNeighborGet({ address: selected })
      .then((out) => setDetail(typeof out === 'string' ? out : ''))
      .catch(() => setDetail('Failed to load neighbor detail.'))
      .finally(() => setDetailLoading(false))
  }, [selected])

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
        <ReloadButton onClick={refresh} loading={loading} />
      </div>
    )
  }, [loading, lastUpdated, refresh, onActionChange])

  if (loading) return <Spinner />

  const peers = bgp?.peers ?? []
  if (!bgp || peers.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center gap-2">
        <EmptyState className="py-10" title="No BGP Data" message="Enable BGP under Routing" />
      </div>
    )
  }

  const established = peers.filter((p) => p.state === 'Established').length
  const groups = groupBGPPeers(sessionNeighbors(peers, neighbors), wireguard, peers)
  const routers = groups.filter((g) => g.sessions.length > 1)
  const others = groups.filter((g) => g.sessions.length === 1).flatMap((g) => g.sessions)

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

      {routers.map((group) => {
        const health = peerHealth(group)
        return (
          <div key={group.id} className="space-y-1.5">
            <div className="flex items-baseline justify-between gap-3 px-1">
              <div className="min-w-0">
                <span className="font-semibold text-sm">{group.name}</span>
                <span className="ml-2 text-xs text-muted-foreground">{groupSummary(group)}</span>
              </div>
              <span className="shrink-0 font-mono text-xs" style={{ color: health.color }}>{health.up}/{health.total} up</span>
            </div>
            <BGPSessionTable sessions={group.sessions} showInterface={group.ifaces.length > 0} onSelect={setSelected} />
          </div>
        )
      })}

      {others.length > 0 && (
        <div className="space-y-1.5">
          {routers.length > 0 && <div className="px-1 font-semibold text-sm">Other Neighbors</div>}
          <BGPSessionTable sessions={others} showInterface={others.some((o) => o.iface)} onSelect={setSelected} />
        </div>
      )}

      <Dialog open={selected !== null} onClose={() => setSelected(null)} title={selected ? `BGP neighbor ${selected}` : 'BGP neighbor'} className="max-w-3xl">
        {detailLoading
          ? <Spinner />
          : <pre className="text-xs font-mono whitespace-pre-wrap break-words">{detail || 'No output.'}</pre>}
      </Dialog>
    </div>
  )
}
