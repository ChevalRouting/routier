import type { TypesSystemNic as SystemNic } from '@/api'
import type { Network } from '@/components/simple/types'
import {
  Label
} from 'cheval-ui'

type FieldShape = { label: string; description?: string; children: React.ReactNode }

export interface NetworkEditorProps {
  networks: Network[]
  nics: SystemNic[]
  onChange: (networks: Network[]) => void
}

export function Field({ label, description, children }: FieldShape) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {description && <p className="text-xs text-muted-foreground">{description}</p>}
      {children}
    </div>
  )
}

export function selectorFor(nic: SystemNic): string {
  return nic.mac ? `mac(${nic.mac})` : `name=${nic.name}`
}

export function splitPool(pool?: string): [string, string] {
  if (!pool) return ['', '']
  const separator = pool.indexOf('-')
  return separator < 0 ? [pool, ''] : [pool.slice(0, separator), pool.slice(separator + 1)]
}

export function joinPool(start: string, end: string): string {
  return start && end ? `${start}-${end}` : start || end
}

export function newNetwork(networks: Network[]): Network {
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

export { NetworkEditor } from './NetworkEditorParts/NetworkEditor'
export { NetworkFields } from './NetworkEditorParts/NetworkFields'
