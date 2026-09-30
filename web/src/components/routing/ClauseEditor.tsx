import { useState } from 'react'
import { Input } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue, SelectGroup, SelectLabel } from 'cheval-ui'
import { cn } from 'cheval-ui'
import { KVPair, newId } from './shared'

export type ClauseKind = 'text' | 'number' | 'none' | 'prefix-list' | 'select'
export interface ClauseOp {
  key: string
  label: string
  desc: string
  group: string
  kind?: ClauseKind
  options?: string[]
  placeholder?: string
}

export const CUSTOM_OP = '__custom__'

export const MATCH_OPS: ClauseOp[] = [
  { key: 'ip address prefix-list', label: 'IP prefix-list', desc: 'Match IPv4 prefixes against a prefix-list', group: 'Prefix', kind: 'prefix-list' },
  { key: 'ipv6 address prefix-list', label: 'IPv6 prefix-list', desc: 'Match IPv6 prefixes against a prefix-list', group: 'Prefix', kind: 'prefix-list' },
  { key: 'ip next-hop prefix-list', label: 'Next-hop prefix-list', desc: 'Match next-hop against a prefix-list', group: 'Prefix', kind: 'prefix-list' },
  { key: 'as-path', label: 'AS-path', desc: 'Match against an as-path access-list', group: 'BGP', placeholder: 'AS-PATH-ACL' },
  { key: 'community', label: 'Community', desc: 'Match a community-list', group: 'BGP', placeholder: 'COMM-LIST' },
  { key: 'large-community', label: 'Large community', desc: 'Match a large-community-list', group: 'BGP', placeholder: 'LARGE-LIST' },
  { key: 'extcommunity', label: 'Ext community', desc: 'Match an extended-community-list', group: 'BGP', placeholder: 'EXT-LIST' },
  { key: 'local-preference', label: 'Local preference', desc: 'Match on local preference value', group: 'BGP', kind: 'number', placeholder: '100' },
  { key: 'metric', label: 'Metric (MED)', desc: 'Match on the metric / MED', group: 'BGP', kind: 'number', placeholder: '100' },
  { key: 'origin', label: 'Origin', desc: 'Match BGP origin code', group: 'BGP', kind: 'select', options: ['igp', 'egp', 'incomplete'] },
  { key: 'peer', label: 'Peer', desc: 'Match the peer the update came from', group: 'BGP', placeholder: '10.0.0.1' },
  { key: 'source-protocol', label: 'Source protocol', desc: 'Match the routing protocol that sourced the route', group: 'Advanced', kind: 'select', options: ['bgp', 'ospf', 'ospf6', 'rip', 'static', 'connected', 'kernel', 'isis'] },
  { key: 'source-vrf', label: 'Source VRF', desc: 'Match the VRF the route was learned in', group: 'Advanced', placeholder: 'default' },
  { key: 'tag', label: 'Tag', desc: 'Match the route tag', group: 'Advanced', kind: 'number', placeholder: '100' },
]

export const SET_OPS: ClauseOp[] = [
  { key: 'local-preference', label: 'Local preference', desc: 'BGP local preference (higher is preferred)', group: 'Common', kind: 'number', placeholder: '200' },
  { key: 'metric', label: 'Metric (MED)', desc: 'MED metric (absolute), or +N / -N to adjust', group: 'Common', placeholder: '100 or +10' },
  { key: 'weight', label: 'Weight', desc: 'Cisco-style weight (higher is preferred, local only)', group: 'Common', kind: 'number', placeholder: '100' },
  { key: 'origin', label: 'Origin', desc: 'BGP origin code', group: 'Common', kind: 'select', options: ['igp', 'egp', 'incomplete'] },
  { key: 'ip next-hop', label: 'IP next-hop', desc: 'Rewrite the IPv4 next-hop', group: 'Common', placeholder: '10.0.0.1' },
  { key: 'ipv6 next-hop global', label: 'IPv6 next-hop', desc: 'Rewrite the IPv6 global next-hop', group: 'Common', placeholder: '2001:db8::1' },
  { key: 'as-path prepend', label: 'AS-path prepend', desc: 'Prepend ASNs to the AS-path', group: 'AS-path', placeholder: '65001 65001' },
  { key: 'as-path exclude', label: 'AS-path exclude', desc: 'Remove matching ASNs from the AS-path', group: 'AS-path', placeholder: '65001' },
  { key: 'community', label: 'Community', desc: 'Set communities (append with "additive")', group: 'Community', placeholder: '65000:100 or no-export' },
  { key: 'large-community', label: 'Large community', desc: 'Set large communities', group: 'Community', placeholder: '65000:1:100' },
  { key: 'comm-list', label: 'Delete communities', desc: 'Delete communities matching a community-list', group: 'Community', placeholder: 'COMM-LIST delete' },
  { key: 'aggregator as', label: 'Aggregator', desc: 'Set the BGP aggregator attribute', group: 'Advanced', placeholder: '65000 10.0.0.1' },
  { key: 'atomic-aggregate', label: 'Atomic aggregate', desc: 'Set the atomic-aggregate attribute', group: 'Advanced', kind: 'none' },
  { key: 'originator-id', label: 'Originator ID', desc: 'Set the BGP originator-id attribute', group: 'Advanced', placeholder: '10.0.0.1' },
  { key: 'distance', label: 'Distance', desc: 'Administrative distance for the route', group: 'Advanced', kind: 'number', placeholder: '20' },
  { key: 'tag', label: 'Tag', desc: 'Set the route tag', group: 'Advanced', kind: 'number', placeholder: '100' },
  { key: 'table', label: 'Kernel table', desc: 'Export the route to a non-main kernel table', group: 'Advanced', kind: 'number', placeholder: '100' },
  { key: 'src', label: 'Source address', desc: 'Source address for the installed route', group: 'Advanced', placeholder: '10.0.0.1' },
]

export function ClauseValueInput({
  op, value, onChange, prefixListNames,
}: {
  op: ClauseOp | undefined
  value: string
  onChange: (v: string) => void
  prefixListNames: string[]
}) {
  const cls = 'font-mono h-7 text-xs'
  if (op?.kind === 'none') {
    return <div className="flex h-7 items-center px-1 text-xs text-muted-foreground italic">no value needed</div>
  }
  if (op?.kind === 'prefix-list') {
    return (
      <Select value={value || undefined} onValueChange={onChange}>
        <SelectTrigger className={cn(cls, 'w-full')}><SelectValue placeholder="prefix-list…" /></SelectTrigger>
        <SelectContent>
          {prefixListNames.length === 0 && <div className="px-2 py-1.5 text-xs text-muted-foreground">No prefix lists defined</div>}
          {prefixListNames.map((n) => <SelectItem key={n} value={n} className="font-mono text-xs">{n}</SelectItem>)}
        </SelectContent>
      </Select>
    )
  }
  if (op?.kind === 'select') {
    return (
      <Select value={value || undefined} onValueChange={onChange}>
        <SelectTrigger className={cn(cls, 'w-full')}><SelectValue placeholder="choose…" /></SelectTrigger>
        <SelectContent>
          {op.options!.map((o) => <SelectItem key={o} value={o} className="font-mono text-xs">{o}</SelectItem>)}
        </SelectContent>
      </Select>
    )
  }
  return (
    <Input
      value={value}
      inputMode={op?.kind === 'number' ? 'numeric' : undefined}
      onChange={(e) => onChange(e.target.value)}
      placeholder={op?.placeholder ?? 'value'}
      className={cn(cls, 'w-full')}
    />
  )
}

export function ClauseEditor({
  pairs, onChange, catalog, prefixListNames, verb,
}: {
  pairs: KVPair[]
  onChange: (v: KVPair[]) => void
  catalog: ClauseOp[]
  prefixListNames: string[]
  verb: 'match' | 'set'
}) {
  const [customIds, setCustomIds] = useState<Set<number>>(new Set())
  const groups = [...new Set(catalog.map((o) => o.group))]

  const upd = (id: number, patch: Partial<KVPair>) =>
    onChange(pairs.map((p) => (p._id === id ? { ...p, ...patch } : p)))
  const add = () => onChange([...pairs, { _id: newId(), key: '', value: '' }])
  const remove = (id: number) => onChange(pairs.filter((p) => p._id !== id))

  const pickOp = (id: number, opKey: string) => {
    if (opKey === CUSTOM_OP) {
      setCustomIds((s) => new Set(s).add(id))
      upd(id, { key: '' })
      return
    }
    setCustomIds((s) => { const n = new Set(s); n.delete(id); return n })
    upd(id, { key: opKey, value: '' })
  }

  return (
    <div className="space-y-1.5">
      {pairs.map(({ _id, key, value }) => {
        const op = catalog.find((o) => o.key === key)
        const isCustom = customIds.has(_id) || (!op && key !== '')
        const selectValue = isCustom ? CUSTOM_OP : (op ? op.key : undefined)
        return (
          <div key={_id} className="flex items-start gap-1.5">
            <Select value={selectValue} onValueChange={(v) => pickOp(_id, v)}>
              <SelectTrigger className="h-7 text-xs w-40 shrink-0"><SelectValue placeholder={`${verb}…`} /></SelectTrigger>
              <SelectContent>
                {groups.map((g) => (
                  <SelectGroup key={g}>
                    <SelectLabel className="text-[10px] uppercase tracking-wide text-muted-foreground">{g}</SelectLabel>
                    {catalog.filter((o) => o.group === g).map((o) => (
                      <SelectItem key={o.key} value={o.key} className="text-xs">{o.label}</SelectItem>
                    ))}
                  </SelectGroup>
                ))}
                <SelectGroup>
                  <SelectItem value={CUSTOM_OP} className="text-xs italic">Custom…</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>

            <div className="flex-1 min-w-0 space-y-1.5">
              {isCustom ? (
                <>
                  <Input
                    value={key}
                    onChange={(e) => upd(_id, { key: e.target.value })}
                    placeholder={`raw ${verb} token`}
                    className="font-mono h-7 text-xs w-full"
                  />
                  <Input
                    value={value}
                    onChange={(e) => upd(_id, { value: e.target.value })}
                    placeholder="value"
                    className="font-mono h-7 text-xs w-full"
                  />
                </>
              ) : (
                <>
                  <ClauseValueInput op={op} value={value} onChange={(v) => upd(_id, { value: v })} prefixListNames={prefixListNames} />
                  {op && <p className="text-[11px] leading-tight text-muted-foreground px-0.5">{op.desc}</p>}
                </>
              )}
            </div>

            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive"
              onClick={() => remove(_id)}
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
        )
      })}
      <Button
        variant="ghost"
        size="sm"
        className="h-6 text-xs gap-1 text-muted-foreground px-1"
        onClick={add}
      >
        <Plus className="h-3 w-3" />Add {verb}
      </Button>
    </div>
  )
}
