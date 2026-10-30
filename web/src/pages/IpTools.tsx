import { RangeTool } from '@/components/ip-tools/RangeTool'
import { ReverseTool } from '@/components/ip-tools/ReverseTool'
import { Tab, TABS } from '@/components/ip-tools/shared'
import { SubnetTool } from '@/components/ip-tools/SubnetTool'
import { PageHeader, Tabs } from 'cheval-ui'
import { Calculator } from 'lucide-react'
import { useState } from 'react'

export default function IpTools() {
  const [tab, setTab] = useState<Tab>('subnet')

  return (
    <div className="space-y-6">
      <PageHeader title="IP Tools" />
      <div className="flex items-center gap-2 text-muted-foreground">
        <Calculator className="h-4 w-4" />
        <Tabs tabs={TABS} active={tab} onChange={setTab} />
      </div>
      <div className="max-w-3xl">
        {tab === 'subnet' && <SubnetTool />}
        {tab === 'reverse' && <ReverseTool />}
        {tab === 'range' && <RangeTool />}
      </div>
    </div>
  )
}
