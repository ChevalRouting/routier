import { ClauseOp } from '@/components/routing/ClauseEditor'
import { cn, Input, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'

type ClauseValueInputShape = {
  op: ClauseOp | undefined
  value: string
  onChange: (v: string) => void
  prefixListNames: string[]
}

export function ClauseValueInput({
  op, value, onChange, prefixListNames,
}: ClauseValueInputShape) {
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
