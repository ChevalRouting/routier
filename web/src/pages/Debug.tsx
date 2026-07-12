import { TerminalPane } from '@/components/TerminalPane'
import { PageHeader } from '@/components/PageHeader'

export default function Debug() {
  return (
    <div className="flex flex-col h-full space-y-4">
      <PageHeader title="Debug" description="Interactive diagnostics, ping, mtr, traceroute, ip, ss, dig, curl, …" />
      <div className="flex-1 rounded-lg overflow-hidden border border-border min-h-0" style={{ minHeight: 400 }}>
        <TerminalPane wsPath="/api/ws/exec?name=debug" className="h-full" />
      </div>
    </div>
  )
}
