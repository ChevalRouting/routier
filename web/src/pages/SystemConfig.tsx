import { useHasParam } from '@/lib/useHasParam'
import { DraftControls } from '@/components/DraftControls'
import { ApiKeysTab } from '@/components/system-config/ApiKeysTab'
import { ApplyHistoryTab } from '@/components/system-config/ApplyHistoryTab'
import { BackupTab } from '@/components/system-config/BackupTab'
import { GeneralTab } from '@/components/system-config/GeneralTab'
import { MonitoringTab } from '@/components/system-config/MonitoringTab'
import { ApiDocsTab, CONFIG_TABS, EMPTY_SAVE, SYSTEM_TABS, SystemTab, TabSaveState } from '@/components/system-config/shared'
import { SnapshotsTab } from '@/components/system-config/SnapshotsTab'
import { SysctlTab } from '@/components/system-config/SysctlTab'
import UpdatesPanel from '@/components/system/UpdatesPanel'
import { useDataVersion } from '@/lib/dataVersion'
import ConfigUsers from '@/pages/ConfigUsers'
import Services from '@/pages/Services'
import { PageHeader, SectionNav, useTabState } from 'cheval-ui'
import { useState } from 'react'

export default function SystemConfig() {
  const [activeTab, setActiveTab] = useTabState<SystemTab>('system-config', 'general')
  const deepLinked = useHasParam('system-config')
  const [saveState, setSaveState] = useState<TabSaveState>(EMPTY_SAVE)
  const { bump } = useDataVersion()

  const handleTabChange = (tab: SystemTab) => {
    setActiveTab(tab)
    setSaveState(EMPTY_SAVE)
  }

  const groupedTab = ['general', 'hostname', 'dns', 'ssh'].includes(activeTab)
  const selectedTab = groupedTab ? 'general' : activeTab
  const isConfigTab = CONFIG_TABS.includes(activeTab) && !groupedTab

  return (
    <div className="space-y-6">
      <PageHeader
        title="General"
        action={isConfigTab ? <DraftControls isDirty={saveState.isDirty} saving={saveState.saving} onClick={saveState.save} /> : undefined}
      />
      <SectionNav items={SYSTEM_TABS} active={selectedTab} onChange={handleTabChange} initiallyEntered={deepLinked}>
        {groupedTab && <GeneralTab initialSection={activeTab} />}
        {activeTab === 'sysctl'    && <SysctlTab onStateChange={setSaveState} />}
        {activeTab === 'monitoring' && <MonitoringTab onStateChange={setSaveState} />}
        {activeTab === 'users' && <ConfigUsers embedded />}
        {activeTab === 'services' && <Services embedded />}
        {activeTab === 'updates'   && <UpdatesPanel />}
        {activeTab === 'apply'     && <ApplyHistoryTab />}
        {activeTab === 'snapshots' && <SnapshotsTab onChanged={bump} />}
        {activeTab === 'backup'    && <BackupTab onChanged={bump} />}
        {activeTab === 'apikeys'   && <ApiKeysTab />}
        {activeTab === 'apidocs'   && <ApiDocsTab />}
      </SectionNav>
    </div>
  )
}
