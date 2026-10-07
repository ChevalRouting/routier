import type { NatSpec } from '@/api'
import { VarPickerInput } from '@/components/VarPickerInput'
import { Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'

type NatBodyShape = {
  spec: NatSpec
  ifaceVars: string[]
  addrVars: string[]
  onChange: (patch: Partial<NatSpec>) => void
}

export function NatBody({ spec, ifaceVars, addrVars, onChange }: NatBodyShape) {
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
      {(spec.kind === 'masquerade' || spec.kind === 'snat') && (
        <div className="space-y-1.5">
          <Label className="text-[11px]">Out interface</Label>
          <VarPickerInput value={spec.out ?? ''} onChange={(v) => onChange({ out: v })} vars={ifaceVars} placeholder="$wan_interfaces" mono />
        </div>
      )}
      {spec.kind === 'snat' && (
        <div className="space-y-1.5">
          <Label className="text-[11px]">SNAT to</Label>
          <Input value={spec.to ?? ''} onChange={(e) => onChange({ to: e.target.value })} placeholder="203.0.113.5" className="font-mono text-sm h-8" />
        </div>
      )}
      {(spec.kind === 'masquerade' || spec.kind === 'snat') && (
        <>
          <div className="space-y-1.5">
            <Label className="text-[11px]">Source (optional)</Label>
            <VarPickerInput value={spec.source ?? ''} onChange={(v) => onChange({ source: v })} vars={addrVars} placeholder="$lan_network" mono />
          </div>
          {spec.source ? (
            <div className="space-y-1.5">
              <Label className="text-[11px]">Source family</Label>
              <Select value={spec.family || 'ip'} onValueChange={(v) => onChange({ family: v })}>
                <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="ip">ip</SelectItem>
                  <SelectItem value="ip6">ip6</SelectItem>
                </SelectContent>
              </Select>
            </div>
          ) : null}
        </>
      )}

      {spec.kind === 'dnat' && (
        <>
          <div className="space-y-1.5">
            <Label className="text-[11px]">In interface</Label>
            <VarPickerInput value={spec.in ?? ''} onChange={(v) => onChange({ in: v })} vars={ifaceVars} placeholder="$wan_interfaces" mono />
          </div>
          <div className="space-y-1.5">
            <Label className="text-[11px]">Protocol</Label>
            <Select value={spec.proto || 'tcp'} onValueChange={(v) => onChange({ proto: v })}>
              <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="tcp">tcp</SelectItem>
                <SelectItem value="udp">udp</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label className="text-[11px]">Dest port</Label>
            <Input value={spec.dport ?? ''} onChange={(e) => onChange({ dport: e.target.value })} placeholder="443" className="font-mono text-sm h-8" />
          </div>
          <div className="space-y-1.5">
            <Label className="text-[11px]">Forward to</Label>
            <Input value={spec.to ?? ''} onChange={(e) => onChange({ to: e.target.value })} placeholder="10.0.0.5:8443" className="font-mono text-sm h-8" />
          </div>
        </>
      )}

      <div className="space-y-1.5">
        <Label className="text-[11px]">Comment</Label>
        <Input value={spec.comment ?? ''} onChange={(e) => onChange({ comment: e.target.value })} placeholder="optional" className="text-sm h-8" />
      </div>
    </div>
  )
}
