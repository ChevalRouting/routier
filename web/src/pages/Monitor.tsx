import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTabState } from '@/lib/useTabState'
import { api } from '@/lib/client'
import { Tabs } from '@/components/ui/tabs'
import { PageHeader } from '@/components/PageHeader'
import { TabAction } from '@/components/monitor/shared'
import { SystemPanel } from '@/components/monitor/SystemPanel'
import { CollectionPanel } from '@/components/monitor/CollectionPanel'
import { TrafficPanel } from '@/components/monitor/TrafficPanel'
import { LogsPanel } from '@/components/monitor/LogsPanel'
import { HAStatusPanel } from '@/components/monitor/HAStatusPanel'
import { NeighborsPanel } from '@/components/monitor/NeighborsPanel'
import { BGPPanel } from '@/components/monitor/BGPPanel'
import { ProcessesPanel } from '@/components/monitor/ProcessesPanel'
import { DhcpPanel } from '@/components/monitor/DhcpPanel'

type MonitorTab = 'system' | 'bgp' | 'traffic' | 'logs' | 'ha-status' | 'neighbors' | 'processes' | 'collection' | 'dhcp'

const MONITOR_TABS: { key: MonitorTab; label: string }[] = [
  { key: 'system',     label: 'System' },
  { key: 'processes',  label: 'Processes' },
  { key: 'bgp',        label: 'BGP' },
  { key: 'traffic',    label: 'Traffic' },
  { key: 'logs',       label: 'Logs' },
  { key: 'ha-status',  label: 'HA Status' },
  { key: 'neighbors',  label: 'Neighbors' },
  { key: 'collection', label: 'Collection' },
]

export default function Monitor() {
  const [activeTab, setActiveTab] = useTabState<MonitorTab>('monitor', 'system' as MonitorTab)
  const [tabAction, setTabAction] = useState<TabAction>(null)
  const [searchParams, setSearchParams] = useSearchParams()
  const [dhcpEnabled, setDhcpEnabled] = useState(false)

  useEffect(() => {
    api.apiConfigSectionGet({ section: 'dhcp' })
      .then((d) => setDhcpEnabled(!!(d as { enabled?: boolean } | null)?.enabled))
      .catch(() => {})
  }, [])

  const tabs = dhcpEnabled ? [...MONITOR_TABS, { key: 'dhcp' as MonitorTab, label: 'DHCP' }] : MONITOR_TABS

  useEffect(() => {
    const t = searchParams.get('tab')
    if (t && (MONITOR_TABS.some((m) => m.key === t) || t === 'dhcp')) {
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
      <Tabs tabs={tabs} active={activeTab} onChange={handleTabChange} />
      <div>
        {activeTab === 'system'     && <SystemPanel onActionChange={setTabAction} />}
        {activeTab === 'processes'  && <ProcessesPanel onActionChange={setTabAction} />}
        {activeTab === 'bgp'        && <BGPPanel onActionChange={setTabAction} />}
        {activeTab === 'traffic'    && <TrafficPanel />}
        {activeTab === 'logs'       && <LogsPanel />}
        {activeTab === 'ha-status'  && <HAStatusPanel />}
        {activeTab === 'neighbors'  && <NeighborsPanel />}
        {activeTab === 'collection' && <CollectionPanel onActionChange={setTabAction} />}
        {activeTab === 'dhcp'       && <DhcpPanel />}
      </div>
    </div>
  )
}
