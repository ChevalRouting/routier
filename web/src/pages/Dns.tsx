import { useEffect, useState } from 'react'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { useTabState } from 'cheval-ui'
import { SaveButton } from 'cheval-ui'
import { Spinner } from 'cheval-ui'
import { PageHeader } from 'cheval-ui'
import { SectionNav } from 'cheval-ui'
import { DnsPanel } from '@/components/monitor/DnsPanel'
import { DnsQueriesPanel } from '@/components/monitor/DnsQueriesPanel'
import { DdnsPanel } from '@/components/monitor/DdnsPanel'
import {
  ResolverSettings, CacheSettings, ZoneEditor,
  type DnsServerData, type IfaceData,
} from '@/pages/DnsConfig'

const ADD_ZONE = '__add_zone__'

export default function DnsPage() {
  const { data, isLoading } = useFetch<DnsServerData | null>(() => api.apiConfigSectionGet({ section: 'dns_server' }) as Promise<DnsServerData | null>)
  const { data: ifaces } = useFetch<Record<string, IfaceData>>(() => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<Record<string, IfaceData>>)
  const { data: ha } = useFetch<{ vrrp?: { interface: string; vips?: string[] }[] }>(() => api.apiConfigSectionGet({ section: 'ha' }) as Promise<{ vrrp?: { interface: string; vips?: string[] }[] }>)
  const [cfg, setCfg] = useState<DnsServerData>({})
  const [initialized, setInitialized] = useState(false)
  const [section, setSection] = useTabState<string>('dns', 'overview')
  const { isDirty, markDirty, save, saving, reset } = usePageSave('dns_server')

  useEffect(() => {
    if (!isLoading && !initialized) {
      setCfg(data ?? {})
      setInitialized(true)
    }
  }, [isLoading, initialized, data])

  if (isLoading) return <Spinner />

  const update = (next: DnsServerData) => { setCfg(next); markDirty() }
  const set = <K extends keyof DnsServerData>(k: K, v: DnsServerData[K]) => update({ ...cfg, [k]: v })

  const ifaceNames = Object.keys(ifaces ?? {})

  const vrrpIfaces = (ha?.vrrp ?? [])
    .filter((v) => (v.vips ?? []).length > 0)
    .map((v) => v.interface)

  const zones = cfg.zones ?? []

  const addZone = () => {
    update({ ...cfg, zones: [...zones, { name: '' }] })
    setSection(`zone.${zones.length}`)
  }

  const removeZone = (i: number) => {
    const next = zones.filter((_, j) => j !== i)
    set('zones', next.length ? next : undefined)
    setSection('overview')
  }

  const items = [
    { key: 'overview', label: 'Overview' },
    { key: 'queries', label: 'Live queries' },
    { key: 'ddns', label: 'Dynamic DNS' },
    { key: 'resolver', label: 'Resolver' },
    { key: 'cache', label: 'Cache' },
    ...zones.map((z, i) => ({ key: `zone.${i}`, label: z.name || 'new zone' })),
    { key: ADD_ZONE, label: '+ Add zone' },
  ]

  const zoneIndex = section.startsWith('zone.') ? Number(section.slice(5)) : -1
  const zoneActive = zoneIndex >= 0 && zoneIndex < zones.length
  const active = section === 'queries' || section === 'ddns' || section === 'resolver' || section === 'cache' || zoneActive ? section : 'overview'

  const onChange = (key: string) => {
    if (key === ADD_ZONE) {
      addZone()
      return
    }
    setSection(key)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="DNS"
        action={<SaveButton isDirty={isDirty} saving={saving} onClick={() => save(cfg)} onCancel={() => { setCfg(data ?? {}); reset() }} />}
      />
      <SectionNav items={items} active={active} onChange={onChange}>
        {active === 'overview' && <DnsPanel />}
        {active === 'queries' && <DnsQueriesPanel />}
        {active === 'ddns' && <DdnsPanel />}
        {active === 'resolver' && <ResolverSettings cfg={cfg} set={set} ifaces={ifaces ?? {}} ifaceNames={ifaceNames} vrrpIfaces={vrrpIfaces} />}
        {active === 'cache' && <CacheSettings cfg={cfg} set={set} />}
        {zoneActive && (
          <ZoneEditor
            zone={zones[zoneIndex]}
            onChange={(z) => set('zones', zones.map((x, j) => (j === zoneIndex ? z : x)))}
            onRemove={() => removeZone(zoneIndex)}
            zones={zones}
            onZonesChange={(z) => set('zones', z.length ? z : undefined)}
          />
        )}
      </SectionNav>
    </div>
  )
}
