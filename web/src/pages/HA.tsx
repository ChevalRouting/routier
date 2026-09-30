import { FeaturePage } from 'cheval-ui'
import { HAStatusPanel } from '@/components/monitor/HAStatusPanel'
import { HAConfig } from '@/components/ha/HAConfig'

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
