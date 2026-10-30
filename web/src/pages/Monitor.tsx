import { useHasParam } from '@/lib/useHasParam'
import { monitorSections, type MonitorGroup, type MonitorSectionKey } from '@/lib/navigation'
import { BGPPanel } from '@/components/monitor/BGPPanel'
import { FlowsPanel } from '@/components/monitor/FlowsPanel'
import { LiveTrafficPanel } from '@/components/monitor/LiveTrafficPanel'
import { LogsPanel } from '@/components/monitor/LogsPanel'
import { NeighborsPanel } from '@/components/monitor/NeighborsPanel'
import { ProbesPanel } from '@/components/monitor/ProbesPanel'
import { ProcessesPanel } from '@/components/monitor/ProcessesPanel'
import { TabAction } from '@/components/monitor/shared'
import { SystemPanel } from '@/components/monitor/SystemPanel'
import { TrafficPanel } from '@/components/monitor/TrafficPanel'
import { PageHeader, SectionNav, Spinner, useTabState } from 'cheval-ui'
import { lazy, Suspense, useState } from 'react'

const Topology = lazy(() => import('@/pages/RoutingOverview'))

type MonitorShape = { group: MonitorGroup }

export default function Monitor({ group }: MonitorShape) {
  const tabs = monitorSections.filter((section) => section.group === group).map(({ key, label, group }) => ({ key, label, group }))
  const [requested, setActiveTab] = useTabState<MonitorSectionKey>('monitor', tabs[0].key)
  const activeTab = tabs.some((tab) => tab.key === requested) ? requested : tabs[0].key
  const deepLinked = useHasParam('monitor')
  const [tabAction, setTabAction] = useState<TabAction>(null)

  const handleTabChange = (tab: MonitorSectionKey) => {
    setActiveTab(tab)
    setTabAction(null)
  }

  return (
    <div className="space-y-6">
      <PageHeader title={group} action={tabAction} />
      <SectionNav items={tabs} active={activeTab} onChange={handleTabChange} initiallyEntered={deepLinked}>
        {activeTab === 'system'     && <SystemPanel onActionChange={setTabAction} />}
        {activeTab === 'processes'  && <ProcessesPanel onActionChange={setTabAction} />}
        {activeTab === 'logs'       && <LogsPanel />}
        {activeTab === 'topology'   && <Suspense fallback={<Spinner />}><Topology /></Suspense>}
        {activeTab === 'traffic'    && <TrafficPanel />}
        {activeTab === 'live'       && <LiveTrafficPanel />}
        {activeTab === 'flows'      && <FlowsPanel />}
        {activeTab === 'bgp'        && <BGPPanel onActionChange={setTabAction} />}
        {activeTab === 'neighbors'  && <NeighborsPanel />}
        {activeTab === 'probes'     && <ProbesPanel onActionChange={setTabAction} />}
      </SectionNav>
    </div>
  )
}
