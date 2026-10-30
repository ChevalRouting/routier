import { monitorSectionPath } from '@/lib/navigation'
import { Navigate, useSearchParams } from 'react-router-dom'

const MOVED: Record<string, string> = { dhcp: '/dhcp', dns: '/dns', nics: monitorSectionPath('system') }

export default function MonitorRedirect() {
  const [params] = useSearchParams()
  const requested = params.get('tab') ?? params.get('monitor') ?? ''
  return <Navigate to={MOVED[requested] ?? monitorSectionPath(requested)} replace />
}
