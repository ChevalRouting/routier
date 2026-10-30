import { migrateBGPPolicies } from '@/lib/bgpPolicies'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Button, EmptyState, PreferencesGroup, SwitchRow } from 'cheval-ui'
import { Route } from 'lucide-react'
import { useEffect } from 'react'
import { BGPConfig, BGPSubTab, PrefixEntry, RouteMapEntry } from '../types'
import { BGPAddressFamiliesPanel } from './AddressFamilies'
import { BGPGeneralPanel } from './GeneralPanel'
import { BGPNeighborsPanel } from './Neighbors'
import { BGPPrefixListsPanel } from './PrefixLists'
import { BGPRouteMapsPanel } from './RouteMaps'

type BGPTabShape = {
  section: BGPSubTab
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
  sharedPrefixLists: string[]
  setSharedPrefixLists: (v: string[]) => void
  sharedRouteMaps: string[]
  setSharedRouteMaps: (v: string[]) => void
}

export function BGPTab({
  section, bgp, setBGP, enabled, setEnabled, vrfNames, bfdProfileNames, onDirty,
  prefixLists, setPrefixLists, routeMaps, setRouteMaps,
  sharedPrefixLists, setSharedPrefixLists, sharedRouteMaps, setSharedRouteMaps,
}: BGPTabShape) {
  useEffect(() => {
    const migrated = migrateBGPPolicies(bgp)
    if (migrated !== bgp) { setBGP(migrated); onDirty() }
  }, [bgp, setBGP, onDirty])

  const prefixListNames = Object.keys(prefixLists)
  const routeMapNames = Object.keys(routeMaps)

  const { data: imported } = useFetch(() => api.apiFriendsBgpPoliciesGet())
  const selectablePrefixListNames = [...prefixListNames, ...(imported?.prefix_lists ?? []).map((p) => p.name)]
  const selectableRouteMapNames = [...routeMapNames, ...(imported?.route_maps ?? []).map((r) => r.name)]

  const enable = (value: boolean) => {
    setEnabled(value)
    onDirty()
  }

  if (!enabled && section !== 'general') {
    return (
      <EmptyState
        icon={<Route />}
        title="BGP Disabled"
        action={<Button variant="suggested" onClick={() => enable(true)}>Enable BGP</Button>}
      />
    )
  }

  return (
    <div className="space-y-6">
      {section === 'general' && (
        <PreferencesGroup>
          <SwitchRow title="BGP" checked={enabled} onCheckedChange={enable} />
        </PreferencesGroup>
      )}
      {enabled && (
        <>
          {section === 'general' && (
            <BGPGeneralPanel bgp={bgp} setBGP={setBGP} onDirty={onDirty} />
          )}
          {section === 'neighbors' && (
            <BGPNeighborsPanel
              neighbors={bgp.neighbors}
              onChange={(neighbors) => setBGP({ ...bgp, neighbors })}
              prefixListNames={selectablePrefixListNames}
              routeMapNames={selectableRouteMapNames}
              bfdProfileNames={bfdProfileNames}
              onDirty={onDirty}
            />
          )}
          {section === 'afs' && (
            <BGPAddressFamiliesPanel
              afs={bgp.address_families ?? {}}
              onChange={(address_families) => setBGP({ ...bgp, address_families })}
              routeMapNames={selectableRouteMapNames}
              vrfNames={vrfNames}
              onDirty={onDirty}
            />
          )}
          {section === 'prefix-lists' && (
            <BGPPrefixListsPanel
              prefixLists={prefixLists}
              onChange={setPrefixLists}
              shared={sharedPrefixLists}
              onSharedChange={setSharedPrefixLists}
              onDirty={onDirty}
            />
          )}
          {section === 'route-maps' && (
            <BGPRouteMapsPanel
              routeMaps={routeMaps}
              onChange={setRouteMaps}
              shared={sharedRouteMaps}
              onSharedChange={setSharedRouteMaps}
              prefixListNames={selectablePrefixListNames}
              onDirty={onDirty}
            />
          )}
        </>
      )}
    </div>
  )
}

