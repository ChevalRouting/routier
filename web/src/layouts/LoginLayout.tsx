import { Outlet, Navigate } from 'react-router-dom'
import { Suspense } from 'react'
import { isAuthenticated } from '@/lib/utils'
import Logo from '@/components/Logo'

export default function LoginLayout() {
  if (isAuthenticated()) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="flex flex-col items-center mb-8">
          <Logo className="h-20 w-20 mb-3" />
          <span className="text-3xl font-bold text-foreground">Routier</span>
          <p className="text-muted-foreground text-sm mt-1">Network Configuration Management</p>
        </div>
        <Suspense fallback={null}>
          <Outlet />
        </Suspense>
      </div>
    </div>
  )
}
