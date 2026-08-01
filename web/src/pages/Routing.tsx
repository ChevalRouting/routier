import React, { useState, useEffect } from 'react'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { useDataRefresh } from '@/lib/dataVersion'
import { useTabState } from '@/lib/useTabState'
import { Tabs } from '@/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { PageHeader } from '@/components/PageHeader'
import SaveButton from '@/components/SaveButton'
import { Spinner } from '@/components/Spinner'
import RadvdTab, { RADVDConfig } from '@/components/routing/RadvdTab'
import NatPanel, { type NatSaveState } from '@/components/routing/NatPanel'
import {
  RoutingConfig, BGPConfig, OSPFConfig, OSPF6Config, PBRConfig, BFDProfile, VRFRouting, defaultBGP, defaultOSPF, defaultOSPF6,
} from '@/components/routing/types'
import { newId } from '@/components/routing/shared'
import { StaticRoutesTab, RouteRow } from '@/components/routing/StaticRoutesTab'
import { BGPTab } from '@/components/routing/bgp/BGPTab'
import { BFDTab } from '@/components/routing/BFDTab'
import { OSPFTab } from '@/components/routing/ospf/OSPFTab'
import { OSPF6Tab } from '@/components/routing/ospf/OSPF6Tab'
import { PBRTab } from '@/components/routing/PBRTab'
import { VRFsTab, VRFRow } from '@/components/routing/VRFsTab'

type Tab = 'static' | 'bgp' | 'ospf' | 'ospf6' | 'bfd' | 'radvd' | 'vrfs' | 'pbr' | 'nat'

const TABS: { key: Tab; label: string }[] = [
  { key: 'static', label: 'Static Routes' },
  { key: 'bgp', label: 'BGP' },
  { key: 'ospf', label: 'OSPF' },
  { key: 'ospf6', label: 'OSPFv3' },
  { key: 'bfd', label: 'BFD' },
  { key: 'radvd', label: 'RADVD' },
  { key: 'vrfs', label: 'VRFs' },
  { key: 'pbr', label: 'PBR' },
  { key: 'nat', label: 'NAT' },
]

export default function Routing() {
  const { data, isLoading } = useFetch<RoutingConfig>(
    () => api.apiConfigSectionGet({ section: 'routing' }) as Promise<RoutingConfig>
  )
  const { data: vrfsData } = useFetch<Record<string, { table: number }>>(
    () => api.apiConfigSectionGet({ section: 'vrfs' }) as Promise<Record<string, { table: number }>>
  )
  const { isDirty, markDirty, save, saving } = usePageSave('routing')

  const [activeTab, setActiveTab] = useTabState<Tab>('routing', 'static')
  const [initialized, setInitialized] = useState(false)
  const [vrfsInitialized, setVrfsInitialized] = useState(false)
  const [refreshKey, setRefreshKey] = useState(0)
  useDataRefresh(() => { setInitialized(false); setVrfsInitialized(false) })

  const [natState, setNatState] = useState<NatSaveState | null>(null)
  const [staticRows, setStaticRows] = useState<RouteRow[]>([])
  const [bgpEnabled, setBGPEnabled] = useState(false)
  const [bgp, setBGP] = useState<BGPConfig>(defaultBGP())
  const [ospfEnabled, setOSPFEnabled] = useState(false)
  const [ospf, setOSPF] = useState<OSPFConfig>(defaultOSPF())
  const [ospf6Enabled, setOSPF6Enabled] = useState(false)
  const [ospf6, setOSPF6] = useState<OSPF6Config>(defaultOSPF6())
  const [radvd, setRadvd] = useState<RADVDConfig | null>(null)
  const [pbr, setPBR] = useState<PBRConfig>({})
  const [bfdProfiles, setBfdProfiles] = useState<BFDProfile[]>([])

  const [vrfRows, setVrfRows] = useState<VRFRow[]>([])

  const [vrfContext, setVrfContext] = useState<string>('global')
  const [vrfRoutings, setVrfRoutings] = useState<Record<string, VRFRouting>>({})

  const { data: ifacesData } = useFetch<Record<string, { vlans?: Record<string, unknown> }>>(
    () => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<Record<string, { vlans?: Record<string, unknown> }>>
  )
  const ifaceNames = [
    ...Object.keys(ifacesData ?? {}),
    ...Object.values(ifacesData ?? {}).flatMap((iface) => Object.keys(iface?.vlans ?? {})),
  ]

  const vrfNames = vrfRows.map((r) => r.name).filter(Boolean)

  useEffect(() => {
    if (data && !initialized) {
      setStaticRows((data.static ?? []).map((r) => ({ ...r, id: newId() })))
      setBGPEnabled(!!data.bgp)
      setBGP(data.bgp ? (data.bgp as BGPConfig) : defaultBGP())
      setOSPFEnabled(!!data.ospf)
      setOSPF(data.ospf ?? defaultOSPF())
      setOSPF6Enabled(!!data.ospf6)
      setOSPF6(data.ospf6 ?? defaultOSPF6())
      setRadvd((data.radvd as RADVDConfig) ?? null)
      setVrfRoutings((data.vrfs as Record<string, VRFRouting>) ?? {})
      setPBR(data.pbr ?? {})
      setBfdProfiles(data.bfd?.profiles ?? [])
      setInitialized(true)
      setRefreshKey((k) => k + 1)
    }
  }, [data, initialized])

  useEffect(() => {
    if (vrfsData && !vrfsInitialized) {
      setVrfRows(
        Object.entries(vrfsData).map(([name, cfg]) => ({ _id: newId(), name, table: cfg.table }))
      )
      setVrfsInitialized(true)
    }
  }, [vrfsData, vrfsInitialized])

  const getVRFBGPEnabled = (): boolean => {
    if (vrfContext === 'global') return bgpEnabled
    return !!vrfRoutings[vrfContext]?.bgp
  }
  const setVRFBGPEnabled = (v: boolean) => {
    if (vrfContext === 'global') { setBGPEnabled(v); return }
    setVrfRoutings((prev) => {
      const current = prev[vrfContext] ?? {}
      if (v) {
        return { ...prev, [vrfContext]: { ...current, bgp: current.bgp ?? defaultBGP() } }
      } else {
        const { bgp: _bgp, ...rest } = current
        return { ...prev, [vrfContext]: rest }
      }
    })
  }
  const getVRFBGP = (): BGPConfig => {
    if (vrfContext === 'global') return bgp
    return vrfRoutings[vrfContext]?.bgp ?? defaultBGP()
  }
  const setVRFBGP = (v: BGPConfig) => {
    if (vrfContext === 'global') { setBGP(v); return }
    setVrfRoutings((prev) => ({
      ...prev,
      [vrfContext]: { ...(prev[vrfContext] ?? {}), bgp: v },
    }))
  }

  const getVRFOSPFEnabled = (): boolean => {
    if (vrfContext === 'global') return ospfEnabled
    return !!vrfRoutings[vrfContext]?.ospf
  }
  const setVRFOSPFEnabled = (v: boolean) => {
    if (vrfContext === 'global') { setOSPFEnabled(v); return }
    setVrfRoutings((prev) => {
      const current = prev[vrfContext] ?? {}
      if (v) {
        return { ...prev, [vrfContext]: { ...current, ospf: current.ospf ?? defaultOSPF() } }
      } else {
        const { ospf: _ospf, ...rest } = current
        return { ...prev, [vrfContext]: rest }
      }
    })
  }
  const getVRFOSPF = (): OSPFConfig => {
    if (vrfContext === 'global') return ospf
    return vrfRoutings[vrfContext]?.ospf ?? defaultOSPF()
  }
  const setVRFOSPF = (v: OSPFConfig) => {
    if (vrfContext === 'global') { setOSPF(v); return }
    setVrfRoutings((prev) => ({
      ...prev,
      [vrfContext]: { ...(prev[vrfContext] ?? {}), ospf: v },
    }))
  }

  const getVRFOSPF6Enabled = (): boolean => {
    if (vrfContext === 'global') return ospf6Enabled
    return !!vrfRoutings[vrfContext]?.ospf6
  }
  const setVRFOSPF6Enabled = (v: boolean) => {
    if (vrfContext === 'global') { setOSPF6Enabled(v); return }
    setVrfRoutings((prev) => {
      const current = prev[vrfContext] ?? {}
      if (v) {
        return { ...prev, [vrfContext]: { ...current, ospf6: current.ospf6 ?? defaultOSPF6() } }
      } else {
        const { ospf6: _ospf6, ...rest } = current
        return { ...prev, [vrfContext]: rest }
      }
    })
  }
  const getVRFOSPF6 = (): OSPF6Config => {
    if (vrfContext === 'global') return ospf6
    return vrfRoutings[vrfContext]?.ospf6 ?? defaultOSPF6()
  }
  const setVRFOSPF6 = (v: OSPF6Config) => {
    if (vrfContext === 'global') { setOSPF6(v); return }
    setVrfRoutings((prev) => ({
      ...prev,
      [vrfContext]: { ...(prev[vrfContext] ?? {}), ospf6: v },
    }))
  }

  const getVRFStaticRows = (): RouteRow[] => {
    if (vrfContext === 'global') return staticRows
    return (vrfRoutings[vrfContext]?.static ?? []).map((r, i) => ({ ...r, id: i }))
  }
  const setVRFStaticRows: React.Dispatch<React.SetStateAction<RouteRow[]>> = (action) => {
    if (vrfContext === 'global') {
      setStaticRows(action)
      return
    }
    setVrfRoutings((prev) => {
      const current = prev[vrfContext] ?? {}
      const prevRows = (current.static ?? []).map((r, i) => ({ ...r, id: i }))
      const next = typeof action === 'function' ? action(prevRows) : action
      return {
        ...prev,
        [vrfContext]: {
          ...current,
          static: next.map(({ id: _id, ...rest }) => rest),
        },
      }
    })
  }

  const buildPayload = (): RoutingConfig => {
    const payload: RoutingConfig = {
      static: staticRows.map(({ id: _id, ...rest }) => rest),
    }
    if (bgpEnabled) payload.bgp = bgp
    if (ospfEnabled) payload.ospf = ospf
    if (ospf6Enabled) payload.ospf6 = ospf6
    if (radvd) payload.radvd = radvd
    if (data?.anycast) payload.anycast = data.anycast
    if (
      Object.keys(pbr.nexthop_groups ?? {}).length > 0 ||
      Object.keys(pbr.maps ?? {}).length > 0 ||
      Object.keys(pbr.policies ?? {}).length > 0
    ) payload.pbr = pbr
    if (bfdProfiles.length > 0) payload.bfd = { profiles: bfdProfiles }
    const vrfs: Record<string, VRFRouting> = {}
    for (const [name, vr] of Object.entries(vrfRoutings)) {
      if (vr.bgp || vr.ospf || vr.ospf6 || (vr.static && vr.static.length > 0)) {
        vrfs[name] = vr
      }
    }
    if (Object.keys(vrfs).length > 0) payload.vrfs = vrfs
    return payload
  }

  const handleSave = async () => {
    const vrfsPayload: Record<string, { table: number }> = {}
    for (const { name, table } of vrfRows) {
      if (name) vrfsPayload[name] = { table }
    }
    await api.apiConfigSectionPut({ section: 'vrfs', body: vrfsPayload })
    save(buildPayload())
  }

  if (isLoading) {
    return (
      <Spinner />
    )
  }

  const showVrfContext = activeTab === 'static' || activeTab === 'bgp' || activeTab === 'ospf' || activeTab === 'ospf6'

  return (
    <div className="space-y-6">
      <PageHeader title="Routing" description="Manage static routes, BGP, OSPF, RADVD, VRFs, and PBR" action={
        activeTab === 'nat'
          ? natState && <SaveButton isDirty={natState.isDirty} saving={natState.saving} onClick={natState.save} onCancel={natState.cancel} />
          : <SaveButton isDirty={isDirty} saving={saving} onClick={handleSave} />
      } />

      <Tabs tabs={TABS} active={activeTab} onChange={setActiveTab} />

      {showVrfContext && (
        <div className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">VRF context:</span>
          <Select value={vrfContext} onValueChange={setVrfContext}>
            <SelectTrigger className="h-7 w-36 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="global">global</SelectItem>
              {vrfNames.map((name) => (
                <SelectItem key={name} value={name}>{name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      <div className="pt-2" key={refreshKey}>
        {activeTab === 'static' && (
          <StaticRoutesTab
            rows={vrfContext === 'global' ? staticRows : getVRFStaticRows()}
            setRows={setVRFStaticRows}
            onDirty={markDirty}
          />
        )}
        {activeTab === 'bgp' && (
          <BGPTab
            bgp={getVRFBGP()}
            setBGP={setVRFBGP}
            enabled={getVRFBGPEnabled()}
            setEnabled={setVRFBGPEnabled}
            vrfNames={vrfNames}
            bfdProfileNames={bfdProfiles.map((p) => p.name).filter(Boolean)}
            onDirty={markDirty}
          />
        )}
        {activeTab === 'bfd' && (
          <BFDTab profiles={bfdProfiles} setProfiles={setBfdProfiles} onDirty={markDirty} />
        )}
        {activeTab === 'ospf' && (
          <OSPFTab
            ospf={getVRFOSPF()}
            setOSPF={setVRFOSPF}
            enabled={getVRFOSPFEnabled()}
            setEnabled={setVRFOSPFEnabled}
            onDirty={markDirty}
            ifaceNames={ifaceNames}
          />
        )}
        {activeTab === 'ospf6' && (
          <OSPF6Tab
            ospf6={getVRFOSPF6()}
            setOSPF6={setVRFOSPF6}
            enabled={getVRFOSPF6Enabled()}
            setEnabled={setVRFOSPF6Enabled}
            onDirty={markDirty}
            ifaceNames={ifaceNames}
          />
        )}
        {activeTab === 'radvd' && (
          <RadvdTab radvd={radvd} onChange={setRadvd} onDirty={markDirty} ifaceNames={ifaceNames} />
        )}
        {activeTab === 'vrfs' && (
          <VRFsTab rows={vrfRows} setRows={setVrfRows} onDirty={markDirty} />
        )}
        {activeTab === 'pbr' && (
          <PBRTab pbr={pbr} setPBR={setPBR} vrfNames={vrfNames} onDirty={markDirty} />
        )}
        {activeTab === 'nat' && <NatPanel onStateChange={setNatState} />}
      </div>
    </div>
  )
}
