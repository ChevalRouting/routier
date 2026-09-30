import { useState } from 'react'
import { useTabState } from 'cheval-ui'
import { PageHeader } from 'cheval-ui'
import { SectionNav } from 'cheval-ui'
import { WireGuardPanel } from '@/components/vpn/WireGuardPanel'

const ITEMS = [{ key: 'wireguard', label: 'WireGuard' }]

export default function Vpn() {
  const [section, setSection] = useTabState<string>('vpn', 'wireguard')
  const [action, setAction] = useState<React.ReactNode>(null)

  return (
    <div className="space-y-6">
      <PageHeader title="VPN" description="Encrypted tunnels and peers" action={action} />
      <SectionNav items={ITEMS} active={section} onChange={setSection}>
        {section === 'wireguard' && <WireGuardPanel onActionChange={setAction} />}
      </SectionNav>
    </div>
  )
}
