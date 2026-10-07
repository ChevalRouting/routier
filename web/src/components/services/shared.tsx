import { Badge, Button } from 'cheval-ui'
import { Trash2 } from 'lucide-react'

export interface ServiceConfig {
  enable: boolean
  configs: ConfigFile[]
  after: string
}

export interface ConfigFile {
  src: string
  dest: string
  mode: number
}

export type ServicesMap = Record<string, ServiceConfig>

let nextFileId = 1

export function newConfigFileId() { return nextFileId++ }

export interface ConfigFileRow extends ConfigFile {
  id: number
}

export function defaultService(): ServiceConfig {
  return { enable: true, configs: [], after: '' }
}

export function buildPayload(services: ServicesMap, configRows: Record<string, ConfigFileRow[]>): ServicesMap {
  const payload: ServicesMap = {}
  for (const name of Object.keys(services)) {
    payload[name] = {
      ...services[name],
      configs: (configRows[name] ?? []).map(({ id: _id, ...rest }) => rest),
    }
  }
  return payload
}

export interface ServiceFormProps {
  service: ServiceConfig
  rows: ConfigFileRow[]
  onServiceChange: (s: ServiceConfig) => void
  onRowsChange: (r: ConfigFileRow[]) => void
  nameInput?: string
  onNameChange?: (n: string) => void
  onAdd?: () => void
  onDone: () => void
}

export interface ServiceRowProps {
  name: string
  service: ServiceConfig
  configCount: number
  onEdit: () => void
  onDelete: () => void
}

export function ServiceRow({ name, service, configCount, onEdit, onDelete }: ServiceRowProps) {
  return (
    <div
      onClick={onEdit}
      className="flex items-start gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors"
    >
      <span className="font-mono font-semibold text-sm w-28 shrink-0 truncate pt-0.5">{name}</span>
      <div className="flex flex-wrap gap-1 flex-1 min-w-0">
        <Badge variant={service.enable ? 'success' : 'neutral'} className="text-xs">
          {service.enable ? 'enabled' : 'disabled'}
        </Badge>
        {service.after && (
          <Badge variant="outline" className="text-xs font-mono">after: {service.after}</Badge>
        )}
        {configCount > 0 && (
          <Badge variant="outline" className="text-xs">{configCount} file{configCount !== 1 ? 's' : ''}</Badge>
        )}
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={(e) => { e.stopPropagation(); onDelete() }}
        className="h-7 w-7 p-0 hover:text-destructive shrink-0"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

export const NEW_KEY = '__new__'
