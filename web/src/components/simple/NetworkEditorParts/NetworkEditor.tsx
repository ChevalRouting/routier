import { NetworkEditorProps, NetworkFields, newNetwork } from '@/components/simple/NetworkEditor'
import type { Network } from '@/components/simple/types'
import { AccordionList } from '@/components/ui/AccordionList'
import {
  Badge, Button
} from 'cheval-ui'
import { Trash2 } from 'lucide-react'

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
