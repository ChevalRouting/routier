import { NeighborAFEditor } from '@/components/routing/bgp/Neighbors'
import { BGPNeighbor } from '@/components/routing/types'
import { checkASN, checkIP } from '@/lib/validate'
import { Button, ComboRow, EntryRow, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, SwitchRow, TagInput } from 'cheval-ui'

type NeighborFormShape = {
  neighbor: BGPNeighbor
  onChange: (v: BGPNeighbor) => void
  prefixListNames: string[]
  routeMapNames: string[]
  bfdProfileNames: string[]
  onAdd?: () => void
  onDone: () => void
  onDirty: () => void
}

export function NeighborForm({
  neighbor, onChange, prefixListNames, routeMapNames, bfdProfileNames, onAdd, onDone, onDirty,
}: NeighborFormShape) {
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
