import { DraftControls } from '@/components/DraftControls'
import { DdnsTab } from '@/components/dhcp-config/DdnsTab'
import { HaTab } from '@/components/dhcp-config/HaTab'
import { ReservationsTab } from '@/components/dhcp-config/ReservationsTab'
import { DhcpConfigData, DhcpTab, IfaceData, SubnetList } from '@/components/dhcp-config/shared'
import { cidrNetwork } from '@/lib/cidr'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { NoticeBanner, Spinner, Switch } from 'cheval-ui'
import { useEffect, useState } from 'react'

type DhcpConfigShape = { section: DhcpTab; onActionChange?: (a: React.ReactNode) => void }

export function DhcpConfig({ section: tab, onActionChange }: DhcpConfigShape) {
  const { data, isLoading, reload } = useFetch<DhcpConfigData | null>(() => api.apiConfigSectionGet({ section: 'dhcp' }) as Promise<DhcpConfigData | null>)
  const { data: ifaces } = useFetch<Record<string, IfaceData>>(() => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<Record<string, IfaceData>>)
  const [cfg, setCfg] = useState<DhcpConfigData>({})
  const [initialized, setInitialized] = useState(false)
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
      <div className="flex items-center gap-3">
        <DraftControls isDirty={isDirty} saving={saving} onClick={() => save(cfg)} onCancel={handleCancel} />
        <Switch
          aria-label="DHCP Server"
          title="DHCP Server"
          checked={!!cfg.enabled}
          onCheckedChange={(v) => { setCfg({ ...cfg, enabled: v || undefined }); markDirty() }}
        />
      </div>
    )
    return () => onActionChange?.(null)
  }, [isDirty, saving, cfg, data, onActionChange, save, reset, reload, markDirty])

  if (isLoading) return <Spinner />

  const remote = !!cfg.control_agent?.url

  return (
    <div className="space-y-6">
      {tab === 'subnets' && (
        <div className="space-y-6">
          {remote && <NoticeBanner intent="info">Served by the remote control agent at {cfg.control_agent?.url}, local subnets are ignored</NoticeBanner>}
          <SubnetList v6={false} subnets={cfg.subnets4 ?? []} networks={networks} ifaceNames={ifaceNames} onChange={(s) => set('subnets4', s.length ? s : undefined)} />
          <SubnetList v6={true} subnets={cfg.subnets6 ?? []} networks={networks} ifaceNames={ifaceNames} onChange={(s) => set('subnets6', s.length ? s : undefined)} />
        </div>
      )}

      {tab === 'reservations' && <ReservationsTab cfg={cfg} onChange={update} />}

      {tab === 'ddns' && <DdnsTab cfg={cfg} onChange={update} />}

      {tab === 'ha' && <HaTab cfg={cfg} ifaceNames={ifaceNames} onChange={update} />}
    </div>
  )
}
