import { checkIP } from '@/lib/validate'
import type { RouteRow } from '@/components/routing/StaticRoutesTab'

export type RouteAttribute = 'type' | 'via' | 'dev' | 'src' | 'metric' | 'scope'

export type RouteAttributeSpec = {
  key: RouteAttribute
  label: string
  placeholder: string
  options?: string[]
  check?: (value: string) => string | null
}

export const ROUTE_SCOPES = ['global', 'link', 'host', 'site']

export const ROUTE_TYPES = ['blackhole', 'unreachable', 'prohibit']

export const ROUTE_ATTRIBUTES: RouteAttributeSpec[] = [
  { key: 'type', label: 'Route Type', placeholder: 'blackhole', options: ROUTE_TYPES },
  { key: 'via', label: 'Gateway (via)', placeholder: '192.168.1.1', check: checkIP },
  { key: 'dev', label: 'Interface (dev)', placeholder: 'wan' },
  { key: 'src', label: 'Source Address (src)', placeholder: '2001:db8::10', check: checkIP },
  { key: 'metric', label: 'Metric', placeholder: '100', check: checkMetric },
  { key: 'scope', label: 'Scope', placeholder: 'link', options: ROUTE_SCOPES },
]

function checkMetric(value: string) {
  return value === '' || /^\d+$/.test(value) ? null : 'Enter a whole number'
}

export function attributeValue(row: RouteRow, key: RouteAttribute): string {
  if (key === 'metric') return row.metric ? String(row.metric) : ''
  return row[key] ?? ''
}

export function attributePatch(key: RouteAttribute, value: string): Partial<RouteRow> {
  if (key === 'metric') return { metric: /^\d+$/.test(value) ? Number(value) : 0 }
  if (key === 'scope' || key === 'type') return { [key]: value || undefined }
  return { [key]: value }
}
