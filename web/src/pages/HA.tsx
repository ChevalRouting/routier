import { HAConfig } from '@/components/ha/HAConfig'
import { HAStatusPanel } from '@/components/monitor/HAStatusPanel'
import { FeaturePage } from 'cheval-ui'

export default function HAPage() {
  return (
    <FeaturePage
      title="HA"
      description="VRRP failover, conntrackd state sync, and HA peers"
      renderStatus={<HAStatusPanel />}
      renderConfig={(setAction) => <HAConfig onActionChange={setAction} />}
    />
  )
}
