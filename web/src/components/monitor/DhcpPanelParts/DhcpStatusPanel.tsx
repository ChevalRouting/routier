import type { DhcpStatsResponse as DhcpStats } from '@/api'
import { DhcpLeases, DhcpStatGrid } from '@/components/monitor/DhcpPanel'
import { api, configLayerRequest } from '@/lib/client'
import { Button, StateChip } from 'cheval-ui'
import { RotateCw } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'

export function DhcpStatusPanel() {
  const [stats, setStats] = useState<DhcpStats | null>(null)
  const [restarting, setRestarting] = useState(false)

  const load = useCallback(() => api.apiDhcpStatsGet().then(setStats).catch(() => {}), [])

  useEffect(() => {
    let alive = true
    const tick = () => api.apiDhcpStatsGet().then((s) => { if (alive) setStats(s) }).catch(() => {})
    tick()
    const id = setInterval(tick, 10_000)
    return () => { alive = false; clearInterval(id) }
  }, [])

  const running = (stats?.services ?? []).some((s) => s.running)

  const restart = async () => {
    setRestarting(true)
    try {
      await configLayerRequest('/api/dhcp/restart', { method: 'POST' })
      toast.success('DHCP server restarted')
      await load()
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setRestarting(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        {(stats?.services ?? []).map((s) => (
          <div key={s.service} className="flex items-center gap-1.5">
            <span className="text-xs font-mono text-muted-foreground">{s.service}</span>
            <StateChip state={s.running ? 'RUNNING' : 'STOPPED'} />
          </div>
        ))}
        <Button variant="outline" size="sm" className="gap-1.5" disabled={restarting || !running} onClick={() => void restart()}>
          <RotateCw className={`h-3.5 w-3.5 ${restarting ? 'animate-spin' : ''}`} /> Restart
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <DhcpStatGrid title="IPv4 statistics" stats={stats?.stats4 ?? null} />
        <DhcpStatGrid title="IPv6 statistics" stats={stats?.stats6 ?? null} />
      </div>

      <DhcpLeases />
    </div>
  )
}
