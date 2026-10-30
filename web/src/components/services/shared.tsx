import { plural } from '@/lib/utils'
import { EntityRow } from '@/components/ui/EntityRow'
import { Badge, StatusChip } from 'cheval-ui'

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
    <EntityRow title={name} onOpen={onEdit} onRemove={onDelete} removeLabel="Remove Service">
      <StatusChip intent={service.enable ? 'success' : 'neutral'} label={service.enable ? 'Enabled' : 'Disabled'} />
      {service.after && <Badge variant="outline" className="font-mono">after: {service.after}</Badge>}
      {configCount > 0 && <Badge variant="outline">{plural(configCount, 'file')}</Badge>}
    </EntityRow>
  )
}

export const NEW_KEY = '__new__'
