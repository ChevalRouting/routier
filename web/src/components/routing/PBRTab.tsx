import { useState, useEffect } from 'react'
import { useTabState } from 'cheval-ui'
import { SectionNav } from 'cheval-ui'
import { Input } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { Sheet } from 'cheval-ui'
import { Card } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { VarPickerInput } from '@/components/VarPickerInput'
import { PreferencesColumns } from 'cheval-ui'
import { PBRNexthop, PBRNexthopGroup, PBRMapEntry, PBRConfig, PBRSubTab } from './types'
import { newId } from './shared'

export interface PBRNexthopGroupRow { _id: number; name: string; nexthops: (PBRNexthop & { _id: number })[] }
export interface PBRMapRow { _id: number; name: string; entries: (PBRMapEntry & { _id: number })[] }
export interface PBRPolicyRow { _id: number; iface: string; map: string }

export function pbrToRows(pbr: PBRConfig): {
  groups: PBRNexthopGroupRow[]
  maps: PBRMapRow[]
  policies: PBRPolicyRow[]
} {
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
  pbr, setPBR, vrfNames, onDirty,
}: {
  pbr: PBRConfig
  setPBR: (p: PBRConfig) => void
  vrfNames: string[]
  onDirty: () => void
}) {
  const [groups, setGroups] = useState<PBRNexthopGroupRow[]>([])
  const [maps, setMaps] = useState<PBRMapRow[]>([])
  const [policies, setPolicies] = useState<PBRPolicyRow[]>([])
  const [inited, setInited] = useState(false)
  const [subTab, setSubTab] = useTabState<PBRSubTab>('routing.pbr', 'nexthop-groups')
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
    const e: PBRMapEntry & { _id: number } = { _id: newId(), seq: 10, match_src: '', match_dst: '', set_nexthop_group: '', set_nexthop: '' }
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

  const subTabs: { key: PBRSubTab; label: string }[] = [
    { key: 'nexthop-groups', label: groups.length ? `Nexthop Groups (${groups.length})` : 'Nexthop Groups' },
    { key: 'maps', label: maps.length ? `Maps (${maps.length})` : 'Maps' },
    { key: 'policies', label: policies.length ? `Policies (${policies.length})` : 'Policies' },
  ]

  return (
    <div className="space-y-5">
      <p className="text-sm text-muted-foreground">
        Policy-Based Routing (PBR) forwards traffic by policy rather than destination alone: match on
        source/destination and steer matching flows to nexthop groups.
      </p>
      <SectionNav items={subTabs} active={subTab} onChange={setSubTab}>
      {subTab === 'nexthop-groups' && (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={handleAddGroup} className="gap-1.5">
              <Plus className="h-4 w-4" />Add Group
            </Button>
          </div>
          {groups.length === 0 ? (
            <EmptyState className="py-10"
              title="No nexthop groups"
              message="Group nexthops to load-balance or fail over policy-routed traffic."
              action={<Button variant="outline" size="sm" onClick={handleAddGroup} className="gap-2"><Plus className="h-4 w-4" />Add group</Button>}
            />
          ) : (
            <PreferencesColumns>
              {groups.map((g) => (
                <div key={g._id} onClick={() => setOpenGroupId(g._id)} className="flex items-start gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors">
                  <span className="font-mono font-semibold text-sm w-28 shrink-0 truncate pt-0.5">{g.name || <span className="text-muted-foreground italic font-normal">unnamed</span>}</span>
                  <div className="flex flex-wrap gap-1 flex-1 min-w-0">
                    <Badge variant="outline" className="text-xs">{g.nexthops.length} nexthop{g.nexthops.length !== 1 ? 's' : ''}</Badge>
                    {g.nexthops.slice(0, 3).map((nh) => (
                      <Badge key={nh._id} variant="outline" className="text-xs font-mono">{nh.address || nh.dev || nh.nexthop_vrf || '-'}</Badge>
                    ))}
                  </div>
                  <Button variant="ghost" size="sm" onClick={(e) => { e.stopPropagation(); removeGroup(g._id) }} className="h-7 w-7 p-0 hover:text-destructive shrink-0">
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ))}
            </PreferencesColumns>
          )}

          <Sheet open={openGroupId !== null} onClose={handleCloseGroupSheet} title={openGroup?.name || 'New Nexthop Group'}>
            {openGroup && (
              <div className="space-y-4">
                <div className="space-y-1.5">
                  <Label>Name</Label>
                  <Input
                    autoFocus={pendingGroupNew}
                    value={openGroup.name}
                    onChange={(e) => updateGroup(openGroup._id, { name: e.target.value })}
                    placeholder="group-name"
                    className="font-mono text-sm"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Nexthops</Label>
                  {openGroup.nexthops.map((nh) => (
                    <div key={nh._id} className="flex items-center gap-2">
                      <Input value={nh.address ?? ''} onChange={(e) => updateNexthop(openGroup._id, nh._id, { address: e.target.value })} placeholder="address" className="font-mono text-xs h-8 flex-1" />
                      <Input value={nh.dev ?? ''} onChange={(e) => updateNexthop(openGroup._id, nh._id, { dev: e.target.value })} placeholder="interface" className="font-mono text-xs h-8 w-28" />
                      <VarPickerInput value={nh.nexthop_vrf ?? ''} onChange={(v) => updateNexthop(openGroup._id, nh._id, { nexthop_vrf: v })} vars={vrfNames} placeholder="nexthop-vrf" prefix="" label="VRFs" mono className="text-xs h-8" wrapperClassName="w-32" />
                      <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-destructive shrink-0" onClick={() => removeNexthop(openGroup._id, nh._id)}>
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  ))}
                  <Button variant="outline" size="sm" className="gap-1 h-7 text-xs" onClick={() => addNexthop(openGroup._id)}>
                    <Plus className="h-3 w-3" />Add nexthop
                  </Button>
                </div>
              </div>
            )}
          </Sheet>
        </div>
      )}

      {subTab === 'maps' && (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={handleAddMap} className="gap-1.5">
              <Plus className="h-4 w-4" />Add Map
            </Button>
          </div>
          {maps.length === 0 ? (
            <EmptyState className="py-10"
              title="No maps"
              message="Maps hold ordered match/action entries applied to inbound traffic."
              action={<Button variant="outline" size="sm" onClick={handleAddMap} className="gap-2"><Plus className="h-4 w-4" />Add map</Button>}
            />
          ) : (
            <PreferencesColumns>
              {maps.map((m) => (
                <div key={m._id} onClick={() => setOpenMapId(m._id)} className="flex items-start gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors">
                  <span className="font-mono font-semibold text-sm w-28 shrink-0 truncate pt-0.5">{m.name || <span className="text-muted-foreground italic font-normal">unnamed</span>}</span>
                  <div className="flex flex-wrap gap-1 flex-1 min-w-0">
                    <Badge variant="outline" className="text-xs">{m.entries.length} entr{m.entries.length !== 1 ? 'ies' : 'y'}</Badge>
                  </div>
                  <Button variant="ghost" size="sm" onClick={(e) => { e.stopPropagation(); removeMap(m._id) }} className="h-7 w-7 p-0 hover:text-destructive shrink-0">
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ))}
            </PreferencesColumns>
          )}

          <Sheet open={openMapId !== null} onClose={handleCloseMapSheet} title={openMap?.name || 'New Map'} className="max-w-2xl">
            {openMap && (
              <div className="space-y-4">
                <div className="space-y-1.5">
                  <Label>Name</Label>
                  <Input
                    autoFocus={pendingMapNew}
                    value={openMap.name}
                    onChange={(e) => updateMap(openMap._id, { name: e.target.value })}
                    placeholder="map-name"
                    className="font-mono text-sm"
                  />
                </div>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <Label>Entries</Label>
                    <Button variant="outline" size="sm" className="h-7 text-xs gap-1" onClick={() => addEntry(openMap._id)}>
                      <Plus className="h-3 w-3" />Add entry
                    </Button>
                  </div>
                  {openMap.entries.length > 0 && (
                    <div className="rounded bg-card shadow-[var(--card-shadow)] overflow-x-auto">
                      <table className="w-full text-xs">
                        <thead className="bg-muted/50">
                          <tr>
                            <th className="text-left px-2 py-1.5 font-medium text-muted-foreground w-16">Seq</th>
                            <th className="text-left px-2 py-1.5 font-medium text-muted-foreground">Src match</th>
                            <th className="text-left px-2 py-1.5 font-medium text-muted-foreground">Dst match</th>
                            <th className="text-left px-2 py-1.5 font-medium text-muted-foreground">Set nexthop group</th>
                            <th className="text-left px-2 py-1.5 font-medium text-muted-foreground">Set nexthop</th>
                            <th className="w-8" />
                          </tr>
                        </thead>
                        <tbody className="divide-y">
                          {openMap.entries.map((e) => (
                            <tr key={e._id}>
                              <td className="px-2 py-1">
                                <Input value={e.seq || ''} onChange={(ev) => updateEntry(openMap._id, e._id, { seq: Number(ev.target.value) || 0 })} placeholder="10" className="font-mono h-7 border-0 shadow-none focus-visible:ring-1 w-14" />
                              </td>
                              <td className="px-2 py-1">
                                <Input value={e.match_src ?? ''} onChange={(ev) => updateEntry(openMap._id, e._id, { match_src: ev.target.value })} placeholder="10.0.0.0/8" className="font-mono h-7 border-0 shadow-none focus-visible:ring-1" />
                              </td>
                              <td className="px-2 py-1">
                                <Input value={e.match_dst ?? ''} onChange={(ev) => updateEntry(openMap._id, e._id, { match_dst: ev.target.value })} placeholder="0.0.0.0/0" className="font-mono h-7 border-0 shadow-none focus-visible:ring-1" />
                              </td>
                              <td className="px-2 py-1">
                                <VarPickerInput value={e.set_nexthop_group ?? ''} onChange={(v) => updateEntry(openMap._id, e._id, { set_nexthop_group: v, set_nexthop: v ? '' : e.set_nexthop })} vars={groupNames} placeholder="group-name" prefix="" label="Nexthop groups" mono className="h-7 border-0 shadow-none focus-visible:ring-1" />
                              </td>
                              <td className="px-2 py-1">
                                <Input value={e.set_nexthop ?? ''} onChange={(ev) => updateEntry(openMap._id, e._id, { set_nexthop: ev.target.value, set_nexthop_group: ev.target.value ? '' : e.set_nexthop_group })} placeholder="192.168.1.1" className="font-mono h-7 border-0 shadow-none focus-visible:ring-1" />
                              </td>
                              <td className="px-1">
                                <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => removeEntry(openMap._id, e._id)}>
                                  <Trash2 className="h-3 w-3" />
                                </Button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              </div>
            )}
          </Sheet>
        </div>
      )}

      {subTab === 'policies' && (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={addPolicy} className="gap-1.5">
              <Plus className="h-4 w-4" />Add Policy
            </Button>
          </div>
          {policies.length === 0 ? (
            <EmptyState className="py-10"
              title="No policies"
              message="Policies bind a map to an interface."
              action={<Button variant="outline" size="sm" onClick={addPolicy} className="gap-2"><Plus className="h-4 w-4" />Add policy</Button>}
            />
          ) : (
            <div className="grid gap-3 md:grid-cols-2">
              {policies.map((p) => (
                <Card key={p._id} className="space-y-4 p-4">
                  <div className="flex items-center justify-between gap-3">
                    <span className="font-mono text-sm font-semibold">{p.iface || 'New policy'}</span>
                    <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => removePolicy(p._id)}>
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">Interface</Label>
                      <Input value={p.iface} onChange={(e) => updatePolicy(p._id, { iface: e.target.value })} placeholder="eth0" className="font-mono text-sm" />
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">Map</Label>
                      <VarPickerInput value={p.map} onChange={(v) => updatePolicy(p._id, { map: v })} vars={maps.map((m) => m.name)} placeholder="map-name" prefix="" label="Route maps" mono className="text-sm" />
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </div>
      )}
      </SectionNav>
    </div>
  )
}
