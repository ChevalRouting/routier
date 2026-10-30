import { fieldWidth, RouteAttributeTag } from '@/components/routing/StaticRoutesParts/RouteAttributeTag'
import { attributePatch, attributeValue, ROUTE_ATTRIBUTES, type RouteAttribute } from '@/components/routing/StaticRoutesParts/routeAttributes'
import type { RouteRow } from '@/components/routing/StaticRoutesTab'
import { checkCIDR } from '@/lib/validate'
import { Button, cn, Input, MenuButton, type MenuItem } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'

type RouteSentenceShape = {
  row: RouteRow
  onChange: (patch: Partial<RouteRow>) => void
  onRemove: () => void
}

const DESTINATION_PLACEHOLDER = '10.0.0.0/8'

function startingAttributes(row: RouteRow): RouteAttribute[] {
  const blank = row.destination === '' && ROUTE_ATTRIBUTES.every((spec) => attributeValue(row, spec.key) === '')
  return blank ? ['via'] : []
}

export function RouteSentence({ row, onChange, onRemove }: RouteSentenceShape) {
  const [added, setAdded] = useState<RouteAttribute[]>(() => startingAttributes(row))
  const [focused, setFocused] = useState<RouteAttribute | null>(null)
  const destinationError = row.destination ? checkCIDR(row.destination) : null
  const shown = ROUTE_ATTRIBUTES.filter((spec) => added.includes(spec.key) || attributeValue(row, spec.key) !== '')

  const handleAdd = (key: RouteAttribute) => {
    setAdded((current) => [...current, key])
    setFocused(key)
  }

  const handleRemove = (key: RouteAttribute) => {
    setAdded((current) => current.filter((item) => item !== key))
    onChange(attributePatch(key, ''))
  }

  const addItems: MenuItem[] = ROUTE_ATTRIBUTES
    .filter((spec) => !shown.includes(spec))
    .map((spec) => ({ key: spec.key, label: spec.label, onSelect: () => handleAdd(spec.key) }))

  return (
    <div className="flex items-start gap-2 px-3 py-2">
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">
        <Input
          value={row.destination}
          placeholder={DESTINATION_PLACEHOLDER}
          aria-label="Destination"
          aria-invalid={destinationError ? true : undefined}
          title={destinationError ?? undefined}
          onChange={(e) => onChange({ destination: e.target.value })}
          style={fieldWidth(row.destination, DESTINATION_PLACEHOLDER, '1.75rem')}
          className={cn('h-8 font-mono text-xs', destinationError && 'border-danger')}
        />
        {shown.map((spec) => (
          <RouteAttributeTag
            key={spec.key}
            spec={spec}
            value={attributeValue(row, spec.key)}
            autoFocus={focused === spec.key}
            onChange={(value) => onChange(attributePatch(spec.key, value))}
            onRemove={() => handleRemove(spec.key)}
          />
        ))}
        {addItems.length > 0 && (
          <MenuButton label="Add Attribute" sections={[addItems]} className="h-8 gap-1 rounded-md border border-dashed border-input px-2 text-xs text-muted-foreground">
            <Plus className="h-3.5 w-3.5" />Add
          </MenuButton>
        )}
      </div>
      <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0 text-muted-foreground hover:text-destructive" onClick={onRemove} aria-label="Remove Route" title="Remove Route">
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}
