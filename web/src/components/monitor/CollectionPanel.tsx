import { useState, useEffect } from 'react'
import { api } from '@/lib/client'
import type { ConfigMonitoringConfig as MonitoringConfig } from '@/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Clock } from 'lucide-react'
import { OnActionChange } from './shared'

export const INTERVAL_KEYS: Array<{ key: keyof NonNullable<MonitoringConfig['collection']>; label: string }> = [
  { key: 'iface',     label: 'Interfaces' },
  { key: 'system',    label: 'CPU / RAM' },
  { key: 'bgp',       label: 'BGP' },
  { key: 'proto',     label: 'Protocols' },
  { key: 'neighbors', label: 'Neighbors' },
  { key: 'routes',    label: 'Route cache TTL' },
]
export function CollectionPanel({ onActionChange }: { onActionChange: OnActionChange }) {
  const [monitoring, setMonitoring] = useState<MonitoringConfig>({})
  const [intervalRaw, setIntervalRaw] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    onActionChange(null)
  }, [onActionChange])

  useEffect(() => {
    const defaults = () => {
      const raw: Record<string, string> = {}
      for (const { key } of INTERVAL_KEYS) raw[key] = '60'
      setIntervalRaw(raw)
    }
    api.apiConfigSectionGet({ section: 'monitoring' }).then((d) => {
      const cfg = (d as MonitoringConfig) ?? {}
      setMonitoring(cfg)
      const raw: Record<string, string> = {}
      for (const { key } of INTERVAL_KEYS) raw[key] = String(cfg.collection?.[key] ?? 60)
      setIntervalRaw(raw)
    }).catch(defaults)
  }, [])

  const handleChange = (key: string, raw: string) => {
    setIntervalRaw((p) => ({ ...p, [key]: raw }))
    const n = parseInt(raw, 10)
    if (!isNaN(n) && n >= 10) {
      setMonitoring((prev) => ({ ...prev, collection: { ...prev.collection, [key]: n } }))
    }
  }

  const save = async () => {
    setSaving(true)
    try { await api.apiConfigSectionPut({ section: 'monitoring', body: monitoring }) } finally { setSaving(false) }
  }

  return (
    <div className="max-w-lg space-y-4">
      <Card>
        <CardHeader className="pb-2">
          <div className="flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground" />
            <CardTitle className="text-sm">Collection Intervals</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid gap-3 grid-cols-2">
            {INTERVAL_KEYS.map(({ key, label }) => (
              <div key={key} className="space-y-1.5">
                <Label className="text-xs text-muted-foreground">{label}</Label>
                <div className="flex items-center gap-1.5">
                  <Input
                    value={intervalRaw[key] ?? String(monitoring.collection?.[key] ?? 60)}
                    onChange={(e) => handleChange(key, e.target.value)}
                    className="font-mono text-xs h-8"
                    placeholder="60"
                  />
                  <span className="text-xs text-muted-foreground shrink-0">s</span>
                </div>
              </div>
            ))}
          </div>
          <p className="text-xs text-muted-foreground">Changes are staged and require an apply to take effect.</p>
          <div className="flex justify-end">
            <Button size="sm" onClick={save} disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

