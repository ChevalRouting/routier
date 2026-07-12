import { useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { DhcpConfig } from '@/pages/DhcpConfig'

export default function DhcpPage() {
  const [action, setAction] = useState<React.ReactNode>(null)

  return (
    <div className="space-y-6">
      <PageHeader
        title="DHCP"
        action={action}
      />
      <DhcpConfig onActionChange={setAction} />
    </div>
  )
}
