import type { TypesFriendInterface as FriendInterface } from '@/api'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'

type AddressSelectShape = {
  value: string
  onChange: (v: string) => void
  ifs: FriendInterface[] | null
  placeholder: string
}

export const TUNNEL_MODES = ['gre', 'ipip', 'sit', 'ip6tnl', 'ip6ip6', 'ip6gre']

export function addrOptions(ifs: FriendInterface[] | null) {
  return (ifs ?? []).flatMap((i) => (i.addresses ?? []).map((a) => ({ iface: i.name, addr: a })))
}

export function AddressSelect({ value, onChange, ifs, placeholder }: AddressSelectShape) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger className="h-9"><SelectValue placeholder={placeholder} /></SelectTrigger>
      <SelectContent>
        {addrOptions(ifs).map((o) => (
          <SelectItem key={`${o.iface}|${o.addr}`} value={`${o.iface}|${o.addr}`} className="font-mono">{o.addr} ({o.iface})</SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export { TunnelDialog } from './dialogsParts/TunnelDialog'
export { WireguardDialog } from './dialogsParts/WireguardDialog'
