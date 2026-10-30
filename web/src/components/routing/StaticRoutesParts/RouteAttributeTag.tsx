import { type RouteAttributeSpec } from '@/components/routing/StaticRoutesParts/routeAttributes'
import { Button, cn, Input, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { X } from 'lucide-react'

type RouteAttributeTagShape = {
  spec: RouteAttributeSpec
  value: string
  autoFocus: boolean
  onChange: (value: string) => void
  onRemove: () => void
}

export function fieldWidth(value: string, placeholder: string, padding: string) {
  return { width: `calc(${Math.max(value.length, placeholder.length, 3) + 1}ch + ${padding})` }
}

export function RouteAttributeTag({ spec, value, autoFocus, onChange, onRemove }: RouteAttributeTagShape) {
  const error = spec.check?.(value) ?? null
  return (
    <span
      className={cn('inline-flex h-8 items-center gap-0.5 rounded-md border border-input bg-muted/40 pl-2 font-mono text-xs focus-within:ring-1 focus-within:ring-ring', error && 'border-danger')}
      title={error ?? undefined}
    >
      <span className="text-muted-foreground">{spec.key}</span>
      {spec.options ? (
        <Select value={value || undefined} onValueChange={onChange} defaultOpen={autoFocus}>
          <SelectTrigger aria-label={spec.label} className="h-7 w-auto select-none gap-1 border-0 bg-transparent px-1 font-mono text-xs [box-shadow:none] focus:ring-0 focus:ring-offset-0">
            <SelectValue placeholder={spec.placeholder} />
          </SelectTrigger>
          <SelectContent>
            {spec.options.map((option) => <SelectItem key={option} value={option} className="font-mono text-xs">{option}</SelectItem>)}
          </SelectContent>
        </Select>
      ) : (
        <Input
          value={value}
          autoFocus={autoFocus}
          inputMode={spec.key === 'metric' ? 'numeric' : undefined}
          placeholder={spec.placeholder}
          aria-label={spec.label}
          aria-invalid={error ? true : undefined}
          onChange={(e) => onChange(e.target.value)}
          style={fieldWidth(value, spec.placeholder, '0.5rem')}
          className="h-7 border-0 bg-transparent px-1 font-mono text-xs [box-shadow:none] focus-visible:ring-0 focus-visible:ring-offset-0"
        />
      )}
      <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive" onClick={onRemove} aria-label={`Remove ${spec.label}`} title={`Remove ${spec.label}`}>
        <X className="h-3 w-3" />
      </Button>
    </span>
  )
}
