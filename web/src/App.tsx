import { lazy } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { isAuthenticated } from '@/lib/utils'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { DataVersionProvider } from '@/lib/dataVersion'
import { Spinner } from '@/components/Spinner'
import LoginLayout from '@/layouts/LoginLayout'
import MainLayout from '@/layouts/MainLayout'

const Login = lazy(() => import('@/pages/Login'))
const Dashboard = lazy(() => import('@/pages/Dashboard'))
const Interfaces = lazy(() => import('@/pages/Interfaces'))
const Tunnels = lazy(() => import('@/pages/Tunnels'))
const RoutingOverview = lazy(() => import('@/pages/RoutingOverview'))
const Routing = lazy(() => import('@/pages/Routing'))
const WireGuard = lazy(() => import('@/pages/WireGuard'))
const Conntrackd = lazy(() => import('@/pages/Conntrackd'))
const Firewall = lazy(() => import('@/pages/Firewall'))
const ConfigUsers = lazy(() => import('@/pages/ConfigUsers'))
const Services = lazy(() => import('@/pages/Services'))
const Settings = lazy(() => import('@/pages/Settings'))
const ConfigBrowser = lazy(() => import('@/pages/ConfigBrowser'))
const Anycast = lazy(() => import('@/pages/Anycast'))
const Dhcp = lazy(() => import('@/pages/Dhcp'))
const Friends = lazy(() => import('@/pages/Friends'))
const FriendDetail = lazy(() => import('@/pages/FriendDetail'))
const Debug = lazy(() => import('@/pages/Debug'))
const Monitor = lazy(() => import('@/pages/Monitor'))
const SystemConfig = lazy(() => import('@/pages/SystemConfig'))
const Macros = lazy(() => import('@/pages/Macros'))
const IpTools = lazy(() => import('@/pages/IpTools'))
const Announcements = lazy(() => import('@/pages/Announcements'))
const Onboarding = lazy(() => import('@/pages/Onboarding'))

function RequireAuth({ children }: { children: React.ReactNode }) {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function RequireSetup({ children }: { children: React.ReactNode }) {
  const { data, isLoading, error } = useFetch(() => api.apiSetupStatusGet())
  if (isLoading) return <Spinner />
  if (error && !isAuthenticated()) return <Spinner />
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
          <Route path="/tunnels" element={<Tunnels />} />
          <Route path="/routing" element={<Routing />} />
          <Route path="/wireguard" element={<WireGuard />} />
          <Route path="/anycast" element={<Anycast />} />
          <Route path="/dhcp" element={<Dhcp />} />
          <Route path="/firewall" element={<Firewall />} />

          <Route path="/system" element={<SystemConfig />} />
          <Route path="/hostname" element={<Navigate to="/system" replace />} />
          <Route path="/dns"      element={<Navigate to="/system" replace />} />
          <Route path="/ssh"      element={<Navigate to="/system" replace />} />
          <Route path="/sysctl"   element={<Navigate to="/system" replace />} />
          <Route path="/users"    element={<ConfigUsers />} />
          <Route path="/services" element={<Services />} />
          <Route path="/conntrackd" element={<Conntrackd />} />
          <Route path="/friends"  element={<Friends />} />
          <Route path="/friends/:name" element={<FriendDetail />} />

          <Route path="/monitor"    element={<Monitor />} />
          <Route path="/traffic"    element={<Navigate to="/monitor?tab=traffic" replace />} />
          <Route path="/logs"       element={<Navigate to="/monitor?tab=logs" replace />} />
          <Route path="/ha-status"  element={<Navigate to="/monitor?tab=ha-status" replace />} />
          <Route path="/neighbors"  element={<Navigate to="/monitor?tab=neighbors" replace />} />
          <Route path="/log-settings" element={<Navigate to="/monitor?tab=logs" replace />} />

          <Route path="/topology"       element={<RoutingOverview />} />
          <Route path="/macros"         element={<Macros />} />
          <Route path="/ip-tools"       element={<IpTools />} />
          <Route path="/announcements"  element={<Announcements />} />
          <Route path="/config-browser" element={<ConfigBrowser />} />
          <Route path="/debug"          element={<Debug />} />
          <Route path="/settings"       element={<Settings />} />

          <Route path="/routing/edit" element={<Navigate to="/routing" replace />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </DataVersionProvider>
  )
}
