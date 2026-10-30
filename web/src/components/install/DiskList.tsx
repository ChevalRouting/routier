import type { LiveDisk } from '@/api'
import { formatSize } from '@/components/install/shared'
import { Badge, cn, PreferencesGroup } from 'cheval-ui'
import { Check, HardDrive } from 'lucide-react'

type DiskListShape = { disks: LiveDisk[]; selected: string; onSelect: (path: string) => void }

type DiskRowShape = { disk: LiveDisk; selected: boolean; onSelect: (path: string) => void }

function DiskRow({ disk, selected, onSelect }: DiskRowShape) {
  const handleClick = () => onSelect(disk.path)
  return (
    <button
      type="button"
      disabled={disk.boot_media}
      aria-pressed={selected}
      onClick={handleClick}
      className={cn(
        'flex min-h-[3.25rem] w-full items-center gap-3 px-4 py-2.5 text-left transition-colors',
        disk.boot_media ? 'cursor-not-allowed opacity-50' : 'hover:bg-accent/50',
      )}
    >
      <HardDrive className="h-5 w-5 shrink-0 text-muted-foreground" />
      <div className="min-w-0 flex-1">
        <div className="font-mono text-sm font-medium text-foreground">{disk.path}</div>
        <div className="mt-0.5 truncate text-xs text-muted-foreground">
          {formatSize(disk.size)}{disk.model ? ` · ${disk.model}` : ''}
        </div>
      </div>
      {disk.boot_media ? <Badge variant="secondary">Installation Media</Badge> : disk.removable && <Badge variant="secondary">Removable</Badge>}
      <Check className={cn('h-4 w-4 shrink-0 text-primary', !selected && 'invisible')} />
    </button>
  )
}

export function DiskList({ disks, selected, onSelect }: DiskListShape) {
  return (
    <PreferencesGroup title="Target Disk" description="Everything on the selected disk is erased">
      {disks.length === 0 && (
        <p className="px-4 py-3 text-sm text-muted-foreground">No disks detected</p>
      )}
      {disks.map((disk) => (
        <DiskRow key={disk.path} disk={disk} selected={disk.path === selected} onSelect={onSelect} />
      ))}
    </PreferencesGroup>
  )
}
