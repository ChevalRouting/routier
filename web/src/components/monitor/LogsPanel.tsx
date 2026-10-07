

type LOGSUBTABSShape = { key: LogSubTab; label: string }

export type LogSource = 'messages' | 'dmesg'
export type LogSubTab = LogSource | 'settings'

export const LOG_LEVELS = ['debug', 'info', 'warn', 'error']
export const LOG_TARGETS = ['', 'syslog', 'file', 'stdout', 'stderr']

export interface LoggingConfig { target?: string; file?: string; level?: string }

export const LOG_SUBTABS: LOGSUBTABSShape[] = [
  { key: 'messages', label: 'System Log' },
  { key: 'dmesg',    label: 'Kernel Log' },
  { key: 'settings', label: 'Log Settings' },
]

export { LogSettingsPanel } from './LogsPanelParts/LogSettingsPanel'
export { LogsPanel } from './LogsPanelParts/LogsPanel'
