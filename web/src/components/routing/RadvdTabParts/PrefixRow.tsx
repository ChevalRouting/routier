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
          className="font-mono text-xs h-8 flex-1"
        />
        <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0 hover:text-destructive" onClick={onDelete}>
          <Trash2 className="h-3.5 w-3.5" />
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

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label className="text-[11px]">Valid lifetime</Label>
          <Input
            value={prefix.adv_valid_lifetime ?? ''}
            onChange={(e) => set('adv_valid_lifetime', e.target.value || undefined)}
            placeholder="86400 or infinity"
            className="font-mono text-xs h-8"
          />
        </div>
        <div className="space-y-1.5">
          <Label className="text-[11px]">Preferred lifetime</Label>
          <Input
            value={prefix.adv_preferred_lifetime ?? ''}
            onChange={(e) => set('adv_preferred_lifetime', e.target.value || undefined)}
            placeholder="14400 or infinity"
            className="font-mono text-xs h-8"
          />
        </div>
      </div>
    </div>
  )
}
