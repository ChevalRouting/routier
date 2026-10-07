import { HwField, IfaceCardProps } from '@/components/interfaces/shared'
import { Badge, Button } from 'cheval-ui'
import {
  Trash2
} from 'lucide-react'

export function IfaceCard({ name, iface, nic, hasVrrp, onEdit, onDelete }: IfaceCardProps) {
  const isVlan = iface.type === 'vlan'
  const addresses = iface.addresses ?? []
  const staticAddrs = addresses.filter((a) => a.includes('/'))
  const modeAddrs = addresses.filter((a) => !a.includes('/'))
  const dynamicAddrs = (nic?.addrs ?? []).filter((a) => !staticAddrs.includes(a) && !a.toLowerCase().startsWith('fe80'))
  const hasModePills = modeAddrs.length > 0 || isVlan || !!iface.bridge || !!iface.bond || hasVrrp
  const up = nic ? nic.operstate === 'up' || !!nic.carrier : undefined
  const model = nic?.pci_vendor && nic?.pci_device ? `${nic.pci_vendor}:${nic.pci_device}` : ''
  const speed = nic?.speed ? `${nic.speed} Mbps${nic.duplex ? ` ${nic.duplex}` : ''}` : ''

  return (
    <div onClick={onEdit} className="flex cursor-pointer flex-col gap-3 rounded-lg bg-card p-4 shadow-[var(--card-shadow)] transition-colors hover:bg-accent/40">
      <div className="flex items-center gap-2">
        {up !== undefined && (
          <span className={`h-2 w-2 shrink-0 rounded-full ${up ? 'bg-success' : 'bg-muted-foreground'}`} title={up ? 'link up' : 'link down'} />
        )}
        <span className="truncate font-mono text-sm font-semibold">{name}</span>
        {nic && nic.name !== name && (
          <span className="shrink-0 font-mono text-xs text-muted-foreground" title="kernel device">→ {nic.name}</span>
        )}
        <div className="ml-auto flex shrink-0 items-center gap-1">
          <Badge variant="secondary" className="font-mono text-xs">{iface.type || 'physical'}</Badge>
          {iface.vrf && <Badge variant="secondary" className="font-mono text-xs">vrf:{iface.vrf}</Badge>}
          <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive"
            onClick={(e) => { e.stopPropagation(); onDelete() }}>
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <div className="flex flex-wrap gap-1">
          {staticAddrs.length === 0 && dynamicAddrs.length === 0
            ? <span className="text-xs text-muted-foreground">no addresses</span>
            : (<>
                {staticAddrs.map((addr) => <Badge key={addr} variant="outline" className="font-mono text-xs">{addr}</Badge>)}
                {dynamicAddrs.map((addr) => <Badge key={addr} variant="info" className="font-mono text-xs" title="obtained via DHCP/SLAAC">{addr}</Badge>)}
              </>)}
          {iface.type === 'tunnel' && (iface.local || iface.remote) && (
            <Badge variant="outline" className="font-mono text-xs">{iface.local || '-'} → {iface.remote || '-'}</Badge>
          )}
        </div>

        {hasModePills && (
          <div className="flex flex-wrap gap-1">
            {modeAddrs.map((m) => <Badge key={m} variant="secondary" className="font-mono text-xs">{m}</Badge>)}
            {isVlan && <Badge variant="outline" className="text-xs">vlan {iface.vlan?.id ?? 0}{iface.select ? ` · ${iface.select}` : ''}</Badge>}
            {iface.bridge && <Badge variant="outline" className="text-xs">bridge</Badge>}
            {iface.bond && <Badge variant="outline" className="text-xs">{iface.bond.mode || 'balance-rr'} · {iface.bond.members?.length ?? 0}</Badge>}
            {hasVrrp && <Badge variant="outline" className="text-xs">VRRP</Badge>}
          </div>
        )}
      </div>

      {nic?.physical && (
        <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 rounded-md bg-muted/30 px-3 py-2 text-xs sm:grid-cols-3">
          <HwField label="Driver" value={nic.driver || '-'} />
          <HwField label="Speed" value={speed || '-'} />
          <HwField label="Model" value={model || '-'} mono />
          <HwField label="MAC" value={nic.mac || '-'} mono />
          <HwField label="Link" value={nic.carrier ? 'up' : 'down'} />
        </div>
      )}
    </div>
  )
}
