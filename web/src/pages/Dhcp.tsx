import { FeaturePage } from 'cheval-ui'
import { DhcpPanel } from '@/components/monitor/DhcpPanel'
import { DhcpConfig } from '@/pages/DhcpConfig'

export default function DhcpPage() {
  return (
    <FeaturePage
      title="DHCP"
      renderStatus={<DhcpPanel />}
      renderConfig={(setAction) => <DhcpConfig onActionChange={setAction} />}
    />
  )
}
