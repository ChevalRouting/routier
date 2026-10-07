import type { ConfigLLDPConfig as LLDPConfig, ConfigMonitoringConfig as MonitoringConfig } from '@/api'
import { INTERVAL_KEYS, OnStateChange } from '@/components/system-config/shared'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { Input, Label, PreferencesGroup, Spinner, SwitchRow, TagInput } from 'cheval-ui'
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

  if (isLoading) return <Spinner />

  return (
    <div className="max-w-2xl space-y-6">
      <PreferencesGroup title="Collection intervals" description="How often each metric is sampled, in seconds (minimum 10).">
        <div className="grid gap-3 grid-cols-2 px-4 py-3">
          {INTERVAL_KEYS.map(({ key, label }) => (
            <div key={key} className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">{label}</Label>
              <div className="flex items-center gap-1.5">
                <Input value={intervalRaw[key] ?? ''} onChange={(e) => updateInterval(key, e.target.value)} className="font-mono text-xs h-8" placeholder="60" />
                <span className="text-xs text-muted-foreground shrink-0">s</span>
              </div>
            </div>
          ))}
        </div>
      </PreferencesGroup>

      <PreferencesGroup title="Link-layer discovery (LLDP / CDP)" description="Discover directly connected switches and routers. Discovered devices appear in Monitor > Neighbors.">
        <SwitchRow
          title="Enable LLDP / CDP"
          subtitle="Run lldpd to discover directly connected devices"
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
              title="Advertise this router"
              subtitle="Transmit LLDP so upstream devices can discover this router. When off, lldpd only listens."
              checked={!!lldp.transmit}
              onCheckedChange={(v) => patchLldp({ transmit: v })}
            />
          </PreferencesGroup>

          <PreferencesGroup title="Interfaces" description="Restrict discovery to these interfaces. Leave empty for all.">
            <div className="px-4 py-3">
              <TagInput values={lldp.interfaces ?? []} onChange={(v) => patchLldp({ interfaces: v })} placeholder="eth1" mono />
            </div>
          </PreferencesGroup>
        </>
      )}
    </div>
  )
}
