import { VarPickerInput } from '@/components/VarPickerInput'
import { ACTIONS, ADDR_FAMILIES, CT_STATES, ManagedRule, PROTOCOLS, RuleMatch } from '@/components/firewall/shared'
import { Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Separator, Switch } from 'cheval-ui'

type RuleBodyShape = { rule: ManagedRule; onChange: (r: ManagedRule) => void; vars: string[] }

export function RuleBody({ rule, onChange, vars }: RuleBodyShape) {
  const m = rule.match ?? {}

  const setMatch = (patch: Partial<RuleMatch>) =>
    onChange({ ...rule, match: { ...m, ...patch } })

  return (
    <>
          <div className="grid grid-cols-3 gap-2">
            <div className="space-y-1.5">
              <Label className="text-[11px]">Protocol</Label>
              <Select value={m.protocol ?? '_any'} onValueChange={(v) => setMatch({ protocol: v === '_any' ? undefined : v })}>
                <SelectTrigger className="h-8 text-xs font-mono">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="_any">any</SelectItem>
                  {PROTOCOLS.map((p) => <SelectItem key={p} value={p} className="font-mono">{p}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Input interface</Label>
              <VarPickerInput value={m.iif ?? ''} onChange={(v) => setMatch({ iif: v || undefined })}
                vars={vars.filter((v) => v.endsWith('_interfaces') || v === 'interfaces' || v === 'tunnels' || v === 'wireguard')}
                placeholder="$lan_interfaces" mono />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Output interface</Label>
              <VarPickerInput value={m.oif ?? ''} onChange={(v) => setMatch({ oif: v || undefined })}
                vars={vars.filter((v) => v.endsWith('_interfaces') || v === 'interfaces' || v === 'tunnels' || v === 'wireguard')}
                placeholder="$wan_interfaces" mono />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Source addr</Label>
              <VarPickerInput value={m.saddr ?? ''} onChange={(v) => setMatch({ saddr: v || undefined })}
                vars={vars.filter((v) => v.includes('_address') || v.includes('_network'))}
                placeholder="$lan_network" mono />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Dest addr</Label>
              <VarPickerInput value={m.daddr ?? ''} onChange={(v) => setMatch({ daddr: v || undefined })}
                vars={vars.filter((v) => v.includes('_address') || v.includes('_network'))}
                placeholder="192.168.0.0/24" mono />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">
                Addr family
                {(m.saddr || m.daddr) && <span className="text-destructive ml-0.5">*</span>}
              </Label>
              <Select value={m.addr_family ?? '_none'} onValueChange={(v) => setMatch({ addr_family: v === '_none' ? undefined : v })}>
                <SelectTrigger className={`h-8 text-xs ${(m.saddr || m.daddr) && !m.addr_family ? 'border-destructive focus:ring-destructive' : ''}`}>
                  <SelectValue placeholder="- required for saddr/daddr -" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="_none">- required for saddr/daddr -</SelectItem>
                  {ADDR_FAMILIES.map((f) => <SelectItem key={f} value={f}>{f}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Source port</Label>
              <Input value={m.sport ?? ''} onChange={(e) => setMatch({ sport: e.target.value || undefined })}
                placeholder="1024-65535" className="font-mono text-xs h-8" />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Dest port</Label>
              <Input value={m.dport ?? ''} onChange={(e) => setMatch({ dport: e.target.value || undefined })}
                placeholder="80,443" className="font-mono text-xs h-8" />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">CT state</Label>
              <Select value={m.ct_state ?? '_any'} onValueChange={(v) => setMatch({ ct_state: v === '_any' ? undefined : v })}>
                <SelectTrigger className="h-8 text-xs font-mono">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="_any">any</SelectItem>
                  {CT_STATES.map((s) => <SelectItem key={s} value={s} className="font-mono">{s}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
          </div>

          <Separator />

          <div className="flex flex-wrap items-end gap-3">
            <div className="space-y-1.5">
              <Label className="text-[11px]">Action</Label>
              <Select value={rule.action} onValueChange={(v) => onChange({ ...rule, action: v })}>
                <SelectTrigger className="h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ACTIONS.map((a) => <SelectItem key={a} value={a}>{a}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            {(rule.action === 'dnat' || rule.action === 'snat') && (
              <div className="space-y-1.5 flex-1 min-w-32">
                <Label className="text-[11px]">Target address</Label>
                <Input value={rule.action_to ?? ''}
                  onChange={(e) => onChange({ ...rule, action_to: e.target.value || undefined })}
                  placeholder="10.0.0.1" className="font-mono text-xs h-8" />
              </div>
            )}
            <div className="space-y-1.5 flex-1 min-w-32">
              <Label className="text-[11px]">Comment</Label>
              <Input value={rule.comment ?? ''}
                onChange={(e) => onChange({ ...rule, comment: e.target.value || undefined })}
                placeholder="optional description" className="text-xs h-8" />
            </div>
            <div className="flex items-center justify-between gap-2 mb-0.5">
              <span className="text-xs text-muted-foreground">Disabled</span>
              <Switch checked={rule.disabled ?? false}
                onCheckedChange={(v) => onChange({ ...rule, disabled: v || undefined })} />
            </div>
          </div>
    </>
  )
}
