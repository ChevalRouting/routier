import { useEffect, useState } from 'react'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { useDataRefresh } from '@/lib/dataVersion'
import { usePageSave } from '@/lib/usePageSave'
import { SectionNav } from 'cheval-ui'
import { useTabState } from 'cheval-ui'
import { SaveButton } from 'cheval-ui'
import { Spinner } from 'cheval-ui'
import { ConntrackdConfig } from '@/components/ha/ConntrackdConfig'
import { VRRPTab } from '@/components/ha/VRRPTab'
import { Conntrackd, HAConfigData, VRRPInstance } from '@/components/ha/types'

type Sub = 'vrrp' | 'conntrackd'

const ITEMS: { key: Sub; label: string }[] = [
  { key: 'vrrp', label: 'VRRP' },
  { key: 'conntrackd', label: 'Conntrackd' },
]

export function HAConfig({ onActionChange }: { onActionChange?: (a: React.ReactNode) => void }) {
  const { data, isLoading } = useFetch<HAConfigData | null>(() => api.apiConfigSectionGet({ section: 'ha' }) as Promise<HAConfigData | null>)
  const { data: ifaces } = useFetch<Record<string, unknown>>(
    () => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<Record<string, unknown>>
  )
  const [vrrp, setVrrp] = useState<VRRPInstance[]>([])
  const [conntrackd, setConntrackd] = useState<Conntrackd | null>(null)
  const [initialized, setInitialized] = useState(false)
  const [sub, setSub] = useTabState<Sub>('ha.config', 'vrrp')
  const { isDirty, markDirty, save, saving, reset } = usePageSave('ha')

  useDataRefresh(() => { setInitialized(false); reset() })

  useEffect(() => {
    if (!isLoading && !initialized) {
      setVrrp(data?.vrrp ?? [])
      setConntrackd(data?.conntrackd ?? null)
      setInitialized(true)
    }
  }, [isLoading, initialized, data])

  useEffect(() => {
    const payload: HAConfigData = {}
    if (vrrp.length > 0) payload.vrrp = vrrp
    if (conntrackd) payload.conntrackd = conntrackd

    const cancel = () => { setVrrp(data?.vrrp ?? []); setConntrackd(data?.conntrackd ?? null); reset() }

    onActionChange?.(
      <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(payload)} onCancel={cancel} />
    )
    return () => onActionChange?.(null)
  }, [isDirty, saving, vrrp, conntrackd, data, onActionChange, save, reset])

  if (isLoading) return <Spinner />

  const ifaceNames = Object.keys(ifaces ?? {})

  return (
    <SectionNav items={ITEMS} active={sub} onChange={setSub}>
      {sub === 'vrrp' && (
        <VRRPTab instances={vrrp} setInstances={(v) => { setVrrp(v); markDirty() }} ifaceNames={ifaceNames} onDirty={markDirty} />
      )}
      {sub === 'conntrackd' && (
        <ConntrackdConfig cfg={conntrackd} onChange={(v) => { setConntrackd(v); markDirty() }} />
      )}
    </SectionNav>
  )
}
