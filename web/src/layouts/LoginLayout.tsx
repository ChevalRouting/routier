import { Outlet, Navigate } from 'react-router-dom'
import { Suspense } from 'react'
import { isAuthenticated } from '@/lib/utils'
import Logo from '@/components/Logo'
import { LoginScreen } from 'cheval-ui'

export default function LoginLayout() {
  if (isAuthenticated()) {
    return <Navigate to="/" replace />
  }

  return (
    <LoginScreen logo={Logo} title="Routier" subtitle="Network Configuration Management">
      <Suspense fallback={null}>
        <Outlet />
      </Suspense>
    </LoginScreen>
  )
}
