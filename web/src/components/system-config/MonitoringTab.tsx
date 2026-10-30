import type { ConfigFlowsConfig as FlowsConfig, ConfigLLDPConfig as LLDPConfig, ConfigMonitoringConfig as MonitoringConfig } from '@/api'
import { FlowSettings } from '@/components/system-config/MonitoringParts/FlowSettings'
import { INTERVAL_KEYS, OnStateChange } from '@/components/system-config/shared'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { PreferencesGroup, Spinner, SwitchRow, TagInput, EntryRow } from 'cheval-ui'
import { useCallback, useEffect, useState } from 'react'

type MonitoringTabShape = { onStateChange: OnStateChange }

export function MonitoringTab({ onStateChange }: MonitoringTabShape) {
  const { data, isLoading } = useFetch<MonitoringConfig>(() => api.apiConfigSectionGet({ section: 'monitoring' }) as Promise<MonitoringConfig>)
  const [monitoring, setMonitoring] = useState<MonitoringConfig>({})
  const [intervalRaw, setIntervalRaw] = useState<Record<string, string>>({})
  const [initialized, setInitialized] = useState(false)
  const { isDirty, markDirty, save, saving } = usePageSave('monitoring')
  useDataRefresh(() => setInitialized(false))

  useEffect(() => {
    if (data && !initialized) {
      setMonitoring(data)
      const raw: Record<string, string> = {}
      for (const { key } of INTERVAL_KEYS) raw[key] = String(data.collection?.[key] ?? 60)
      setIntervalRaw(raw)
      setInitialized(true)
    }
  }, [data, initialized])

  const handleSave = useCallback(() => save(monitoring), [save, monitoring])
  useEffect(() => { onStateChange({ isDirty, saving, save: handleSave }) }, [isDirty, saving, handleSave, onStateChange])

  const updateInterval = (key: string, raw: string) => {
    setIntervalRaw((p) => ({ ...p, [key]: raw }))
    const n = parseInt(raw, 10)
    if (!isNaN(n) && n >= 10) setMonitoring((prev) => ({ ...prev, collection: { ...prev.collection, [key]: n } }))
    markDirty()
  }

  const lldp = monitoring.lldp ?? {}
  const patchLldp = (next: Partial<LLDPConfig>) => { setMonitoring((prev) => ({ ...prev, lldp: { ...prev.lldp, ...next } })); markDirty() }
  const setFlows = (flows: FlowsConfig) => { setMonitoring((prev) => ({ ...prev, flows })); markDirty() }

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      <PreferencesGroup title="Collection Intervals" description="In seconds, minimum 10">
        {INTERVAL_KEYS.map(({ key, label }) => (
          <EntryRow key={key} title={label} value={intervalRaw[key] ?? ''} onChange={(e) => updateInterval(key, e.target.value)} className="font-mono" placeholder="60" suffix={<span className="text-xs text-muted-foreground">s</span>} />
        ))}
      </PreferencesGroup>

      <PreferencesGroup title="Link-Layer Discovery (LLDP / CDP)" description="Discovered devices appear in Monitoring › Network › Neighbors">
        <SwitchRow
          title="LLDP / CDP"
          subtitle="Run lldpd"
          checked={!!lldp.enabled}
          onCheckedChange={(v) => patchLldp({ enabled: v })}
        />
      </PreferencesGroup>

      {lldp.enabled && (
        <>
          <PreferencesGroup>
            <SwitchRow
              title="Receive CDP"
              subtitle="Also discover neighbors advertising Cisco Discovery Protocol"
              checked={!!lldp.cdp}
              onCheckedChange={(v) => patchLldp({ cdp: v })}
            />
            <SwitchRow
              title="Advertise This Router"
              subtitle="When off, lldpd only listens"
              checked={!!lldp.transmit}
              onCheckedChange={(v) => patchLldp({ transmit: v })}
            />
          </PreferencesGroup>

          <PreferencesGroup title="Interfaces" description="Empty uses every interface">
            <div className="px-4 py-3">
              <TagInput values={lldp.interfaces ?? []} onChange={(v) => patchLldp({ interfaces: v })} placeholder="eth1" mono />
            </div>
          </PreferencesGroup>
        </>
      )}

      <FlowSettings flows={monitoring.flows ?? {}} onChange={setFlows} />
    </div>
  )
}
