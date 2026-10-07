import { Badge, Button } from 'cheval-ui'
import { Trash2 } from 'lucide-react'
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
    <div onClick={onEdit} className="flex items-center gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors">
      <span className="font-mono font-semibold text-sm w-40 shrink-0 truncate">
        {neighbor.address || <span className="text-muted-foreground italic font-normal">no address</span>}
      </span>
      <div className="flex flex-wrap items-center gap-1.5 flex-1 min-w-0">
        <Badge variant="outline" className="text-xs font-mono">AS{neighbor.remote_asn || '?'}</Badge>
        {neighbor.bfd && (
          <Badge variant="secondary" className="text-xs">BFD{neighbor.bfd_profile ? `: ${neighbor.bfd_profile}` : ''}</Badge>
        )}
        {neighbor.description && (
          <span className="text-xs text-muted-foreground truncate">{neighbor.description}</span>
        )}
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={(e) => { e.stopPropagation(); onDelete() }}
        className="h-7 w-7 p-0 hover:text-destructive shrink-0"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

export { BGPNeighborsPanel } from './NeighborsParts/BGPNeighborsPanel'
export { NeighborAFEditor } from './NeighborsParts/NeighborAFEditor'
export { NeighborForm } from './NeighborsParts/NeighborForm'
