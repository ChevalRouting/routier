import { NameSelect } from '@/components/routing/shared'
import { BGPNeighborAF, STANDARD_AFS } from '@/components/routing/types'
import { Badge, Button, Card, ComboRow, EntryRow, Label, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, SwitchRow, TagInput } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'

type NeighborAFEditorShape = {
  afs: Record<string, BGPNeighborAF>
  onChange: (v: Record<string, BGPNeighborAF>) => void
  prefixListNames: string[]
  routeMapNames: string[]
  onDirty: () => void
}

export function NeighborAFEditor({
  afs, onChange, prefixListNames, routeMapNames, onDirty,
}: NeighborAFEditorShape) {
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
                <PreferencesGroup>
                  <SwitchRow title="Activate in this address family" checked={!af.disabled} onCheckedChange={(v) => upd(afName, { disabled: !v })} />
                  <SwitchRow title="Soft reconfiguration inbound" checked={!!af.soft_reconfiguration} onCheckedChange={(v) => upd(afName, { soft_reconfiguration: v })} />
                  <SwitchRow title="Advertise default route (default-originate)" checked={!!af.default_originate} onCheckedChange={(v) => upd(afName, { default_originate: v })} />
                  <SwitchRow title="Next-hop-self" checked={!!af.next_hop_self} onCheckedChange={(v) => upd(afName, { next_hop_self: v })} />
                  <SwitchRow title="Route-reflector client" checked={!!af.route_reflector_client} onCheckedChange={(v) => upd(afName, { route_reflector_client: v })} />
                  <SwitchRow title="Remove private ASNs (remove-private-AS)" checked={!!af.remove_private_as} onCheckedChange={(v) => upd(afName, { remove_private_as: v })} />
                  <EntryRow title="allowas-in (0 = off)" value={af.allowas_in ?? ''} onChange={(e) => upd(afName, { allowas_in: Number(e.target.value) || undefined })} placeholder="0" className="font-mono" />
                  <EntryRow title="Weight (0 = default)" value={af.weight ?? ''} onChange={(e) => upd(afName, { weight: Number(e.target.value) || undefined })} placeholder="0" className="font-mono" />
                </PreferencesGroup>
                <PreferencesGroup title="Route filters" description="Apply filters to this neighbor for this address family.">
                  {([
                    ['prefix_list_in', 'Prefix list in', prefixListNames],
                    ['prefix_list_out', 'Prefix list out', prefixListNames],
                    ['route_map_in', 'Inbound route map', routeMapNames],
                    ['route_map_out', 'Outbound route map', routeMapNames],
                  ] as const).map(([field, label, names]) => (
                    <ComboRow key={field} title={label}>
                      <NameSelect value={af[field]} onChange={(v) => upd(afName, { [field]: v })} names={names as string[]} />
                    </ComboRow>
                  ))}
                </PreferencesGroup>
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
