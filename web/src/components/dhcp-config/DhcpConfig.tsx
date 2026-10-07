import { DdnsTab } from '@/components/dhcp-config/DdnsTab'
import { ReservationsTab } from '@/components/dhcp-config/ReservationsTab'
import { DhcpConfigData, DhcpTab, IfaceData, SubnetList } from '@/components/dhcp-config/shared'
import { cidrNetwork } from '@/lib/cidr'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { SaveButton, Spinner, Switch, Tabs, useTabState } from 'cheval-ui'
import { useEffect, useState } from 'react'

type DhcpConfigShape = { onActionChange?: (a: React.ReactNode) => void }

export function DhcpConfig({ onActionChange }: DhcpConfigShape) {
  const { data, isLoading, reload } = useFetch<DhcpConfigData | null>(() => api.apiConfigSectionGet({ section: 'dhcp' }) as Promise<DhcpConfigData | null>)
  const { data: ifaces } = useFetch<Record<string, IfaceData>>(() => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<Record<string, IfaceData>>)
  const [cfg, setCfg] = useState<DhcpConfigData>({})
  const [initialized, setInitialized] = useState(false)
  const [tab, setTab] = useTabState<DhcpTab>('dhcp.config', 'subnets')
  const { isDirty, markDirty, save, saving, reset } = usePageSave('dhcp')
  useDataRefresh(() => setInitialized(false))

  const ifaceNames = Object.keys(ifaces ?? {})

  const networks = Array.from(new Set(
    Object.values(ifaces ?? {}).flatMap((i) => i?.addresses ?? [])
      .map(cidrNetwork)
      .filter((n): n is string => n !== null),
  ))

  useEffect(() => {
    if (!isLoading && !initialized) {
      setCfg(data ?? {})
      setInitialized(true)
    }
  }, [isLoading, initialized, data])

  const update = (next: DhcpConfigData) => { setCfg(next); markDirty() }
  const set = <K extends keyof DhcpConfigData>(k: K, v: DhcpConfigData[K]) => update({ ...cfg, [k]: v })

  useEffect(() => {
    const handleCancel = () => { setInitialized(false); reset(); reload(true) }

    onActionChange?.(
      <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(cfg)} onCancel={handleCancel} />
    )
    return () => onActionChange?.(null)
  }, [isDirty, saving, cfg, data, onActionChange, save, reset, reload])

  if (isLoading) return <Spinner />

  const remote = !!cfg.control_agent?.url

  return (
    <div className="space-y-5">
      <label className="flex items-center gap-3">
        <Switch checked={!!cfg.enabled} onCheckedChange={(v) => set('enabled', v || undefined)} />
        <div>
          <div className="text-sm font-medium">Enable DHCP server</div>
          <div className="text-xs text-muted-foreground">Render and run a local Kea DHCP server from this config.</div>
        </div>
      </label>

      {remote && (
        <p className="rounded-md bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
          A remote control agent ({cfg.control_agent?.url}) is configured, so no local Kea is rendered. The subnets below are ignored for a remote server.
        </p>
      )}

      <Tabs
        active={tab}
        onChange={setTab}
        tabs={[
          { key: 'subnets', label: 'Subnets' },
          { key: 'reservations', label: 'Reservations' },
          { key: 'ddns', label: 'Dynamic DNS' },
        ]}
      />

      {tab === 'subnets' && (
        <div className="space-y-5">
          <SubnetList v6={false} subnets={cfg.subnets4 ?? []} networks={networks} ifaceNames={ifaceNames} onChange={(s) => set('subnets4', s.length ? s : undefined)} />
          <SubnetList v6={true} subnets={cfg.subnets6 ?? []} networks={networks} ifaceNames={ifaceNames} onChange={(s) => set('subnets6', s.length ? s : undefined)} />
        </div>
      )}

      {tab === 'reservations' && <ReservationsTab cfg={cfg} onChange={update} />}

      {tab === 'ddns' && <DdnsTab cfg={cfg} onChange={update} />}
    </div>
  )
}
