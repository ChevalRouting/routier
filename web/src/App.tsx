import { liveSkipped } from '@/components/install/shared'
import { lazy } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { isAuthenticated } from '@/lib/utils'
import { monitorSectionPath } from '@/lib/navigation'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { DataVersionProvider } from '@/lib/dataVersion'
import { Spinner } from 'cheval-ui'
import { useActiveInstance } from '@/components/InstanceSwitcher'
import LoginLayout from '@/layouts/LoginLayout'
import MainLayout from '@/layouts/MainLayout'

type RequireAuthShape = { children: React.ReactNode }

type RequireSetupShape = { children: React.ReactNode }

const Login = lazy(() => import('@/pages/Login'))
const MobileMenu = lazy(() => import('@/pages/MobileMenu'))
const Dashboard = lazy(() => import('@/pages/Dashboard'))
const Interfaces = lazy(() => import('@/pages/Interfaces'))
const Switches = lazy(() => import('@/pages/Switches'))
const SwitchDetail = lazy(() => import('@/pages/SwitchDetail'))
const Routing = lazy(() => import('@/pages/Routing'))
const Vpn = lazy(() => import('@/pages/Vpn'))
const HA = lazy(() => import('@/pages/HA'))
const Firewall = lazy(() => import('@/pages/Firewall'))
const UserDetail = lazy(() => import('@/pages/UserDetail'))
const Settings = lazy(() => import('@/pages/Settings'))
const ConfigBrowser = lazy(() => import('@/pages/ConfigBrowser'))
const Anycast = lazy(() => import('@/pages/Anycast'))
const Dhcp = lazy(() => import('@/pages/Dhcp'))
const Dns = lazy(() => import('@/pages/Dns'))
const Certificates = lazy(() => import('@/pages/Certificates'))
const Friends = lazy(() => import('@/pages/Friends'))
const Debug = lazy(() => import('@/pages/Debug'))
const Monitor = lazy(() => import('@/pages/Monitor'))
const MonitorRedirect = lazy(() => import('@/pages/MonitorRedirect'))
const SystemConfig = lazy(() => import('@/pages/SystemConfig'))
const Macros = lazy(() => import('@/pages/Macros'))
const IpTools = lazy(() => import('@/pages/IpTools'))
const Announcements = lazy(() => import('@/pages/Announcements'))
const Onboarding = lazy(() => import('@/pages/Onboarding'))
const Install = lazy(() => import('@/pages/Install'))

function RequireAuth({ children }: RequireAuthShape) {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function RequireSetup({ children }: RequireSetupShape) {
  const active = useActiveInstance()
  const { data, hasData, isLoading, error } = useFetch(() => api.apiSetupStatusGet())
  const { data: install, isLoading: installLoading } = useFetch(() => api.apiInstallStatusGet())
  if (active.id !== 'self') return <>{children}</>
  if (isLoading && !hasData) return <Spinner />
  if (installLoading) return <Spinner />
  if (error && !isAuthenticated()) return <Spinner />
  if (install?.live && (install.installer_only || install.state !== 'idle' || !liveSkipped())) {
    return <Navigate to="/install" replace />
  }
  if (data && (data.needs_password_change || !data.onboarding_complete)) {
    return <Navigate to="/onboarding" replace />
  }
  return <>{children}</>
}

export default function App() {
  return (
    <DataVersionProvider>
    <Routes>
        <Route element={<LoginLayout />}>
          <Route path="/login" element={<Login />} />
        </Route>
        <Route path="/onboarding" element={<RequireAuth><Onboarding /></RequireAuth>} />
        <Route path="/install" element={<RequireAuth><Install /></RequireAuth>} />
        <Route
          element={
            <RequireAuth>
              <RequireSetup>
                <MainLayout />
              </RequireSetup>
            </RequireAuth>
          }
        >
          <Route path="/" element={<Dashboard />} />

          <Route path="/interfaces" element={<Interfaces />} />
          <Route path="/switches" element={<Switches />} />
          <Route path="/switches/:name" element={<SwitchDetail />} />
          <Route path="/routing" element={<Routing />} />
          <Route path="/vpn" element={<Vpn />} />
          <Route path="/anycast" element={<Anycast />} />
          <Route path="/dhcp" element={<Dhcp />} />
          <Route path="/dns" element={<Dns />} />
          <Route path="/ha" element={<HA />} />
          <Route path="/firewall" element={<Firewall />} />

          <Route path="/system" element={<SystemConfig />} />
          <Route path="/certificates" element={<Certificates />} />
          <Route path="/hostname" element={<Navigate to="/system" replace />} />
          <Route path="/dns"      element={<Navigate to="/system" replace />} />
          <Route path="/ssh"      element={<Navigate to="/system?system-config=ssh" replace />} />
          <Route path="/sysctl"   element={<Navigate to="/system" replace />} />
          <Route path="/users" element={<Navigate to="/system?system-config=users" replace />} />
          <Route path="/system/users/:name" element={<UserDetail />} />
          <Route path="/users/:name" element={<UserDetail />} />
          <Route path="/services" element={<Navigate to="/system?system-config=services" replace />} />
          <Route path="/friends"  element={<Friends />} />

          <Route path="/monitor"         element={<MonitorRedirect />} />
          <Route path="/monitor/system"  element={<Monitor key="system" group="System" />} />
          <Route path="/monitor/network" element={<Monitor key="network" group="Network" />} />
          <Route path="/traffic"    element={<Navigate to={monitorSectionPath('traffic')} replace />} />
          <Route path="/logs"       element={<Navigate to={monitorSectionPath('logs')} replace />} />
          <Route path="/neighbors"  element={<Navigate to={monitorSectionPath('neighbors')} replace />} />
          <Route path="/log-settings" element={<Navigate to={monitorSectionPath('logs')} replace />} />

          <Route path="/topology"       element={<Navigate to={monitorSectionPath('topology')} replace />} />
          <Route path="/macros"         element={<Macros />} />
          <Route path="/ip-tools"       element={<IpTools />} />
          <Route path="/announcements"  element={<Announcements />} />
          <Route path="/config-browser" element={<ConfigBrowser />} />
          <Route path="/debug"          element={<Debug />} />
          <Route path="/settings"       element={<Settings />} />
          <Route path="/menu/:group"    element={<MobileMenu />} />

          <Route path="/routing/edit" element={<Navigate to="/routing" replace />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </DataVersionProvider>
  )
}
