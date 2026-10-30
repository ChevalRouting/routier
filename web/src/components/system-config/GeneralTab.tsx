import { DraftControls } from '@/components/DraftControls'
import { DNSTab } from '@/components/system-config/DNSTab'
import { SSHTab } from '@/components/system-config/SSHTab'
import { EMPTY_SAVE, HostnameTab, SystemTab, TabSaveState } from '@/components/system-config/shared'
import { ChevronDown } from 'lucide-react'
import { ReactNode, useState } from 'react'

type GeneralTabShape = { initialSection: SystemTab }
type GeneralSectionShape = { title: string; description: string; initiallyOpen?: boolean; children: ReactNode }

function GeneralSection({ title, description, initiallyOpen = false, children }: GeneralSectionShape) {
  return (
    <details open={initiallyOpen} className="group rounded-xl border border-border bg-card">
      <summary className="flex min-h-14 cursor-pointer list-none items-center gap-3 rounded-xl px-4 py-3 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring [&::-webkit-details-marker]:hidden">
        <div className="min-w-0 flex-1">
          <h2 className="text-sm font-semibold">{title}</h2>
          <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
        </div>
        <ChevronDown aria-hidden="true" className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-open:rotate-180" />
      </summary>
      <div className="border-t border-border p-4">{children}</div>
    </details>
  )
}

export function GeneralTab({ initialSection }: GeneralTabShape) {
  const [hostname, setHostname] = useState<TabSaveState>(EMPTY_SAVE)
  const [dns, setDNS] = useState<TabSaveState>(EMPTY_SAVE)
  const [ssh, setSSH] = useState<TabSaveState>(EMPTY_SAVE)

  return (
    <div className="max-w-3xl space-y-4">
      <DraftControls isDirty={hostname.isDirty} saving={hostname.saving} onClick={hostname.save} />
      <DraftControls isDirty={dns.isDirty} saving={dns.saving} onClick={dns.save} />
      <DraftControls isDirty={ssh.isDirty} saving={ssh.saving} onClick={ssh.save} />
      <GeneralSection title="Hostname" description="The name of this router" initiallyOpen={initialSection !== 'dns' && initialSection !== 'ssh'}>
        <HostnameTab onStateChange={setHostname} />
      </GeneralSection>
      <GeneralSection title="DNS" description="Nameservers and search domains for this router" initiallyOpen={initialSection === 'dns'}>
        <DNSTab onStateChange={setDNS} />
      </GeneralSection>
      <GeneralSection title="SSH" description="Remote login, authentication and forwarding" initiallyOpen={initialSection === 'ssh'}>
        <SSHTab onStateChange={setSSH} />
      </GeneralSection>
    </div>
  )
}
