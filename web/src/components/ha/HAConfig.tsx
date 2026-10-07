import { ConntrackdConfig } from '@/components/ha/ConntrackdConfig'
import { VRRPTab } from '@/components/ha/VRRPTab'
import { Conntrackd, HAConfigData, VRRPInstance } from '@/components/ha/types'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { SaveButton, SectionNav, Spinner, useTabState } from 'cheval-ui'
import { useEffect, useState } from 'react'

type ITEMSShape = { key: Sub; label: string }

type HAConfigShape = { onActionChange?: (a: React.ReactNode) => void }

type Sub = 'vrrp' | 'conntrackd'

const ITEMS: ITEMSShape[] = [
  { key: 'vrrp', label: 'VRRP' },
  { key: 'conntrackd', label: 'Conntrackd' },
]

export function HAConfig({ onActionChange }: HAConfigShape) {
  const { data, isLoading, reload } = useFetch<HAConfigData | null>(() => api.apiConfigSectionGet({ section: 'ha' }) as Promise<HAConfigData | null>)
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

    const cancel = () => { setInitialized(false); reset(); reload(true) }

    onActionChange?.(
      <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(payload)} onCancel={cancel} />
    )
    return () => onActionChange?.(null)
  }, [isDirty, saving, vrrp, conntrackd, data, onActionChange, save, reset, reload])

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
