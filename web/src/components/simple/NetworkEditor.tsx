import { Trash2 } from 'lucide-react'
import {
  Badge, Button, Input, Label, Select, SelectContent, SelectItem, SelectTrigger,
  SelectValue, SwitchRow, TagInput,
} from 'cheval-ui'
import type { TypesSystemNic as SystemNic } from '@/api'
import type { Network } from '@/components/simple/types'
import { AccordionList } from '@/components/ui/AccordionList'

interface NetworkEditorProps {
  networks: Network[]
  nics: SystemNic[]
  onChange: (networks: Network[]) => void
}

export function NetworkEditor({ networks, nics, onChange }: NetworkEditorProps) {
  const update = (network: Network, patch: Partial<Network>) => {
    onChange(networks.map((current) => current === network ? { ...current, ...patch } : current))
  }

  const add = () => onChange([...networks, newNetwork(networks)])
  const remove = (network: Network) => onChange(networks.filter((current) => current !== network))

  return (
    <AccordionList
      items={networks}
      getId={(network, index) => `${network.id || 'new'}-${index}`}
      description={`${networks.length} local ${networks.length === 1 ? 'network' : 'networks'}`}
      addLabel="Add network"
      onAdd={add}
      emptyTitle="No local networks"
      emptyMessage="Add a network for devices connected to Routier."
      renderSummary={(network) => (
        <div className="flex items-center gap-2">
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-medium">{network.name || 'Unnamed network'}</div>
            <div className="truncate font-mono text-xs text-muted-foreground">{(network.addresses ?? []).join(', ') || 'No address'}</div>
          </div>
          {network.manage_dhcp && <Badge variant="secondary">Managed DHCP</Badge>}
          {network.manage_dhcp6 && <Badge variant="secondary">DHCPv6 + RA</Badge>}
          {!network.editable && <Badge variant="outline">Advanced</Badge>}
        </div>
      )}
      renderActions={(network) => network.editable && (
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive"
          title={`Delete ${network.name}`}
          onClick={() => remove(network)}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      )}
      renderBody={(network) => (
        <NetworkFields network={network} nics={nics} onChange={(patch) => update(network, patch)} />
      )}
    />
  )
}

function NetworkFields({ network, nics, onChange }: {
  network: Network
  nics: SystemNic[]
  onChange: (patch: Partial<Network>) => void
}) {
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

function Field({ label, description, children }: { label: string; description?: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {description && <p className="text-xs text-muted-foreground">{description}</p>}
      {children}
    </div>
  )
}

function selectorFor(nic: SystemNic): string {
  return nic.mac ? `mac(${nic.mac})` : `name=${nic.name}`
}

function splitPool(pool?: string): [string, string] {
  if (!pool) return ['', '']
  const separator = pool.indexOf('-')
  return separator < 0 ? [pool, ''] : [pool.slice(0, separator), pool.slice(separator + 1)]
}

function joinPool(start: string, end: string): string {
  return start && end ? `${start}-${end}` : start || end
}

function newNetwork(networks: Network[]): Network {
  const used = new Set(networks.map((network) => network.name))
  let suffix = 1
  while (used.has(`network-${suffix}`)) suffix++
  const octet = Math.min(suffix, 254)
  const router = `192.168.${octet}.1`
  return {
    id: '',
    name: `network-${suffix}`,
    select: '',
    addresses: [`${router}/24`],
    manage_dhcp: true,
    manage_dhcp6: false,
    pool: `192.168.${octet}.100-192.168.${octet}.250`,
    gateway: router,
    dns: [router],
    exclusions: [],
    valid_lifetime: 3600,
    masquerade: true,
    editable: true,
  }
}
