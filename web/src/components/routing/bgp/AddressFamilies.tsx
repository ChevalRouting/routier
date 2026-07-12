import { useState } from 'react'
import { checkCIDR } from '@/lib/validate'
import { NumberInput } from '@/components/ui/number-input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import TagInput from '@/components/TagInput'
import { Plus, Trash2 } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { BGPAddressFamily, STANDARD_AFS } from '../types'
import { NameSelect } from '../shared'

export function BGPAddressFamiliesPanel({
  afs, onChange, routeMapNames, vrfNames, onDirty,
}: {
  afs: Record<string, BGPAddressFamily>
  onChange: (v: Record<string, BGPAddressFamily>) => void
  routeMapNames: string[]
  vrfNames: string[]
  onDirty: () => void
}) {
  const [adding, setAdding] = useState('')
  const configured = Object.keys(afs)
  const available = STANDARD_AFS.filter((af) => !configured.includes(af))

  const upd = (name: string, patch: Partial<BGPAddressFamily>) => {
    onChange({ ...afs, [name]: { ...afs[name], ...patch } }); onDirty()
  }
  const remove = (name: string) => {
    const next = { ...afs }; delete next[name]; onChange(next); onDirty()
  }
  const add = () => {
    if (!adding || adding in afs) return
    onChange({ ...afs, [adding]: { networks: [], redistribute: [] } }); setAdding(''); onDirty()
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Label className="text-sm font-semibold">Address families</Label>
        {available.length > 0 && (
          <div className="flex items-center gap-1.5">
            <Select value={adding} onValueChange={setAdding}>
              <SelectTrigger className="h-8 text-xs w-44">
                <SelectValue placeholder="Select AF…" />
              </SelectTrigger>
              <SelectContent>
                {available.map((af) => (
                  <SelectItem key={af} value={af}>{af}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button variant="outline" size="sm" className="h-8 gap-1.5" onClick={add} disabled={!adding}>
              <Plus className="h-3.5 w-3.5" />Add
            </Button>
          </div>
        )}
      </div>

      {configured.length === 0 ? (
        <p className="text-sm text-muted-foreground italic">No address families configured. Add one above.</p>
      ) : (
        <div className="space-y-3">
          {configured.map((afName) => {
            const af = afs[afName] ?? {}
            return (
              <div key={afName} className="border rounded-md p-4 space-y-4">
                <div className="flex items-center justify-between">
                  <Badge className="font-mono">{afName}</Badge>
                  <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => remove(afName)}>
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div className="space-y-1.5">
                    <Label className="text-xs text-muted-foreground">Networks to advertise</Label>
                    <TagInput
                      values={af.networks ?? []}
                      onChange={(v) => upd(afName, { networks: v })}
                      placeholder="10.0.0.0/24"
                      mono
                      validate={checkCIDR}
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label className="text-xs text-muted-foreground">Redistribute</Label>
                    <TagInput
                      values={af.redistribute ?? []}
                      onChange={(v) => upd(afName, { redistribute: v })}
                      placeholder="connected, static, ospf…"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-1.5">
                    <Label className="text-xs text-muted-foreground">Route map in</Label>
                    <NameSelect value={af.route_map_in} onChange={(v) => upd(afName, { route_map_in: v })} names={routeMapNames} />
                  </div>
                  <div className="space-y-1.5">
                    <Label className="text-xs text-muted-foreground">Route map out</Label>
                    <NameSelect value={af.route_map_out} onChange={(v) => upd(afName, { route_map_out: v })} names={routeMapNames} />
                  </div>
                  <div className="space-y-1.5">
                    <Label className="text-xs text-muted-foreground">Maximum paths</Label>
                    <NumberInput
                      value={af.maximum_paths || undefined}
                      onChange={(v) => upd(afName, { maximum_paths: v })}
                      placeholder="1"
                      className="font-mono h-8 text-sm"
                    />
                  </div>
                  <div className="flex items-center gap-2 pt-5">
                    <Switch checked={!!af.default_originate} onCheckedChange={(v) => upd(afName, { default_originate: v })} />
                    <span className="text-sm">Default originate</span>
                  </div>
                </div>

                <div className="space-y-1.5">
                  <Label className="text-xs text-muted-foreground">Import VRF</Label>
                  <TagInput
                    values={af.import_vrf ?? []}
                    onChange={(v) => upd(afName, { import_vrf: v })}
                    placeholder={vrfNames.length > 0 ? vrfNames.join(', ') + '…' : 'vrf name…'}
                  />
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

