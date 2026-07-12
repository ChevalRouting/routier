import { useTabState } from '@/lib/useTabState'
import { SectionNav } from '@/components/ui/section-nav'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { BGPConfig, BGPSubTab } from '../types'
import { BGPGeneralPanel } from './GeneralPanel'
import { BGPNeighborsPanel } from './Neighbors'
import { BGPAddressFamiliesPanel } from './AddressFamilies'
import { BGPPrefixListsPanel } from './PrefixLists'
import { BGPRouteMapsPanel } from './RouteMaps'

export function BGPTab({
  bgp, setBGP, enabled, setEnabled, vrfNames, bfdProfileNames, onDirty,
}: {
  bgp: BGPConfig
  setBGP: (v: BGPConfig) => void
  enabled: boolean
  setEnabled: (v: boolean) => void
  vrfNames: string[]
  bfdProfileNames: string[]
  onDirty: () => void
}) {
  const [subTab, setSubTab] = useTabState<BGPSubTab>('routing.bgp', 'general')

  const prefixListNames = Object.keys(bgp.prefix_lists ?? {})
  const routeMapNames = Object.keys(bgp.route_maps ?? {})

  const subTabs: { key: BGPSubTab; label: string }[] = [
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
              prefixLists={bgp.prefix_lists ?? {}}
              onChange={(prefix_lists) => setBGP({ ...bgp, prefix_lists })}
              onDirty={onDirty}
            />
          )}
          {subTab === 'route-maps' && (
            <BGPRouteMapsPanel
              routeMaps={bgp.route_maps ?? {}}
              onChange={(route_maps) => setBGP({ ...bgp, route_maps })}
              prefixListNames={prefixListNames}
              onDirty={onDirty}
            />
          )}
        </SectionNav>
      )}
    </div>
  )
}

