import { Fragment, useState } from 'react'
import { checkCIDR } from '@/lib/validate'
import { NumberInput } from 'cheval-ui'
import { Input } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { Switch } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { TagInput } from 'cheval-ui'
import { Card } from 'cheval-ui'
import { Sheet } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
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
  const [openName, setOpenName] = useState<string | null>(null)
  const configured = Object.keys(afs)
  const available = STANDARD_AFS.filter((af) => !configured.includes(af))

  const upd = (name: string, patch: Partial<BGPAddressFamily>) => {
    onChange({ ...afs, [name]: { ...afs[name], ...patch } }); onDirty()
  }
  const remove = (name: string) => {
    const next = { ...afs }; delete next[name]; onChange(next); onDirty()
    if (openName === name) setOpenName(null)
  }
  const add = () => {
    if (!adding || adding in afs) return
    onChange({ ...afs, [adding]: { networks: [], redistribute: [] } }); setOpenName(adding); setAdding(''); onDirty()
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
            const isEVPN = afName === 'l2vpn-evpn'
            const isUnicast = afName === 'ipv4-unicast' || afName === 'ipv6-unicast'
            return (
              <Fragment key={afName}>
                <Card
                  className="flex cursor-pointer items-center gap-3 p-4 transition-colors hover:bg-accent/50"
                  onClick={() => setOpenName(afName)}
                >
                  <Badge className="font-mono">{afName}</Badge>
                  <div className="flex flex-1 flex-wrap gap-1.5">
                    <Badge variant="outline">{af.networks?.length ?? 0} networks</Badge>
                    <Badge variant="outline">{af.redistribute?.length ?? 0} redistributed</Badge>
                    {af.default_originate && <Badge variant="secondary">default originate</Badge>}
                  </div>
                  <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={(event) => { event.stopPropagation(); remove(afName) }}>
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </Card>

                <Sheet open={openName === afName} onClose={() => setOpenName(null)} title={afName} className="max-w-2xl">
                  <div className="space-y-4">

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

                {isUnicast && (
                  <div className="rounded-lg bg-muted/30 p-4 space-y-4">
                    <Label className="text-xs font-semibold">VPN route leaking</Label>
                    <p className="text-xs text-muted-foreground">
                      Leak routes between VRFs through the global VPN table. Set a route distinguisher and
                      matching route targets, then enable import/export in each participating VRF.
                    </p>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div className="space-y-1.5">
                        <Label className="text-xs text-muted-foreground">Route distinguisher (RD)</Label>
                        <Input
                          value={af.rd ?? ''}
                          onChange={(e) => upd(afName, { rd: e.target.value || undefined })}
                          placeholder="65000:1"
                          className="font-mono h-8 text-sm"
                        />
                      </div>
                      <div className="hidden sm:block" />
                      <div className="space-y-1.5">
                        <Label className="text-xs text-muted-foreground">Import route targets (RT)</Label>
                        <TagInput values={af.rt_vpn_import ?? []} onChange={(v) => upd(afName, { rt_vpn_import: v.length ? v : undefined })} placeholder="65000:1" mono />
                      </div>
                      <div className="space-y-1.5">
                        <Label className="text-xs text-muted-foreground">Export route targets (RT)</Label>
                        <TagInput values={af.rt_vpn_export ?? []} onChange={(v) => upd(afName, { rt_vpn_export: v.length ? v : undefined })} placeholder="65000:1" mono />
                      </div>
                    </div>
                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                      <div className="flex items-center gap-2">
                        <Switch checked={!!af.import_vpn} onCheckedChange={(v) => upd(afName, { import_vpn: v || undefined })} />
                        <span className="text-sm">Import VPN</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <Switch checked={!!af.export_vpn} onCheckedChange={(v) => upd(afName, { export_vpn: v || undefined })} />
                        <span className="text-sm">Export VPN</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <Switch checked={!!af.label_vpn_export_auto} onCheckedChange={(v) => upd(afName, { label_vpn_export_auto: v || undefined })} />
                        <span className="text-sm">Auto label</span>
                      </div>
                    </div>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div className="space-y-1.5">
                        <Label className="text-xs text-muted-foreground">Import route map</Label>
                        <NameSelect value={af.route_map_vpn_import} onChange={(v) => upd(afName, { route_map_vpn_import: v })} names={routeMapNames} />
                      </div>
                      <div className="space-y-1.5">
                        <Label className="text-xs text-muted-foreground">Export route map</Label>
                        <NameSelect value={af.route_map_vpn_export} onChange={(v) => upd(afName, { route_map_vpn_export: v })} names={routeMapNames} />
                      </div>
                    </div>
                  </div>
                )}

                {isEVPN && (
                  <div className="rounded-lg bg-muted/30 p-4 space-y-4">
                    <Label className="text-xs font-semibold">EVPN advertisements</Label>
                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                      <div className="flex items-center gap-2">
                        <Switch checked={!!af.advertise_all_vni} onCheckedChange={(v) => upd(afName, { advertise_all_vni: v })} />
                        <span className="text-sm">All VNIs</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <Switch checked={!!af.advertise_default_gateway} onCheckedChange={(v) => upd(afName, { advertise_default_gateway: v })} />
                        <span className="text-sm">Default gateway</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <Switch checked={!!af.advertise_svi_ip} onCheckedChange={(v) => upd(afName, { advertise_svi_ip: v })} />
                        <span className="text-sm">SVI IP</span>
                      </div>
                    </div>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div className="space-y-1.5">
                        <Label className="text-xs text-muted-foreground">Advertise address families</Label>
                        <TagInput values={af.advertise ?? []} onChange={(v) => upd(afName, { advertise: v })} suggestions={['ipv4-unicast', 'ipv6-unicast']} validate={(v) => ['ipv4-unicast', 'ipv6-unicast'].includes(v) ? null : 'Use ipv4-unicast or ipv6-unicast'} mono />
                      </div>
                      <div className="space-y-1.5">
                        <Label className="text-xs text-muted-foreground">Import route targets</Label>
                        <TagInput values={af.route_target_import ?? []} onChange={(v) => upd(afName, { route_target_import: v })} placeholder="65001:100" mono />
                      </div>
                      <div className="space-y-1.5 sm:col-start-2">
                        <Label className="text-xs text-muted-foreground">Export route targets</Label>
                        <TagInput values={af.route_target_export ?? []} onChange={(v) => upd(afName, { route_target_export: v })} placeholder="65001:100" mono />
                      </div>
                    </div>
                  </div>
                )}
                  </div>
                </Sheet>
              </Fragment>
            )
          })}
        </div>
      )}
    </div>
  )
}
