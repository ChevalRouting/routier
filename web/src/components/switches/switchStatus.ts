import type { StatusIntent } from 'cheval-ui'
import type { TypesSwitchStatus as SwitchStatus } from '@/api'

export type SwitchHealth = { intent: StatusIntent; label: string }

export function switchHealth(status: SwitchStatus | undefined): SwitchHealth {
  if (!status) return { intent: 'info', label: 'not applied' }
  if (!status.has_credentials) return { intent: 'warning', label: 'no credentials' }
  if (status.reachable) return { intent: 'success', label: 'online' }
  if (status.read_at === 0) return { intent: 'neutral', label: 'not read yet' }
  return { intent: 'danger', label: 'unreachable' }
}
