import { useState } from 'react'
import { checkIP, checkASN } from '@/lib/validate'
import { Input } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { Switch } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { TagInput } from 'cheval-ui'
import { Card } from 'cheval-ui'
import { Sheet } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { PreferencesGroup, PreferencesColumns, EntryRow, ComboRow, SwitchRow } from 'cheval-ui'
import { BGPNeighborAF, BGPNeighbor, STANDARD_AFS } from '../types'
import { NameSelect } from '../shared'

export function NeighborAFEditor({
  afs, onChange, prefixListNames, routeMapNames, onDirty,
}: {
  afs: Record<string, BGPNeighborAF>
  onChange: (v: Record<string, BGPNeighborAF>) => void
  prefixListNames: string[]
  routeMapNames: string[]
  onDirty: () => void
}) {
  const [adding, setAdding] = useState('')
  const configured = Object.keys(afs)
  const available = STANDARD_AFS.filter((af) => !configured.includes(af))

  const upd = (name: string, patch: Partial<BGPNeighborAF>) => {
    onChange({ ...afs, [name]: { ...afs[name], ...patch } })
    onDirty()
  }
  const remove = (name: string) => {
    const next = { ...afs }; delete next[name]; onChange(next); onDirty()
  }
  const add = () => {
    if (!adding || adding in afs) return
    onChange({ ...afs, [adding]: {} }); setAdding(''); onDirty()
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-semibold">Per-neighbor Address Families</Label>
        {available.length > 0 && (
          <div className="flex items-center gap-1.5">
            <Select value={adding} onValueChange={setAdding}>
              <SelectTrigger className="h-7 text-xs w-40">
                <SelectValue placeholder="Select AF…" />
              </SelectTrigger>
              <SelectContent>
                {available.map((af) => (
                  <SelectItem key={af} value={af}>{af}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button variant="outline" size="sm" className="h-7 text-xs gap-1" onClick={add} disabled={!adding}>
              <Plus className="h-3 w-3" />Add
            </Button>
          </div>
        )}
      </div>

      {configured.length === 0 ? (
        <p className="text-xs text-muted-foreground italic">No address families configured for this neighbor.</p>
      ) : (
        <div className="space-y-2">
          {configured.map((afName) => {
            const af = afs[afName] ?? {}
            return (
              <Card key={afName} className="p-3 space-y-3">
                <div className="flex items-center justify-between">
                  <Badge variant="outline" className="font-mono text-xs">{afName}</Badge>
                  <Button variant="ghost" size="icon" className="h-6 w-6 text-muted-foreground hover:text-destructive" onClick={() => remove(afName)}>
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
                <div className="flex flex-col gap-2">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs">Activate in this address family</span>
                    <Switch checked={!af.disabled} onCheckedChange={(v) => upd(afName, { disabled: !v })} />
                  </div>
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs">Soft reconfiguration inbound</span>
                    <Switch checked={!!af.soft_reconfiguration} onCheckedChange={(v) => upd(afName, { soft_reconfiguration: v })} />
                  </div>
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs">Advertise default route (default-originate)</span>
                    <Switch checked={!!af.default_originate} onCheckedChange={(v) => upd(afName, { default_originate: v })} />
                  </div>
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs">Next-hop-self</span>
                    <Switch checked={!!af.next_hop_self} onCheckedChange={(v) => upd(afName, { next_hop_self: v })} />
                  </div>
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs">Route-reflector client</span>
                    <Switch checked={!!af.route_reflector_client} onCheckedChange={(v) => upd(afName, { route_reflector_client: v })} />
                  </div>
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs">Remove private ASNs (remove-private-AS)</span>
                    <Switch checked={!!af.remove_private_as} onCheckedChange={(v) => upd(afName, { remove_private_as: v })} />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1.5">
                    <Label className="text-xs text-muted-foreground">allowas-in (0 = off)</Label>
                    <Input value={af.allowas_in ?? ''} onChange={(e) => upd(afName, { allowas_in: Number(e.target.value) || undefined })} placeholder="0" className="font-mono text-xs h-8" />
                  </div>
                  <div className="space-y-1.5">
                    <Label className="text-xs text-muted-foreground">Weight (0 = default)</Label>
                    <Input value={af.weight ?? ''} onChange={(e) => upd(afName, { weight: Number(e.target.value) || undefined })} placeholder="0" className="font-mono text-xs h-8" />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  {([
                    ['prefix_list_in', 'Prefix List In', prefixListNames],
                    ['prefix_list_out', 'Prefix List Out', prefixListNames],
                    ['route_map_in', 'Route Map In', routeMapNames],
                    ['route_map_out', 'Route Map Out', routeMapNames],
                  ] as const).map(([field, label, names]) => (
                    <div key={field} className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">{label}</Label>
                      <NameSelect
                        value={af[field]}
                        onChange={(v) => upd(afName, { [field]: v })}
                        names={names as string[]}
                      />
                    </div>
                  ))}
                </div>
                <div className="space-y-1.5">
                  <Label className="text-xs text-muted-foreground">Additional directives (per address-family)</Label>
                  <TagInput values={af.extra ?? []} onChange={(v) => upd(afName, { extra: v.length ? v : undefined })} placeholder="send-community extended" mono />
                </div>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}

export function emptyNeighbor(): BGPNeighbor {
  return { address: '', remote_asn: 0, address_families: {} }
}

export function NeighborForm({
  neighbor, onChange, prefixListNames, routeMapNames, bfdProfileNames, onAdd, onDone, onDirty,
}: {
  neighbor: BGPNeighbor
  onChange: (v: BGPNeighbor) => void
  prefixListNames: string[]
  routeMapNames: string[]
  bfdProfileNames: string[]
  onAdd?: () => void
  onDone: () => void
  onDirty: () => void
}) {
  const set = (patch: Partial<BGPNeighbor>) => onChange({ ...neighbor, ...patch })

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        <EntryRow title="Peer address" autoFocus value={neighbor.address} onChange={(e) => set({ address: e.target.value })} placeholder="10.0.0.1" className="font-mono" error={checkIP(neighbor.address)} />
        <EntryRow title="Remote ASN" inputMode="numeric" value={neighbor.remote_asn || ''} onChange={(e) => set({ remote_asn: Number(e.target.value) || 0 })} placeholder="65001" className="font-mono" error={checkASN(String(neighbor.remote_asn || ''))} />
        <EntryRow title="Description" value={neighbor.description ?? ''} onChange={(e) => set({ description: e.target.value })} placeholder="Upstream transit" />
        <EntryRow title="Password" type="password" value={neighbor.password ?? ''} onChange={(e) => set({ password: e.target.value })} placeholder="BGP session password" className="font-mono" />
        <EntryRow title="Update source" value={neighbor.update_source ?? ''} onChange={(e) => set({ update_source: e.target.value })} placeholder="lo, 10.0.0.1…" className="font-mono" />
        <EntryRow title="EBGP multihop (0 = disabled)" value={neighbor.ebgp_multihop ?? ''} onChange={(e) => set({ ebgp_multihop: e.target.value ? Number(e.target.value) : undefined })} placeholder="TTL hops" className="font-mono" />
        <SwitchRow title="Disable connected check" checked={!!neighbor.disable_connected_check} onCheckedChange={(v) => set({ disable_connected_check: v })} />
        <SwitchRow title="Passive" subtitle="Do not initiate the connection" checked={!!neighbor.passive} onCheckedChange={(v) => set({ passive: v || undefined })} />
        <SwitchRow title="Shutdown (administratively down)" checked={!!neighbor.shutdown} onCheckedChange={(v) => set({ shutdown: v || undefined })} />
        <SwitchRow title="Enable BFD" checked={!!neighbor.bfd} onCheckedChange={(v) => set({ bfd: v, bfd_profile: v ? neighbor.bfd_profile : undefined })} />
        {neighbor.bfd && (
          <ComboRow title="BFD profile" subtitle={bfdProfileNames.length === 0 ? 'No profiles, define them in the BFD tab' : undefined}>
            <Select
              value={neighbor.bfd_profile || '_default'}
              onValueChange={(v) => set({ bfd_profile: v === '_default' ? undefined : v })}
            >
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="_default">Default timers</SelectItem>
                {bfdProfileNames.map((name) => (
                  <SelectItem key={name} value={name} className="font-mono">{name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </ComboRow>
        )}
      </PreferencesGroup>

      <PreferencesGroup title="Additional directives">
        <div className="px-4 py-3 space-y-2">
          <p className="text-xs text-muted-foreground">
            Raw FRR directives appended verbatim as <span className="font-mono">neighbor &lt;addr&gt; …</span> (e.g. <span className="font-mono">timers 10 30</span>, <span className="font-mono">local-as 65100</span>).
          </p>
          <TagInput values={neighbor.extra ?? []} onChange={(v) => set({ extra: v.length ? v : undefined })} placeholder="timers 10 30" mono />
        </div>
      </PreferencesGroup>

      <NeighborAFEditor
        afs={neighbor.address_families ?? {}}
        onChange={(v) => set({ address_families: v })}
        prefixListNames={prefixListNames}
        routeMapNames={routeMapNames}
        onDirty={onDirty}
      />

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add Neighbor</Button>}
      </div>
    </div>
  )
}

export function NeighborRow({
  neighbor, onEdit, onDelete,
}: {
  neighbor: BGPNeighbor
  onEdit: () => void
  onDelete: () => void
}) {
  return (
    <div onClick={onEdit} className="flex items-center gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors">
      <span className="font-mono font-semibold text-sm w-40 shrink-0 truncate">
        {neighbor.address || <span className="text-muted-foreground italic font-normal">no address</span>}
      </span>
      <div className="flex flex-wrap items-center gap-1.5 flex-1 min-w-0">
        <Badge variant="outline" className="text-xs font-mono">AS{neighbor.remote_asn || '?'}</Badge>
        {neighbor.bfd && (
          <Badge variant="secondary" className="text-xs">BFD{neighbor.bfd_profile ? `: ${neighbor.bfd_profile}` : ''}</Badge>
        )}
        {neighbor.description && (
          <span className="text-xs text-muted-foreground truncate">{neighbor.description}</span>
        )}
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={(e) => { e.stopPropagation(); onDelete() }}
        className="h-7 w-7 p-0 hover:text-destructive shrink-0"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

export function BGPNeighborsPanel({
  neighbors, onChange, prefixListNames, routeMapNames, bfdProfileNames, onDirty,
}: {
  neighbors: BGPNeighbor[]
  onChange: (v: BGPNeighbor[]) => void
  prefixListNames: string[]
  routeMapNames: string[]
  bfdProfileNames: string[]
  onDirty: () => void
}) {
  const [openIdx, setOpenIdx] = useState<number | null>(null)
  const [formDraft, setFormDraft] = useState<BGPNeighbor | null>(null)

  const update = (next: BGPNeighbor[]) => { onChange(next); onDirty() }

  const handleAddNew = () => { setFormDraft(emptyNeighbor()); setOpenIdx(-1) }
  const handleCommitNew = () => {
    if (!formDraft) return
    update([...neighbors, formDraft])
    setOpenIdx(null); setFormDraft(null)
  }
  const handleCloseSheet = () => { setOpenIdx(null); setFormDraft(null) }
  const remove = (idx: number) => {
    update(neighbors.filter((_, i) => i !== idx))
    if (openIdx === idx) handleCloseSheet()
  }

  const isAdding = openIdx === -1
  const openNeighbor = openIdx !== null && openIdx >= 0 ? neighbors[openIdx] : null
  const sheetTitle = isAdding ? 'Add Neighbor' : (openNeighbor?.address || 'Neighbor')

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label className="text-sm font-semibold">Neighbors</Label>
        <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
          <Plus className="h-3.5 w-3.5" />Add Neighbor
        </Button>
      </div>

      {neighbors.length === 0 ? (
        <EmptyState
          title="No neighbors"
          message="Add a BGP neighbor to exchange routes with a peer AS."
          action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add neighbor</Button>}
        />
      ) : (
        <PreferencesColumns>
          {neighbors.map((n, idx) => (
            <NeighborRow
              key={idx}
              neighbor={n}
              onEdit={() => setOpenIdx(idx)}
              onDelete={() => remove(idx)}
            />
          ))}
        </PreferencesColumns>
      )}

      <Sheet open={openIdx !== null} onClose={handleCloseSheet} title={sheetTitle} className="max-w-2xl">
        {openIdx !== null && (isAdding ? formDraft : openNeighbor) && (
          <NeighborForm
            neighbor={isAdding ? formDraft! : openNeighbor!}
            onChange={isAdding ? setFormDraft : (u) => update(neighbors.map((n, i) => i === openIdx ? u : n))}
            prefixListNames={prefixListNames}
            routeMapNames={routeMapNames}
            bfdProfileNames={bfdProfileNames}
            onAdd={isAdding ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
            onDirty={onDirty}
          />
        )}
      </Sheet>
    </div>
  )
}
