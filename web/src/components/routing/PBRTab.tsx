import { plural } from '@/lib/utils'
import { EntityRow } from '@/components/ui/EntityRow'
import { VarPickerInput } from '@/components/VarPickerInput'
import { Badge, Button, EmptyState, Input, Sheet, PreferencesGroup, EntryRow, ComboRow } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { PBRMapEntryCard } from '@/components/routing/PBRTabParts/PBRMapEntryCard'
import { newId } from './shared'
import { PBRConfig, PBRMapEntry, PBRNexthop, PBRNexthopGroup, PBRSubTab } from './types'

type NexthopsShape = { _id: number }

type EntriesShape = { _id: number }

type PbrToRowsShape = {
  groups: PBRNexthopGroupRow[]
  maps: PBRMapRow[]
  policies: PBRPolicyRow[]
}

type PBRTabShape = {
  section: PBRSubTab
  pbr: PBRConfig
  setPBR: (p: PBRConfig) => void
  vrfNames: string[]
  onDirty: () => void
}

type EShape = { _id: number }


export interface PBRNexthopGroupRow { _id: number; name: string; nexthops: (PBRNexthop & NexthopsShape)[] }
export interface PBRMapRow { _id: number; name: string; entries: (PBRMapEntry & EntriesShape)[] }
export interface PBRPolicyRow { _id: number; iface: string; map: string }

export function pbrToRows(pbr: PBRConfig): PbrToRowsShape {
  let id = 0
  const groups: PBRNexthopGroupRow[] = Object.entries(pbr.nexthop_groups ?? {}).map(([name, g]) => ({
    _id: ++id,
    name,
    nexthops: g.nexthops.map((nh) => ({ ...nh, _id: ++id })),
  }))
  const maps: PBRMapRow[] = Object.entries(pbr.maps ?? {}).map(([name, entries]) => ({
    _id: ++id,
    name,
    entries: entries.map((e) => ({ ...e, _id: ++id })),
  }))
  const policies: PBRPolicyRow[] = Object.entries(pbr.policies ?? {}).map(([iface, map]) => ({
    _id: ++id,
    iface,
    map,
  }))
  return { groups, maps, policies }
}

export function rowsToPBR(
  groups: PBRNexthopGroupRow[],
  maps: PBRMapRow[],
  policies: PBRPolicyRow[],
): PBRConfig {
  const nexthop_groups: Record<string, PBRNexthopGroup> = {}
  for (const g of groups) {
    if (!g.name) continue
    nexthop_groups[g.name] = {
      nexthops: g.nexthops.map(({ _id: _i, ...nh }) => nh).filter((nh) => nh.address || nh.dev),
    }
  }
  const mapsOut: Record<string, PBRMapEntry[]> = {}
  for (const m of maps) {
    if (!m.name) continue
    mapsOut[m.name] = m.entries
      .filter((e) => e.seq > 0)
      .map(({ _id: _i, ...e }) => e)
  }
  const policiesOut: Record<string, string> = {}
  for (const p of policies) {
    if (p.iface && p.map) policiesOut[p.iface] = p.map
  }
  return { nexthop_groups, maps: mapsOut, policies: policiesOut }
}

export function PBRTab({
  section: subTab, pbr, setPBR, vrfNames, onDirty,
}: PBRTabShape) {
  const [groups, setGroups] = useState<PBRNexthopGroupRow[]>([])
  const [maps, setMaps] = useState<PBRMapRow[]>([])
  const [policies, setPolicies] = useState<PBRPolicyRow[]>([])
  const [inited, setInited] = useState(false)
  const [openGroupId, setOpenGroupId] = useState<number | null>(null)
  const [openMapId, setOpenMapId] = useState<number | null>(null)
  const [pendingGroupNew, setPendingGroupNew] = useState(false)
  const [pendingMapNew, setPendingMapNew] = useState(false)

  useEffect(() => {
    if (!inited) {
      const r = pbrToRows(pbr)
      setGroups(r.groups)
      setMaps(r.maps)
      setPolicies(r.policies)
      setInited(true)
    }
  }, [pbr, inited])

  const sync = (g: PBRNexthopGroupRow[], m: PBRMapRow[], p: PBRPolicyRow[]) => {
    setPBR(rowsToPBR(g, m, p))
    onDirty()
  }

  const setGroupsAndSync = (next: PBRNexthopGroupRow[]) => { setGroups(next); sync(next, maps, policies) }
  const setMapsAndSync = (next: PBRMapRow[]) => { setMaps(next); sync(groups, next, policies) }
  const setPoliciesAndSync = (next: PBRPolicyRow[]) => { setPolicies(next); sync(groups, maps, next) }

  const handleAddGroup = () => {
    const g: PBRNexthopGroupRow = { _id: newId(), name: '', nexthops: [] }
    const next = [...groups, g]
    setGroups(next)
    setOpenGroupId(g._id)
    setPendingGroupNew(true)
  }
  const handleCloseGroupSheet = () => {
    if (pendingGroupNew && openGroupId !== null) {
      const g = groups.find((x) => x._id === openGroupId)
      if (!g?.name?.trim()) {
        setGroupsAndSync(groups.filter((x) => x._id !== openGroupId))
      } else {
        sync(groups, maps, policies)
      }
    }
    setOpenGroupId(null)
    setPendingGroupNew(false)
  }
  const updateGroup = (id: number, patch: Partial<PBRNexthopGroupRow>) => {
    const next = groups.map((g) => g._id === id ? { ...g, ...patch } : g)
    setGroups(next)
    if (!pendingGroupNew) sync(next, maps, policies)
  }
  const removeGroup = (id: number) => {
    setGroupsAndSync(groups.filter((g) => g._id !== id))
    if (openGroupId === id) { setOpenGroupId(null); setPendingGroupNew(false) }
  }
  const addNexthop = (gid: number) => {
    const next = groups.map((g) =>
      g._id === gid ? { ...g, nexthops: [...g.nexthops, { _id: newId(), address: '', dev: '', nexthop_vrf: '' }] } : g
    )
    setGroups(next)
    if (!pendingGroupNew) sync(next, maps, policies)
  }
  const removeNexthop = (gid: number, nhId: number) => {
    const next = groups.map((g) =>
      g._id === gid ? { ...g, nexthops: g.nexthops.filter((nh) => nh._id !== nhId) } : g
    )
    setGroups(next)
    if (!pendingGroupNew) sync(next, maps, policies)
  }
  const updateNexthop = (gid: number, nhId: number, patch: Partial<PBRNexthop>) => {
    const next = groups.map((g) =>
      g._id === gid ? { ...g, nexthops: g.nexthops.map((nh) => nh._id === nhId ? { ...nh, ...patch } : nh) } : g
    )
    setGroups(next)
    if (!pendingGroupNew) sync(next, maps, policies)
  }

  const handleAddMap = () => {
    const m: PBRMapRow = { _id: newId(), name: '', entries: [] }
    const next = [...maps, m]
    setMaps(next)
    setOpenMapId(m._id)
    setPendingMapNew(true)
  }
  const handleCloseMapSheet = () => {
    if (pendingMapNew && openMapId !== null) {
      const m = maps.find((x) => x._id === openMapId)
      if (!m?.name?.trim()) {
        setMapsAndSync(maps.filter((x) => x._id !== openMapId))
      } else {
        sync(groups, maps, policies)
      }
    }
    setOpenMapId(null)
    setPendingMapNew(false)
  }
  const updateMap = (id: number, patch: Partial<PBRMapRow>) => {
    const next = maps.map((m) => m._id === id ? { ...m, ...patch } : m)
    setMaps(next)
    if (!pendingMapNew) sync(groups, next, policies)
  }
  const removeMap = (id: number) => {
    setMapsAndSync(maps.filter((m) => m._id !== id))
    if (openMapId === id) { setOpenMapId(null); setPendingMapNew(false) }
  }
  const addEntry = (mid: number) => {
    const seq = Math.max(0, ...(maps.find((m) => m._id === mid)?.entries.map((x) => x.seq) ?? [])) + 10
    const e: PBRMapEntry & EShape = { _id: newId(), seq, match_src: '', match_dst: '', set_nexthop_group: '', set_nexthop: '', set_vrf: '' }
    const next = maps.map((m) => m._id === mid ? { ...m, entries: [...m.entries, e] } : m)
    setMaps(next)
    if (!pendingMapNew) sync(groups, next, policies)
  }
  const removeEntry = (mid: number, eid: number) => {
    const next = maps.map((m) => m._id === mid ? { ...m, entries: m.entries.filter((e) => e._id !== eid) } : m)
    setMapsAndSync(next)
  }
  const updateEntry = (mid: number, eid: number, patch: Partial<PBRMapEntry>) => {
    const next = maps.map((m) =>
      m._id === mid ? { ...m, entries: m.entries.map((e) => e._id === eid ? { ...e, ...patch } : e) } : m
    )
    setMaps(next)
    if (!pendingMapNew) sync(groups, next, policies)
  }

  const addPolicy = () => setPoliciesAndSync([...policies, { _id: newId(), iface: '', map: '' }])
  const removePolicy = (id: number) => setPoliciesAndSync(policies.filter((p) => p._id !== id))
  const updatePolicy = (id: number, patch: Partial<PBRPolicyRow>) =>
    setPoliciesAndSync(policies.map((p) => p._id === id ? { ...p, ...patch } : p))

  const groupNames = groups.map((g) => g.name).filter(Boolean)
  const openGroup = openGroupId !== null ? groups.find((g) => g._id === openGroupId) ?? null : null
  const openMap = openMapId !== null ? maps.find((m) => m._id === openMapId) ?? null : null

  return (
    <div className="space-y-5">
      {subTab === 'nexthop-groups' && (
        <div className="space-y-4">
          {groups.length === 0 ? (
            <EmptyState className="py-10"
              title="No Nexthop Groups"
              action={<Button variant="outline" size="sm" onClick={handleAddGroup} className="gap-1.5"><Plus className="h-4 w-4" />Add Group…</Button>}
            />
          ) : (
            <PreferencesGroup
              title="Nexthop Groups"
              header={<Button variant="outline" size="sm" onClick={handleAddGroup} className="gap-1.5"><Plus className="h-4 w-4" />Add Group…</Button>}
            >
              {groups.map((g) => (
                <EntityRow key={g._id} title={g.name} onOpen={() => setOpenGroupId(g._id)} onRemove={() => removeGroup(g._id)} removeLabel="Remove Group">
                  <Badge variant="outline">{plural(g.nexthops.length, 'nexthop')}</Badge>
                  {g.nexthops.slice(0, 3).map((nh) => (
                    <Badge key={nh._id} variant="outline" className="font-mono">{nh.address || nh.dev || nh.nexthop_vrf || '-'}</Badge>
                  ))}
                </EntityRow>
              ))}
            </PreferencesGroup>
          )}

          <Sheet open={openGroupId !== null} onClose={handleCloseGroupSheet} title={openGroup?.name || 'New Nexthop Group'}>
            {openGroup && (
              <div className="space-y-6">
                <PreferencesGroup>
                  <EntryRow title="Name" autoFocus={pendingGroupNew} value={openGroup.name} onChange={(e) => updateGroup(openGroup._id, { name: e.target.value })} placeholder="group-name" className="font-mono" />
                </PreferencesGroup>
                <PreferencesGroup
                  title="Nexthops"
                  header={<Button variant="outline" size="sm" className="gap-1.5" onClick={() => addNexthop(openGroup._id)}><Plus className="h-4 w-4" />Add Nexthop</Button>}
                >
                  {openGroup.nexthops.length === 0 && <p className="px-4 py-3 text-sm text-muted-foreground">No nexthops</p>}
                  {openGroup.nexthops.map((nh) => (
                    <div key={nh._id} className="flex flex-wrap items-center gap-2 px-4 py-2 sm:flex-nowrap">
                      <Input value={nh.address ?? ''} onChange={(e) => updateNexthop(openGroup._id, nh._id, { address: e.target.value })} placeholder="address" aria-label="Address" className="min-w-0 flex-1 basis-64 font-mono" />
                      <Input value={nh.dev ?? ''} onChange={(e) => updateNexthop(openGroup._id, nh._id, { dev: e.target.value })} placeholder="interface" aria-label="Interface" className="w-32 font-mono" />
                      <VarPickerInput value={nh.nexthop_vrf ?? ''} onChange={(v) => updateNexthop(openGroup._id, nh._id, { nexthop_vrf: v })} vars={vrfNames} placeholder="nexthop-vrf" prefix="" label="VRFs" mono wrapperClassName="w-44" />
                      <Button variant="ghost" size="icon" className="shrink-0 text-muted-foreground hover:text-destructive" onClick={() => removeNexthop(openGroup._id, nh._id)} aria-label="Remove Nexthop" title="Remove Nexthop">
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                </PreferencesGroup>
              </div>
            )}
          </Sheet>
        </div>
      )}

      {subTab === 'maps' && (
        <div className="space-y-4">
          {maps.length === 0 ? (
            <EmptyState className="py-10"
              title="No Maps"
              action={<Button variant="outline" size="sm" onClick={handleAddMap} className="gap-1.5"><Plus className="h-4 w-4" />Add Map…</Button>}
            />
          ) : (
            <PreferencesGroup
              title="Maps"
              header={<Button variant="outline" size="sm" onClick={handleAddMap} className="gap-1.5"><Plus className="h-4 w-4" />Add Map…</Button>}
            >
              {maps.map((m) => (
                <EntityRow key={m._id} title={m.name} onOpen={() => setOpenMapId(m._id)} onRemove={() => removeMap(m._id)} removeLabel="Remove Map">
                  <Badge variant="outline">{plural(m.entries.length, 'entry', 'entries')}</Badge>
                </EntityRow>
              ))}
            </PreferencesGroup>
          )}

          <Sheet open={openMapId !== null} onClose={handleCloseMapSheet} title={openMap?.name || 'New Map'}>
            {openMap && (
              <div className="space-y-6">
                <PreferencesGroup>
                  <EntryRow title="Name" autoFocus={pendingMapNew} value={openMap.name} onChange={(e) => updateMap(openMap._id, { name: e.target.value })} placeholder="map-name" className="font-mono" />
                </PreferencesGroup>
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <h2 className="text-sm font-semibold">Entries</h2>
                    <Button variant="outline" size="sm" className="gap-1.5" onClick={() => addEntry(openMap._id)}>
                      <Plus className="h-4 w-4" />Add Entry
                    </Button>
                  </div>
                  {openMap.entries.map((e) => (
                    <PBRMapEntryCard
                      key={e._id}
                      entry={e}
                      groupNames={groupNames}
                      vrfNames={vrfNames}
                      onChange={(patch) => updateEntry(openMap._id, e._id, patch)}
                      onRemove={() => removeEntry(openMap._id, e._id)}
                    />
                  ))}
                </div>
              </div>
            )}
          </Sheet>
        </div>
      )}

      {subTab === 'policies' && (
        <div className="space-y-4">
          {policies.length === 0 ? (
            <EmptyState className="py-10"
              title="No Policies"
              action={<Button variant="outline" size="sm" onClick={addPolicy} className="gap-1.5"><Plus className="h-4 w-4" />Add Policy</Button>}
            />
          ) : (
            <div className="space-y-6">
              <div className="flex justify-end">
                <Button variant="outline" size="sm" onClick={addPolicy} className="gap-1.5"><Plus className="h-4 w-4" />Add Policy</Button>
              </div>
              {policies.map((p) => (
                <PreferencesGroup
                  key={p._id}
                  title={p.iface || 'New Policy'}
                  header={(
                    <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-destructive" onClick={() => removePolicy(p._id)} aria-label="Remove Policy" title="Remove Policy">
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                >
                  <EntryRow title="Interface" value={p.iface} onChange={(e) => updatePolicy(p._id, { iface: e.target.value })} placeholder="eth0" className="font-mono" />
                  <ComboRow title="Map">
                    <VarPickerInput value={p.map} onChange={(v) => updatePolicy(p._id, { map: v })} vars={maps.map((m) => m.name)} placeholder="map-name" prefix="" label="Route Maps" mono />
                  </ComboRow>
                </PreferencesGroup>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
