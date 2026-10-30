import { LOG_SUBTABS, LogSettingsPanel, LogSubTab } from '@/components/monitor/LogsPanel'
import { LiveLogControls, LiveLogView, useLiveLog } from '@/components/monitor/LiveLog'
import { Tabs, useTabState } from 'cheval-ui'

export function LogsPanel() {
  const [activeTab, setActiveTab] = useTabState<LogSubTab>('monitor.logs', 'messages')
  const isLogTab = activeTab === 'messages' || activeTab === 'dmesg'
  const log = useLiveLog(isLogTab ? `/api/logs/stream?source=${activeTab}` : null)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <Tabs tabs={LOG_SUBTABS} active={activeTab} onChange={setActiveTab} variant="pills" />
        {isLogTab && <LiveLogControls log={log} />}
      </div>

      {isLogTab ? <LiveLogView log={log} waitingLabel="Waiting for log output…" /> : <LogSettingsPanel />}
    </div>
  )
}
