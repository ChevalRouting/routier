import { useState, useEffect, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/client'
import type { TypesStatsResponse as StatsResponse, TypesStatsHistoryResponse as StatsHistoryResponse, TypesFriendStatus as FriendStatus } from '@/api'
import { useFetch } from '@/lib/useFetch'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { TimeSeriesChart } from '@/components/TimeSeriesChart'
import { ApplyStreamModal } from '@/components/ApplyStreamModal'
import { fmtBitrate, fmtBytes, fmtPps, fmtUptime } from '@/lib/fmt'
import { cn } from '@/lib/utils'
import {
  Network, GitBranch, Lock, Server,
  Activity, Shield, Users, Cpu, MemoryStick,
  BarChart2, ArrowRight, RefreshCw, Handshake,
} from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'

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

const TRAFFIC_PERIODS = [
  { label: '15m', minutes: 15 },
  { label: '30m', minutes: 30 },
  { label: '1h',  minutes: 60 },
  { label: '6h',  minutes: 360 },
  { label: '24h', minutes: 1440 },
  { label: '7d',  minutes: 10080 },
]

function SkeletonBox({ className }: { className?: string }) {
  return <div className={`animate-pulse rounded-md bg-muted ${className ?? ''}`} />
}

function DashboardSkeleton() {
  return (
    <div className="space-y-5">
      <PageHeader title="Dashboard" description="Network configuration overview" />
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="rounded-xl border border-border bg-card p-5 min-h-28">
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
          <div key={i} className="rounded-xl border border-border bg-card p-5 h-28">
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
  to, title, icon: Icon, value, sub, linkable = false,
}: {
  to?: string
  title: string
  icon: React.ComponentType<{ className?: string }>
  value: React.ReactNode
  sub?: string
  linkable?: boolean
}) {
  const inner = (
    <Card className={`min-h-28 ${linkable ? 'hover:bg-muted/50 transition-colors cursor-pointer' : ''}`}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold truncate">{value}</div>
        {sub && <p className="text-xs text-muted-foreground">{sub}</p>}
      </CardContent>
    </Card>
  )
  if (to) return <Link to={to} className="block">{inner}</Link>
  return inner
}

export default function Dashboard() {
  const { data: config, isLoading, error } = useFetch<Config>(() => api.apiConfigGet() as unknown as Promise<Config>)
  const { data: stats } = useFetch<StatsResponse>(() => api.apiStatsGet())
  const { data: friendStats } = useFetch<FriendStatus[]>(() => api.apiFriendsStatusGet())
  const [history, setHistory] = useState<StatsHistoryResponse | null>(null)
  const [trafficPeriodIdx, setTrafficPeriodIdx] = useState(0)
  const [trafficHistory, setTrafficHistory] = useState<StatsHistoryResponse | null>(null)
  const [reapplyOpen, setReapplyOpen] = useState(false)

  useEffect(() => {
    const load = () => api.apiStatsHistoryGet({ minutes: 60, series: 'system' }).then(setHistory).catch(() => {})
    load()
    const id = setInterval(load, 60_000)
    return () => clearInterval(id)
  }, [])

  useEffect(() => {
    const minutes = TRAFFIC_PERIODS[trafficPeriodIdx].minutes
    const load = () => api.apiStatsHistoryGet({ minutes, series: 'total,usage' }).then(setTrafficHistory).catch(() => {})
    load()
    const id = setInterval(load, 60_000)
    return () => clearInterval(id)
  }, [trafficPeriodIdx])

  const trafficChartData = useMemo(() => {
    return (trafficHistory?.total ?? []).map((p) => ({
      ts:     p.ts,
      rx_bps: p.rx_bps,
      tx_bps: p.tx_bps,
      rx_pps: p.rx_pps,
      tx_pps: p.tx_pps,
    }))
  }, [trafficHistory])

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
  const sys = stats?.system

  const sysHistory = history?.system ?? []
  const trafficUsage = trafficHistory?.usage

  return (
    <div className="space-y-5">
      <PageHeader title="Dashboard" description="Network configuration overview" action={
        <Button variant="outline" size="sm" onClick={() => setReapplyOpen(true)} className="gap-2">
          <RefreshCw className="h-4 w-4" />Force Re-apply
        </Button>
      } />

      <ApplyStreamModal open={reapplyOpen} onClose={() => setReapplyOpen(false)} />

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard to="/hostname" title="Hostname" icon={Server} linkable
          value={config?.hostname || <span className="text-muted-foreground text-base font-normal">not set</span>}
        />
        <StatCard to="/interfaces" title="Interfaces" icon={Network} linkable
          value={ifaceCount} sub="configured"
        />
        <StatCard to="/tunnels" title="Tunnels" icon={GitBranch} linkable
          value={tunnelCount} sub="configured"
        />
        <StatCard to="/wireguard" title="WireGuard" icon={Lock} linkable
          value={wgCount} sub="interfaces"
        />
        <StatCard to="/users" title="Users" icon={Users} linkable
          value={userCount} sub="accounts"
        />
        <StatCard to="/services" title="Services" icon={Server} linkable
          value={svcCount} sub="managed"
        />
        <StatCard to="/firewall" title="Firewall Rules" icon={Shield} linkable
          value={extraRules} sub="extra rules"
        />
        {friendCount > 0 && (
          <StatCard to="/friends" title="Friends" icon={Handshake} linkable
            value={`${friendsAlive} / ${friendCount}`} sub="alive"
          />
        )}
        <StatCard to={bgpCfg ? '/routing?routing=bgp&routing.bgp=neighbors' : '/routing'} title="Routing" icon={Activity} linkable
          value={
            bgpCfg
              ? <span className="text-sm font-medium">BGP AS{bgpCfg.asn}</span>
              : <span className="text-muted-foreground text-base font-normal">no routing</span>
          }
          sub={bgpCfg ? `${bgpCfg.neighbors?.length ?? 0} neighbor(s)` : undefined}
        />
      </div>

      {sys && (
        <div className="grid gap-3 sm:grid-cols-2">
          <Card className="relative overflow-hidden">
            <div className="relative z-10 flex items-center justify-between px-5 pt-5 pb-1">
              <CardTitle className="text-sm font-medium">CPU</CardTitle>
              <Cpu className="h-4 w-4 text-muted-foreground" />
            </div>
            <div className="relative z-10 px-5 pb-7">
              <div className={`text-3xl font-bold ${sys.cpu_percent > 90 ? 'text-danger' : sys.cpu_percent > 70 ? 'text-warning' : 'text-primary'}`}>
                {sys.cpu_percent.toFixed(1)}%
              </div>
              <p className="text-xs text-muted-foreground mt-0.5">
                load {sys.load1.toFixed(2)} · {sys.processes} proc · up {fmtUptime(sys.uptime_seconds)}
              </p>
            </div>
            <div className="absolute inset-x-0 bottom-0 h-20 opacity-25 pointer-events-none">
              <TimeSeriesChart
                mini
                data={sysHistory.map((p) => ({ ts: p.ts, cpu: p.cpu_pct }))}
                series={[{ dataKey: 'cpu', label: 'CPU %', color: '#3584e4' }]}
                height={80}
              />
            </div>
          </Card>

          <Card className="relative overflow-hidden">
            <div className="relative z-10 flex items-center justify-between px-5 pt-5 pb-1">
              <CardTitle className="text-sm font-medium">Memory</CardTitle>
              <MemoryStick className="h-4 w-4 text-muted-foreground" />
            </div>
            <div className="relative z-10 px-5 pb-7">
              <div className="text-3xl font-bold text-[#9141ac]">{fmtBytes(sys.mem_used)}</div>
              <p className="text-xs text-muted-foreground mt-0.5">
                of {fmtBytes(sys.mem_total)}
                {sys.swap_total > 0 && ` · swap ${fmtBytes(sys.swap_used)}/${fmtBytes(sys.swap_total)}`}
              </p>
            </div>
            <div className="absolute inset-x-0 bottom-0 h-20 opacity-25 pointer-events-none">
              <TimeSeriesChart
                mini
                data={sysHistory.map((p) => ({
                  ts: p.ts,
                  mem: p.mem_total > 0 ? (p.mem_used / p.mem_total) * 100 : 0,
                }))}
                series={[{ dataKey: 'mem', label: 'Mem %', color: '#9141ac' }]}
                height={80}
              />
            </div>
          </Card>
        </div>
      )}

      <div className="space-y-3">
        <div className="flex items-center justify-between gap-2 flex-wrap">
          <div className="flex items-center gap-2">
            <BarChart2 className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm font-medium">Traffic</span>
          </div>
          <div className="flex items-center gap-2 ml-auto">
            <div className="flex rounded-md border border-input overflow-hidden text-xs">
              {TRAFFIC_PERIODS.map((p, i) => (
                <button
                  key={p.label}
                  onClick={() => setTrafficPeriodIdx(i)}
                  className={cn(
                    'px-2.5 py-1 transition-colors',
                    trafficPeriodIdx === i
                      ? 'bg-primary text-primary-foreground font-medium'
                      : 'bg-background text-muted-foreground hover:bg-muted',
                  )}
                >
                  {p.label}
                </button>
              ))}
            </div>
            <Link
              to="/traffic"
              className="flex items-center gap-1 text-xs text-primary hover:underline whitespace-nowrap"
            >
              View all <ArrowRight className="h-3 w-3" />
            </Link>
          </div>
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <Card>
            <CardHeader className="pb-2 pt-4 px-4">
              <CardTitle className="text-xs font-medium text-muted-foreground">Bandwidth</CardTitle>
              {trafficUsage && (
                <div className="flex gap-4 text-xs text-muted-foreground mt-0.5">
                  <span>
                    <span className="font-medium text-foreground">{fmtBytes(trafficUsage.total.rx_bytes)}</span>
                    {' '}in
                  </span>
                  <span>
                    <span className="font-medium text-foreground">{fmtBytes(trafficUsage.total.tx_bytes)}</span>
                    {' '}out
                  </span>
                </div>
              )}
            </CardHeader>
            <CardContent className="px-4 pb-4">
              <TimeSeriesChart
                data={trafficChartData}
                series={[
                  { dataKey: 'rx_bps', label: 'RX', color: '#3584e4' },
                  { dataKey: 'tx_bps', label: 'TX', color: '#26a269' },
                ]}
                yFormatter={fmtBitrate}
                height={200}
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2 pt-4 px-4">
              <CardTitle className="text-xs font-medium text-muted-foreground">Packets / second</CardTitle>
            </CardHeader>
            <CardContent className="px-4 pb-4">
              <TimeSeriesChart
                data={trafficChartData}
                series={[
                  { dataKey: 'rx_pps', label: 'RX pkt/s', color: '#3584e4' },
                  { dataKey: 'tx_pps', label: 'TX pkt/s', color: '#26a269' },
                ]}
                yFormatter={fmtPps}
                height={200}
              />
            </CardContent>
          </Card>
        </div>
      </div>

    </div>
  )
}
