import { EntityRow } from '@/components/ui/EntityRow'
import { plural } from '@/lib/utils'
import { Badge } from 'cheval-ui'
import { WgIface } from './shared'

export interface WgRowProps {
  name: string
  iface: WgIface
  onEdit: () => void
  onDelete: () => void
}

export function WgRow({ name, iface, onEdit, onDelete }: WgRowProps) {
  return (
    <EntityRow title={name} onOpen={onEdit} onRemove={onDelete} removeLabel="Remove Interface">
      <Badge variant="secondary" className="font-mono">:{iface.listen_port || 51820}</Badge>
      {iface.friend && <Badge variant="outline">derived · {iface.friend}</Badge>}
      {(iface.addresses ?? []).map((addr) => (
        <Badge key={addr} variant="outline" className="max-w-full whitespace-normal break-all font-mono">{addr}</Badge>
      ))}
      {(iface.peers ?? []).length > 0 && <Badge variant="outline">{plural((iface.peers ?? []).length, 'peer')}</Badge>}
    </EntityRow>
  )
}
