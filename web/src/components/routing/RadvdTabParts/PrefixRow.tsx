import { RADVDPrefix } from '@/components/routing/RadvdTab'
import { Button, Input, Label, Switch } from 'cheval-ui'
import { Trash2 } from 'lucide-react'

type PrefixRowShape = {
  prefix: RADVDPrefix
  onChange: (p: RADVDPrefix) => void
  onDelete: () => void
}

export function PrefixRow({
  prefix,
  onChange,
  onDelete,
}: PrefixRowShape) {
  const set = <K extends keyof RADVDPrefix>(k: K, v: RADVDPrefix[K]) =>
    onChange({ ...prefix, [k]: v })

  return (
    <div className="rounded-md bg-card p-3 space-y-3 shadow-[var(--card-shadow)]">
      <div className="flex items-center gap-2">
        <Input
          value={prefix.prefix}
          onChange={(e) => set('prefix', e.target.value)}
          placeholder="2001:db8::/64"
          className="flex-1 font-mono"
        />
        <Button variant="ghost" size="icon" className="shrink-0 text-muted-foreground hover:text-destructive" onClick={onDelete} aria-label="Remove Prefix" title="Remove Prefix">
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-2">
        {(
          [
            ['adv_on_link', 'On-link (L)'],
            ['adv_autonomous', 'Autonomous (A)'],
            ['adv_router_addr', 'Router addr (R)'],
          ] as const
        ).map(([key, label]) => (
          <div key={key} className="flex items-center justify-between gap-2">
            <span className="text-xs text-muted-foreground">{label}</span>
            <Switch checked={!!prefix[key]} onCheckedChange={(v) => set(key, v || undefined)} />
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label className="text-xs">Valid Lifetime</Label>
          <Input
            value={prefix.adv_valid_lifetime ?? ''}
            onChange={(e) => set('adv_valid_lifetime', e.target.value || undefined)}
            placeholder="86400 or infinity"
            className="font-mono"
          />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs">Preferred Lifetime</Label>
          <Input
            value={prefix.adv_preferred_lifetime ?? ''}
            onChange={(e) => set('adv_preferred_lifetime', e.target.value || undefined)}
            placeholder="14400 or infinity"
            className="font-mono"
          />
        </div>
      </div>
    </div>
  )
}
