import { api } from '@/lib/client'

export const DEFAULT_WATCHDOG = 60

export type Phase = 'streaming' | 'completed' | 'interrupted' | 'reconnected' | 'confirming' | 'kept' | 'done' | 'rolledback' | 'failed'

export const armed = (p: Phase) => p === 'completed' || p === 'reconnected'

export async function lastApplyFailed(): Promise<boolean> {
  const logs = await api.apiApplyLogsGet()
  const latest = logs.records?.find((r) => r.source === 'web')
  return latest?.result === 'failed'
}

export interface ApplyStreamModalProps {
  open: boolean
  onClose: () => void
}

export { ApplyStreamModal } from './ApplyStreamModalParts/ApplyStreamModal'
export { StatusBar } from './ApplyStreamModalParts/StatusBar'
