import { ApiKeysTab } from '@/components/system-config/ApiKeysTab'
import { ApplyHistoryTab } from '@/components/system-config/ApplyHistoryTab'
import { BackupTab } from '@/components/system-config/BackupTab'
import { DNSTab } from '@/components/system-config/DNSTab'
import { MonitoringTab } from '@/components/system-config/MonitoringTab'
import { ApiDocsTab, CONFIG_TABS, EMPTY_SAVE, HostnameTab, SYSTEM_TABS, SystemTab, TabSaveState } from '@/components/system-config/shared'
import { SnapshotsTab } from '@/components/system-config/SnapshotsTab'
import { SSHTab } from '@/components/system-config/SSHTab'
import { SysctlTab } from '@/components/system-config/SysctlTab'
import UpdatesPanel from '@/components/system/UpdatesPanel'
import { useDataVersion } from '@/lib/dataVersion'
import ConfigUsers from '@/pages/ConfigUsers'
import Services from '@/pages/Services'
import { PageHeader, SaveButton, SectionNav, useTabState } from 'cheval-ui'
import { useState } from 'react'

export default function SystemConfig() {
  const [activeTab, setActiveTab] = useTabState<SystemTab>('system-config', 'hostname')
  const [saveState, setSaveState] = useState<TabSaveState>(EMPTY_SAVE)
  const { bump } = useDataVersion()

  const handleTabChange = (tab: SystemTab) => {
    setActiveTab(tab)
    setSaveState(EMPTY_SAVE)
  }

  const isConfigTab = CONFIG_TABS.includes(activeTab)

  return (
    <div className="space-y-6">
      <PageHeader
        title="General"
        description="OS hostname, DNS, SSH daemon, users, services, kernel parameters, monitoring, apply history and backups"
        action={isConfigTab ? <SaveButton isDirty={saveState.isDirty} saving={saveState.saving} onClick={saveState.save} /> : undefined}
      />
      <SectionNav items={SYSTEM_TABS} active={activeTab} onChange={handleTabChange}>
        {activeTab === 'hostname'  && <HostnameTab onStateChange={setSaveState} />}
        {activeTab === 'dns'       && <DNSTab onStateChange={setSaveState} />}
        {activeTab === 'ssh'       && <SSHTab onStateChange={setSaveState} />}
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
