import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Trash2 } from 'lucide-react'
import { WgIface } from './shared'

export interface WgRowProps {
  name: string
  iface: WgIface
  onEdit: () => void
  onDelete: () => void
}

export function WgRow({ name, iface, onEdit, onDelete }: WgRowProps) {
  return (
    <div onClick={onEdit} className="flex items-start gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors">
      <span className="font-mono font-semibold text-sm w-24 shrink-0 truncate pt-0.5">{name}</span>
      <div className="flex flex-wrap items-start gap-1 flex-1 min-w-0">
        <Badge variant="secondary" className="text-xs font-mono">:{iface.listen_port || 51820}</Badge>
        {iface.friend && <Badge variant="outline" className="text-xs">derived · {iface.friend}</Badge>}
        {(iface.addresses ?? []).map((addr) => (
          <Badge key={addr} variant="outline" className="max-w-full whitespace-normal break-all text-xs font-mono">{addr}</Badge>
        ))}
        {(iface.peers ?? []).length > 0 && (
          <Badge variant="outline" className="text-xs">
            {(iface.peers ?? []).length} peer{(iface.peers ?? []).length > 1 ? 's' : ''}
          </Badge>
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

