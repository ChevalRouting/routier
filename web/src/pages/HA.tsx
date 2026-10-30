import { useHasParam } from '@/lib/useHasParam'
import { HAConfig, type HASection } from '@/components/ha/HAConfig'
import { HAStatusPanel } from '@/components/monitor/HAStatusPanel'
import { cn, PageHeader, SectionNav, useTabState } from 'cheval-ui'
import { useState } from 'react'

type HAPageSection = 'status' | HASection

interface HASectionItem {
  key: HAPageSection
  label: string
  group: string
}

const SECTIONS: HASectionItem[] = [
  { key: 'status', label: 'Status', group: 'Status' },
  { key: 'vrrp', label: 'VRRP', group: 'Configuration' },
  { key: 'conntrackd', label: 'Conntrackd', group: 'Configuration' },
]

export default function HAPage() {
  const [section, setSection] = useTabState<HAPageSection>('ha', 'status')
  const deepLinked = useHasParam('ha')
  const [action, setAction] = useState<React.ReactNode>(null)
  const configSection = section === 'status' ? null : section

  return (
    <div className="space-y-6">
      <PageHeader title="HA" action={action} />
      <SectionNav items={SECTIONS} active={section} onChange={setSection} initiallyEntered={deepLinked}>
        {section === 'status' && <HAStatusPanel />}
        <div className={cn('max-w-3xl', !configSection && 'hidden')}>
          <HAConfig section={configSection ?? 'vrrp'} onActionChange={setAction} />
        </div>
      </SectionNav>
    </div>
  )
}
