import { useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/client'
import type { TypesFriendStatus as FriendStatus } from '@/api'
import { useFetch } from '@/lib/useFetch'
import { Card, CardContent, CardHeader, CardTitle } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { cn } from 'cheval-ui'
import { ApplyStreamModal } from '@/components/ApplyStreamModal'
import { PowerControls } from '@/components/system/PowerControls'
import { SystemOverview } from '@/components/SystemOverview'
import { ProbePanel } from '@/components/ProbePanel'
import {
  Network, GitBranch, Lock, Server,
  Activity, Shield, Users,
  RefreshCw, Handshake,
} from 'lucide-react'
import { PageHeader } from 'cheval-ui'

interface Config {
  hostname?: string
  interfaces?: Record<string, unknown>
  tunnels?: Record<string, unknown>
  wireguard?: Record<string, unknown>
  routing?: { bgp?: { asn?: number; neighbors?: unknown[] } }
  nftables?: unknown[]
  users?: Record<string, unknown>
  services?: Record<string, unknown>
  friends?: unknown[]
}

function SkeletonBox({ className }: { className?: string }) {
  return <div className={`animate-pulse rounded-md bg-muted ${className ?? ''}`} />
}

function DashboardSkeleton() {
  return (
    <div className="space-y-5">
      <PageHeader title="Dashboard" description="Network configuration overview" />
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="rounded-xl bg-card p-5 min-h-28 shadow-[var(--card-shadow)]">
            <div className="flex items-center justify-between mb-3">
              <SkeletonBox className="h-3 w-20" />
              <SkeletonBox className="h-4 w-4" />
            </div>
            <SkeletonBox className="h-7 w-14 mb-1.5" />
            <SkeletonBox className="h-3 w-12" />
          </div>
        ))}
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        {[0, 1].map((i) => (
          <div key={i} className="rounded-xl bg-card p-5 h-28 shadow-[var(--card-shadow)]">
            <SkeletonBox className="h-3 w-8 mb-3" />
            <SkeletonBox className="h-8 w-20 mb-1.5" />
            <SkeletonBox className="h-3 w-32" />
          </div>
        ))}
      </div>
    </div>
  )
}

function StatCard({
  to, title, icon: Icon, value, sub, linkable = false, className,
}: {
  to?: string
  title: string
  icon: React.ComponentType<{ className?: string }>
  value: React.ReactNode
  sub?: string
  linkable?: boolean
  className?: string
}) {
  const inner = (
    <Card className={`min-h-28 ${linkable ? 'hover:bg-muted/50 transition-colors cursor-pointer' : ''}`}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="flex items-center min-h-8 text-2xl font-bold truncate">{value}</div>
        <p className="text-xs text-muted-foreground min-h-4">{sub}</p>
      </CardContent>
    </Card>
  )
  if (to) return <Link to={to} className={cn('block', className)}>{inner}</Link>
  return <div className={className}>{inner}</div>
}

export default function Dashboard() {
  const { data: config, isLoading, error } = useFetch<Config>(() => api.apiConfigGet() as unknown as Promise<Config>)
  const { data: friendStats } = useFetch<FriendStatus[]>(() => api.apiFriendsStatusGet())
  const [reapplyOpen, setReapplyOpen] = useState(false)

  if (isLoading) return <DashboardSkeleton />
  if (error) return <div className="text-sm text-danger">Failed to load: {error}</div>

  const ifaceCount = Object.keys(config?.interfaces ?? {}).length
  const tunnelCount = Object.keys(config?.tunnels ?? {}).length
  const wgCount = Object.keys(config?.wireguard ?? {}).length
  const userCount = Object.keys(config?.users ?? {}).length
  const svcCount = Object.keys(config?.services ?? {}).length
  const extraRules = config?.nftables?.length ?? 0
  const friendCount = config?.friends?.length ?? 0
  const friendsAlive = (friendStats ?? []).filter((f) => f.reachable).length
  const bgpCfg = config?.routing?.bgp

  const powerActions = (
    <>
      <Button variant="outline" size="sm" onClick={() => setReapplyOpen(true)} className="gap-2">
        <RefreshCw className="h-4 w-4" />Force Re-apply
      </Button>
      <PowerControls />
    </>
  )

  return (
    <div className="flex flex-col gap-5">
      <div className="hidden md:block">
        <PageHeader title="Dashboard" description="Network configuration overview" action={
          <div className="flex items-center gap-2">{powerActions}</div>
        } />
      </div>
      <div className="md:hidden space-y-3">
        <PageHeader title="Dashboard" description="Network configuration overview" />
        <div className="flex flex-wrap items-center gap-2">{powerActions}</div>
      </div>

      <ApplyStreamModal open={reapplyOpen} onClose={() => setReapplyOpen(false)} />

      <div className="order-2 md:order-none grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard to="/hostname" title="Hostname" icon={Server} linkable
          value={config?.hostname || <span className="text-muted-foreground text-base font-normal">not set</span>}
        />
        <StatCard to="/interfaces" title="Interfaces" icon={Network} linkable className="hidden md:block"
          value={ifaceCount} sub="configured"
        />
        <StatCard to="/interfaces" title="Tunnels" icon={GitBranch} linkable className="hidden md:block"
          value={tunnelCount} sub="configured"
        />
        <StatCard to="/vpn" title="WireGuard" icon={Lock} linkable className="hidden md:block"
          value={wgCount} sub="interfaces"
        />
        <StatCard to="/users" title="Users" icon={Users} linkable className="hidden md:block"
          value={userCount} sub="accounts"
        />
        <StatCard to="/services" title="Services" icon={Server} linkable className="hidden md:block"
          value={svcCount} sub="managed"
        />
        <StatCard to="/firewall" title="Firewall Rules" icon={Shield} linkable className="hidden md:block"
          value={extraRules} sub="extra rules"
        />
        {friendCount > 0 && (
          <StatCard to="/friends" title="Friends" icon={Handshake} linkable className="hidden md:block"
            value={`${friendsAlive} / ${friendCount}`} sub="alive"
          />
        )}
        <StatCard to={bgpCfg ? '/routing?routing=bgp&routing.bgp=neighbors' : '/routing'} title="Routing" icon={Activity} linkable className="hidden md:block"
          value={
            bgpCfg
              ? <span className="text-sm font-medium">BGP AS{bgpCfg.asn}</span>
              : <span className="text-muted-foreground text-base font-normal">no routing</span>
          }
          sub={bgpCfg ? `${bgpCfg.neighbors?.length ?? 0} neighbor(s)` : undefined}
        />
      </div>

      <div className="order-1 md:order-none">
        <SystemOverview trafficHref="/traffic" />
      </div>
      <div className="order-3 md:order-none">
        <ProbePanel />
      </div>
    </div>
  )
}
