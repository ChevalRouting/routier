import { EntityRow } from '@/components/ui/EntityRow'
import { Badge } from 'cheval-ui'
import { BGPNeighbor } from '../types'

type NeighborRowShape = {
  neighbor: BGPNeighbor
  onEdit: () => void
  onDelete: () => void
}

export function emptyNeighbor(): BGPNeighbor {
  return { address: '', remote_asn: 0, address_families: {} }
}

export function NeighborRow({
  neighbor, onEdit, onDelete,
}: NeighborRowShape) {
  return (
    <EntityRow title={neighbor.address} placeholder="No address" onOpen={onEdit} onRemove={onDelete} removeLabel="Remove Neighbor">
      <Badge variant="outline" className="font-mono">AS{neighbor.remote_asn || '?'}</Badge>
      {neighbor.bfd && <Badge variant="secondary">BFD{neighbor.bfd_profile ? `: ${neighbor.bfd_profile}` : ''}</Badge>}
      {neighbor.description && <span className="truncate text-xs text-muted-foreground">{neighbor.description}</span>}
    </EntityRow>
  )
}

export { BGPNeighborsPanel } from './NeighborsParts/BGPNeighborsPanel'
export { NeighborAFEditor } from './NeighborsParts/NeighborAFEditor'
export { NeighborForm } from './NeighborsParts/NeighborForm'
