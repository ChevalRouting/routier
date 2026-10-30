import { ClauseEditor, MATCH_OPS, SET_OPS } from '@/components/routing/ClauseEditor'
import { RouteMapRow } from '@/components/routing/bgp/RouteMaps'
import { Button, ComboRow, PreferencesGroup, Segmented } from 'cheval-ui'
import { NumberRow } from '@/components/ui/NumberRow'
import { SelectRow } from '@/components/ui/SelectRow'
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
    <PreferencesGroup
      title={`Entry ${row.seq || ''}`.trim()}
      header={(
        <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-destructive" onClick={onDelete} aria-label="Remove Entry" title="Remove Entry">
          <Trash2 className="h-4 w-4" />
        </Button>
      )}
    >
      <NumberRow title="Seq" value={row.seq} onChange={(v) => onChange({ seq: v ?? 0 })} />
      <ComboRow title="Action">
        <Segmented
          value={row.action}
          onChange={(v) => onChange({ action: v })}
          options={[
            { value: 'permit', label: 'permit', activeClass: 'bg-success text-success-foreground' },
            { value: 'deny', label: 'deny', activeClass: 'bg-danger text-danger-foreground' },
          ]}
        />
      </ComboRow>
      <div className="space-y-2 px-4 py-3">
        <p className="text-sm font-medium">Match</p>
        <ClauseEditor pairs={row.match} onChange={(v) => onChange({ match: v })} catalog={MATCH_OPS} prefixListNames={prefixListNames} verb="match" />
      </div>
      <div className="space-y-2 px-4 py-3">
        <p className="text-sm font-medium">Set</p>
        <ClauseEditor pairs={row.set} onChange={(v) => onChange({ set: v })} catalog={SET_OPS} prefixListNames={prefixListNames} verb="set" />
      </div>
      <SelectRow title="Call Route Map" value={row.call} onChange={(v) => onChange({ call: v ?? '' })} options={callTargets} noneLabel="None" />
      <ComboRow title="On Match">
        <Segmented
          value={onMatchMode}
          onChange={(m) => onChange({ on_match: m === 'none' ? '' : m === 'next' ? 'next' : `goto ${gotoSeq || row.seq + 10}` })}
          options={[{ value: 'none', label: 'default' }, { value: 'next', label: 'next' }, { value: 'goto', label: 'goto' }]}
        />
      </ComboRow>
      {onMatchMode === 'goto' && (
        <NumberRow title="Goto Seq" value={gotoSeq} onChange={(v) => onChange({ on_match: `goto ${v ?? 0}` })} />
      )}
      <NumberRow title="Continue Seq" value={row.continue} onChange={(v) => onChange({ continue: v ?? 0 })} />
    </PreferencesGroup>
  )
}
