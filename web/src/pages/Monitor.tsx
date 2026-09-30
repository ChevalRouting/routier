import { useState, useEffect } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { useTabState } from 'cheval-ui'
import { Tabs } from 'cheval-ui'
import { PageHeader } from 'cheval-ui'
import { TabAction } from '@/components/monitor/shared'
import { SystemPanel } from '@/components/monitor/SystemPanel'
import { TrafficPanel } from '@/components/monitor/TrafficPanel'
import { LogsPanel } from '@/components/monitor/LogsPanel'
import { NeighborsPanel } from '@/components/monitor/NeighborsPanel'
import { BGPPanel } from '@/components/monitor/BGPPanel'
import { ProcessesPanel } from '@/components/monitor/ProcessesPanel'
import { ProbesPanel } from '@/components/monitor/ProbesPanel'

type MonitorTab = 'system' | 'bgp' | 'traffic' | 'logs' | 'neighbors' | 'processes' | 'probes'

const MONITOR_TABS: { key: MonitorTab; label: string }[] = [
  { key: 'system',     label: 'System' },
  { key: 'processes',  label: 'Processes' },
  { key: 'bgp',        label: 'BGP' },
  { key: 'traffic',    label: 'Traffic' },
  { key: 'logs',       label: 'Logs' },
  { key: 'neighbors',  label: 'Neighbors' },
  { key: 'probes',     label: 'Probes' },
]

const MOVED: Record<string, string> = { dhcp: '/dhcp', dns: '/dns' }

export default function Monitor() {
  const [activeTab, setActiveTab] = useTabState<MonitorTab>('monitor', 'system' as MonitorTab)
  const [tabAction, setTabAction] = useState<TabAction>(null)
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()

  useEffect(() => {
    const requested = searchParams.get('tab') ?? searchParams.get('monitor')
    const t = requested === 'nics' ? 'system' : requested
    if ((activeTab as string) === 'nics') setActiveTab('system')
    if (t && MOVED[t]) {
      navigate(MOVED[t], { replace: true })
      return
    }

    if (t && MONITOR_TABS.some((m) => m.key === t)) {
      setActiveTab(t as MonitorTab)
      setTabAction(null)
      searchParams.delete('tab')
      setSearchParams(searchParams, { replace: true })
    }
  }, [])

  const handleTabChange = (tab: MonitorTab) => {
    setActiveTab(tab)
    setTabAction(null)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Monitor"
        description="Live system statistics, traffic, logs, and network state"
        action={tabAction}
      />
      <Tabs tabs={MONITOR_TABS} active={activeTab} onChange={handleTabChange} />
      <div>
        {activeTab === 'system'     && <SystemPanel onActionChange={setTabAction} />}
        {activeTab === 'processes'  && <ProcessesPanel onActionChange={setTabAction} />}
        {activeTab === 'bgp'        && <BGPPanel onActionChange={setTabAction} />}
        {activeTab === 'traffic'    && <TrafficPanel />}
        {activeTab === 'logs'       && <LogsPanel />}
        {activeTab === 'neighbors'  && <NeighborsPanel />}
        {activeTab === 'probes'     && <ProbesPanel onActionChange={setTabAction} />}
      </div>
    </div>
  )
}
