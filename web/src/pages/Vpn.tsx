import { WireGuardPanel } from '@/components/vpn/WireGuardPanel'
import { PageHeader } from 'cheval-ui'
import { useState } from 'react'

export default function Vpn() {
  const [action, setAction] = useState<React.ReactNode>(null)

  return (
    <div className="space-y-6">
      <PageHeader title="VPN" action={action} />
      <WireGuardPanel onActionChange={setAction} />
    </div>
  )
}
