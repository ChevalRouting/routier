import type { TypesSystemNic as SystemNic } from '@/api'
import { Field, joinPool, selectorFor, splitPool } from '@/components/simple/NetworkEditor'
import type { Network } from '@/components/simple/types'
import {
  Input,
  Select, SelectContent, SelectItem, SelectTrigger,
  SelectValue, SwitchRow, TagInput
} from 'cheval-ui'

type NetworkFieldsShape = {
  network: Network
  nics: SystemNic[]
  onChange: (patch: Partial<Network>) => void
}

export function NetworkFields({ network, nics, onChange }: NetworkFieldsShape) {
  const disabled = !network.editable
  const [poolStart, poolEnd] = splitPool(network.pool)

  return (
    <div className="space-y-5">
      {network.issue && <p className="text-sm text-warning">{network.issue}</p>}
      <div className="grid gap-4 md:grid-cols-2">
        <Field label="Network name">
          <Input value={network.name} disabled={disabled} placeholder="office" onChange={(event) => onChange({ name: event.target.value })} />
        </Field>
        <Field label="Physical port">
          <Select value={network.select} disabled={disabled} onValueChange={(select) => onChange({ select })}>
            <SelectTrigger><SelectValue placeholder="Select a port" /></SelectTrigger>
            <SelectContent>
              {!nics.some((nic) => selectorFor(nic) === network.select) && network.select && (
                <SelectItem value={network.select}>{network.select}</SelectItem>
              )}
              {nics.filter((nic) => nic.physical !== false).map((nic) => (
                <SelectItem key={nic.name} value={selectorFor(nic)}>
                  {nic.name}{nic.carrier ? ' (connected)' : ''}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        <Field label="Router addresses" description="IPv4 and IPv6 addresses with prefix used by Routier on this network.">
          {disabled
            ? <Input className="font-mono" value={(network.addresses ?? []).join(', ')} disabled />
            : <TagInput values={network.addresses ?? []} onChange={(addresses) => onChange({ addresses })} placeholder="192.168.1.1/24" mono />}
        </Field>
      </div>

      <SwitchRow
        title="Manage connected devices automatically"
        subtitle="Run DHCP on this network and assign addresses to clients."
        checked={network.manage_dhcp}
        disabled={disabled}
        onCheckedChange={(manage_dhcp) => onChange({ manage_dhcp })}
      />

      <SwitchRow
        title="Manage IPv6 devices automatically"
        subtitle="Run DHCPv6 and advertise the selected IPv6 prefix with router advertisements."
        checked={!!network.manage_dhcp6}
        disabled={disabled}
        onCheckedChange={(manage_dhcp6) => onChange({ manage_dhcp6 })}
      />

      <SwitchRow
        title="Allow internet access"
        subtitle="Let devices on this network reach the internet through the router (masquerade)."
        checked={network.masquerade}
        disabled={disabled}
        onCheckedChange={(masquerade) => onChange({ masquerade })}
      />

      {network.manage_dhcp && (
        <div className="space-y-4 border-t border-border/60 pt-4">
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="First automatic address">
              <Input className="font-mono" value={poolStart} disabled={disabled} placeholder="192.168.1.100" onChange={(event) => onChange({ pool: joinPool(event.target.value, poolEnd) })} />
            </Field>
            <Field label="Last automatic address">
              <Input className="font-mono" value={poolEnd} disabled={disabled} placeholder="192.168.1.250" onChange={(event) => onChange({ pool: joinPool(poolStart, event.target.value) })} />
            </Field>
            <Field label="Gateway" description="Defaults to the router address when left empty.">
              <Input className="font-mono" value={network.gateway ?? ''} disabled={disabled} placeholder="192.168.1.1" onChange={(event) => onChange({ gateway: event.target.value || undefined })} />
            </Field>
            <Field label="Lease time (seconds)">
              <Input type="number" min={60} step={60} value={network.valid_lifetime ?? 3600} disabled={disabled} onChange={(event) => onChange({ valid_lifetime: Number(event.target.value) })} />
            </Field>
          </div>
          <Field label="DNS servers" description="Clients use these resolvers. Add one address at a time.">
            {disabled
              ? <Input className="font-mono" value={(network.dns ?? []).join(', ')} disabled />
              : <TagInput values={network.dns ?? []} onChange={(dns) => onChange({ dns })} placeholder="1.1.1.1" mono />}
          </Field>
          <Field label="Excluded addresses" description="Addresses or ranges DHCP must never assign automatically.">
            {disabled
              ? <Input className="font-mono" value={(network.exclusions ?? []).join(', ')} disabled />
              : <TagInput values={network.exclusions ?? []} onChange={(exclusions) => onChange({ exclusions })} placeholder="192.168.1.2-192.168.1.20" mono />}
          </Field>
        </div>
      )}

      {network.manage_dhcp6 && (
        <div className="space-y-4 border-t border-border/60 pt-4">
          <Field label="DHCPv6 address pool" description="An IPv6 range inside the first IPv6 prefix configured above.">
            <Input className="font-mono" value={network.pool_v6 ?? ''} disabled={disabled} placeholder="fd00:1::100-fd00:1::ffff" onChange={(event) => onChange({ pool_v6: event.target.value || undefined })} />
          </Field>
          <Field label="DHCPv6 lease time (seconds)">
            <Input type="number" min={60} step={60} value={network.valid_lifetime_v6 ?? 3600} disabled={disabled} onChange={(event) => onChange({ valid_lifetime_v6: Number(event.target.value) })} />
          </Field>
        </div>
      )}
    </div>
  )
}
