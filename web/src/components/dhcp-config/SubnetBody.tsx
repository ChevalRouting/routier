import { VarPickerInput } from '@/components/VarPickerInput'
import { InterfaceSelect, KeaSubnet, PoolRows } from '@/components/dhcp-config/shared'
import { Input, Label, TagInput } from 'cheval-ui'

type SubnetBodyShape = {
  v6: boolean
  subnet: KeaSubnet
  networks: string[]
  ifaceNames: string[]
  onChange: (s: KeaSubnet) => void
}

export function SubnetBody({ v6, subnet, networks, ifaceNames, onChange }: SubnetBodyShape) {
  const set = <K extends keyof KeaSubnet>(k: K, val: KeaSubnet[K]) => onChange({ ...subnet, [k]: val })

  return (
    <>
      <div className="space-y-2">
        <div className="space-y-1.5">
          <Label className="text-[11px] text-muted-foreground">Network (CIDR)</Label>
          <VarPickerInput
            value={subnet.subnet}
            onChange={(v) => set('subnet', v)}
            vars={networks}
            prefix=""
            label="Configured networks"
            placeholder={v6 ? '2001:db8::/64' : '10.0.0.0/24'}
            mono
            wrapperClassName="flex-1"
            className="h-8 text-xs"
          />
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-1.5">
            <Label className="text-[11px] text-muted-foreground">Interface</Label>
            <InterfaceSelect value={subnet.interface} options={ifaceNames} onChange={(v) => set('interface', v)} />
          </div>
          {!v6 && (
            <div className="space-y-1.5">
              <Label className="text-[11px] text-muted-foreground">Gateway</Label>
              <Input value={subnet.gateway ?? ''} onChange={(e) => set('gateway', e.target.value || undefined)} placeholder="10.0.0.1" className="h-8 text-xs font-mono" />
            </div>
          )}
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <PoolRows v6={v6} pools={subnet.pools ?? []} onChange={(v) => set('pools', v.length ? v : undefined)} />
        <div className="space-y-1.5">
          <Label className="text-[11px] text-muted-foreground">DNS servers</Label>
          <TagInput values={subnet.dns ?? []} onChange={(v) => set('dns', v.length ? v : undefined)} placeholder={v6 ? '2001:db8::53' : '1.1.1.1'} mono />
        </div>
        <div className="space-y-1.5 md:col-span-2">
          <Label className="text-[11px] text-muted-foreground">Reserved for manual use (excluded from auto-suggested addresses)</Label>
          <TagInput values={subnet.exclusions ?? []} onChange={(v) => set('exclusions', v.length ? v : undefined)} placeholder={v6 ? '2001:db8::1-2001:db8::ff' : '10.0.0.1-10.0.0.20 or 10.0.0.5'} mono />
        </div>
      </div>
    </>
  )
}
