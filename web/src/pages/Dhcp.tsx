import { useHasParam } from '@/lib/useHasParam'
import { DhcpConfig } from '@/components/dhcp-config/DhcpConfig'
import type { DhcpTab } from '@/components/dhcp-config/shared'
import { DhcpLogsPanel, DhcpStatusPanel } from '@/components/monitor/DhcpPanel'
import { cn, PageHeader, SectionNav, useTabState } from 'cheval-ui'
import { useState } from 'react'

type DhcpSection = 'leases' | 'logs' | DhcpTab

interface DhcpSectionItem {
  key: DhcpSection
  label: string
  group: string
}

const SECTIONS: DhcpSectionItem[] = [
  { key: 'leases', label: 'Leases', group: 'Status' },
  { key: 'logs', label: 'Logs', group: 'Status' },
  { key: 'subnets', label: 'Subnets', group: 'Configuration' },
  { key: 'reservations', label: 'Reservations', group: 'Configuration' },
  { key: 'ddns', label: 'Dynamic DNS', group: 'Configuration' },
  { key: 'ha', label: 'High Availability', group: 'Configuration' },
]

export default function DhcpPage() {
  const [section, setSection] = useTabState<DhcpSection>('dhcp', 'leases')
  const deepLinked = useHasParam('dhcp')
  const [action, setAction] = useState<React.ReactNode>(null)
  const configSection = (['subnets', 'reservations', 'ddns', 'ha'] as string[]).includes(section) ? section as DhcpTab : null

  return (
    <div className="space-y-6">
      <PageHeader title="DHCP" action={action} />
      <SectionNav items={SECTIONS} active={section} onChange={setSection} initiallyEntered={deepLinked}>
        {section === 'leases' && <DhcpStatusPanel />}
        {section === 'logs' && <DhcpLogsPanel />}
        <div className={cn('max-w-3xl', !configSection && 'hidden')}>
          <DhcpConfig section={configSection ?? 'subnets'} onActionChange={setAction} />
        </div>
      </SectionNav>
    </div>
  )
}
