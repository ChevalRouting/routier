import { DHCP_LOG_TABS, DhcpLogSource } from '@/components/monitor/DhcpPanel'
import { LiveLogControls, LiveLogView, useLiveLog } from '@/components/monitor/LiveLog'
import { Tabs, useTabState } from 'cheval-ui'

export function DhcpLogsPanel() {
  const [source, setSource] = useTabState<DhcpLogSource>('monitor.dhcp.logs', 'kea-dhcp4')
  const log = useLiveLog(`/api/dhcp/leases/stream?source=${encodeURIComponent(source)}`)

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Tabs tabs={DHCP_LOG_TABS} active={source} onChange={setSource} variant="pills" />
        <LiveLogControls log={log} />
      </div>

      <LiveLogView log={log} waitingLabel={`Waiting for ${source} log output…`} />
    </div>
  )
}
