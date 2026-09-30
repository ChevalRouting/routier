import { TerminalPane } from 'cheval-ui/terminal'
import { PageHeader } from 'cheval-ui'
import { getToken } from '@/lib/utils'

export default function Debug() {
  return (
    <div className="flex flex-col h-full space-y-4">
      <PageHeader title="Debug" description="Interactive diagnostics, ping, mtr, traceroute, ip, ss, dig, curl, …" />
      <div className="flex-1 rounded-lg overflow-hidden shadow-[var(--card-shadow)] min-h-0" style={{ minHeight: 400 }}>
        <TerminalPane wsPath="/api/ws/exec?name=debug" token={getToken() ?? undefined} className="h-full" />
      </div>
    </div>
  )
}
