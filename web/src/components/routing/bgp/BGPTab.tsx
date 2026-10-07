import { migrateBGPPolicies } from '@/lib/bgpPolicies'
import { Label, SectionNav, Switch, useTabState } from 'cheval-ui'
import { useEffect } from 'react'
import { BGPConfig, BGPSubTab, PrefixEntry, RouteMapEntry } from '../types'
import { BGPAddressFamiliesPanel } from './AddressFamilies'
import { BGPGeneralPanel } from './GeneralPanel'
import { BGPNeighborsPanel } from './Neighbors'
import { BGPPrefixListsPanel } from './PrefixLists'
import { BGPRouteMapsPanel } from './RouteMaps'

type BGPTabShape = {
  bgp: BGPConfig
  setBGP: (v: BGPConfig) => void
  enabled: boolean
  setEnabled: (v: boolean) => void
  vrfNames: string[]
  bfdProfileNames: string[]
  onDirty: () => void
  prefixLists: Record<string, PrefixEntry[]>
  setPrefixLists: (v: Record<string, PrefixEntry[]>) => void
  routeMaps: Record<string, RouteMapEntry[]>
  setRouteMaps: (v: Record<string, RouteMapEntry[]>) => void
}

type SubTabsShape = { key: BGPSubTab; label: string }

export function BGPTab({
  bgp, setBGP, enabled, setEnabled, vrfNames, bfdProfileNames, onDirty,
  prefixLists, setPrefixLists, routeMaps, setRouteMaps,
}: BGPTabShape) {
  useEffect(() => {
    const migrated = migrateBGPPolicies(bgp)
    if (migrated !== bgp) { setBGP(migrated); onDirty() }
  }, [bgp, setBGP, onDirty])

  const [subTab, setSubTab] = useTabState<BGPSubTab>('routing.bgp', 'general')

  const prefixListNames = Object.keys(prefixLists)
  const routeMapNames = Object.keys(routeMaps)

  const subTabs: SubTabsShape[] = [
    { key: 'general', label: 'General' },
    { key: 'neighbors', label: bgp.neighbors.length ? `Neighbors (${bgp.neighbors.length})` : 'Neighbors' },
    { key: 'afs', label: 'Address Families' },
    { key: 'prefix-lists', label: prefixListNames.length ? `Prefix Lists (${prefixListNames.length})` : 'Prefix Lists' },
    { key: 'route-maps', label: routeMapNames.length ? `Route Maps (${routeMapNames.length})` : 'Route Maps' },
  ]

  return (
    <div className="space-y-5">
      <p className="text-sm text-muted-foreground">
        BGP (Border Gateway Protocol) exchanges routes with external peers. Configure neighbors, address
        families, prefix lists and route maps; enable BFD on a neighbor for sub-second failure detection.
      </p>
      <div className="flex items-center justify-between gap-2">
        <Label htmlFor="bgp-enable" className="cursor-pointer">Enable BGP</Label>
        <Switch id="bgp-enable" checked={enabled} onCheckedChange={(v) => { setEnabled(v); onDirty() }} />
      </div>

      {!enabled ? (
        <p className="text-sm text-muted-foreground italic">BGP is disabled. Enable it above to configure.</p>
      ) : (
        <SectionNav items={subTabs} active={subTab} onChange={setSubTab}>
          {subTab === 'general' && (
            <BGPGeneralPanel bgp={bgp} setBGP={setBGP} onDirty={onDirty} />
          )}
          {subTab === 'neighbors' && (
            <BGPNeighborsPanel
              neighbors={bgp.neighbors}
              onChange={(neighbors) => setBGP({ ...bgp, neighbors })}
              prefixListNames={prefixListNames}
              routeMapNames={routeMapNames}
              bfdProfileNames={bfdProfileNames}
              onDirty={onDirty}
            />
          )}
          {subTab === 'afs' && (
            <BGPAddressFamiliesPanel
              afs={bgp.address_families ?? {}}
              onChange={(address_families) => setBGP({ ...bgp, address_families })}
              routeMapNames={routeMapNames}
              vrfNames={vrfNames}
              onDirty={onDirty}
            />
          )}
          {subTab === 'prefix-lists' && (
            <BGPPrefixListsPanel
              prefixLists={prefixLists}
              onChange={setPrefixLists}
              onDirty={onDirty}
            />
          )}
          {subTab === 'route-maps' && (
            <BGPRouteMapsPanel
              routeMaps={routeMaps}
              onChange={setRouteMaps}
              prefixListNames={prefixListNames}
              onDirty={onDirty}
            />
          )}
        </SectionNav>
      )}
    </div>
  )
}

