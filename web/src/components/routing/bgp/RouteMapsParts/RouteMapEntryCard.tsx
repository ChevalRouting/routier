import { ClauseEditor, MATCH_OPS, SET_OPS } from '@/components/routing/ClauseEditor'
import { RouteMapRow } from '@/components/routing/bgp/RouteMaps'
import { Button, Label, NumberInput, PreferencesGroup, Segmented, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { Trash2 } from 'lucide-react'

type RouteMapEntryCardShape = {
  row: RouteMapRow
  onChange: (patch: Partial<RouteMapRow>) => void
  onDelete: () => void
  prefixListNames: string[]
  callTargets: string[]
}

export function RouteMapEntryCard({
  row, onChange, onDelete, prefixListNames, callTargets,
}: RouteMapEntryCardShape) {
  const onMatchMode: 'none' | 'next' | 'goto' =
    row.on_match === 'next' ? 'next' : row.on_match.startsWith('goto') ? 'goto' : 'none'
  const gotoSeq = onMatchMode === 'goto' ? Number(row.on_match.slice(5)) || 0 : 0

  return (
    <PreferencesGroup>
      <div className="flex items-end gap-4 px-4 py-3">
        <div className="space-y-1.5">
          <Label className="text-xs text-muted-foreground">Seq</Label>
          <NumberInput
            value={row.seq || undefined}
            onChange={(v) => onChange({ seq: v ?? 0 })}
            className="font-mono h-8 text-xs w-24"
          />
        </div>
        <div className="space-y-1.5 mr-4">
          <Label className="mr-4 text-xs text-muted-foreground">Action</Label>
          <Segmented
            value={row.action}
            onChange={(v) => onChange({ action: v })}
            options={[
              { value: 'permit', label: 'permit', activeClass: 'bg-success text-success-foreground' },
              { value: 'deny', label: 'deny', activeClass: 'bg-danger text-danger-foreground' },
            ]}
          />
        </div>
        <div className="flex-1" />
        <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-destructive" onClick={onDelete}>
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>

      <div className="px-4 py-3 space-y-4">
        <div className="space-y-1.5">
          <Label className="text-xs font-semibold text-muted-foreground">Match</Label>
          <ClauseEditor pairs={row.match} onChange={(v) => onChange({ match: v })} catalog={MATCH_OPS} prefixListNames={prefixListNames} verb="match" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs font-semibold text-muted-foreground">Set</Label>
          <ClauseEditor pairs={row.set} onChange={(v) => onChange({ set: v })} catalog={SET_OPS} prefixListNames={prefixListNames} verb="set" />
        </div>
      </div>

      <div className="px-4 py-3 space-y-3">
        <Label className="text-xs font-semibold text-muted-foreground">Flow control</Label>
        <div className="flex flex-wrap items-end gap-4">
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">Call route-map</Label>
            <Select value={row.call || '_none'} onValueChange={(v) => onChange({ call: v === '_none' ? '' : v })}>
              <SelectTrigger className="h-8 text-xs font-mono w-44"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="_none">- none -</SelectItem>
                {callTargets.map((n) => <SelectItem key={n} value={n} className="font-mono text-xs">{n}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5 mr-4">
            <Label className="text-xs text-muted-foreground">On match</Label>
            <Segmented
              value={onMatchMode}
              onChange={(m) => onChange({ on_match: m === 'none' ? '' : m === 'next' ? 'next' : `goto ${gotoSeq || row.seq + 10}` })}
              options={[{ value: 'none', label: 'default' }, { value: 'next', label: 'next' }, { value: 'goto', label: 'goto' }]}
            />
          </div>
          {onMatchMode === 'goto' && (
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Goto seq</Label>
              <NumberInput
                value={gotoSeq || undefined}
                onChange={(v) => onChange({ on_match: `goto ${v ?? 0}` })}
                className="font-mono h-8 text-xs w-24"
              />
            </div>
          )}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">Continue seq</Label>
            <NumberInput
              value={row.continue || undefined}
              onChange={(v) => onChange({ continue: v ?? 0 })}
              placeholder="–"
              className="font-mono h-8 text-xs w-24"
            />
          </div>
        </div>
      </div>
    </PreferencesGroup>
  )
}
