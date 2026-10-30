import type { ConfigFlowCollector as FlowCollector, ConfigFlowsConfig as FlowsConfig } from '@/api'
import { isIP } from '@/lib/validate'
import { Button, EntryRow, IntegerEntryRow, PreferencesGroup, SwitchRow, TagInput } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { formatCollector, parseCollector } from './collectorAddress'

type FlowSettingsShape = {
  flows: FlowsConfig
  onChange: (next: FlowsConfig) => void
}

type CollectorRowShape = {
  index: number
  collector: FlowCollector
  onChange: (index: number, collector: FlowCollector | null, text: string) => void
  onRemove: (index: number) => void
}

function CollectorRow({ index, collector, onChange, onRemove }: CollectorRowShape) {
  const [text, setText] = useState(formatCollector(collector))
  const invalid = text !== '' && parseCollector(text) === null

  const handleChange = (value: string) => {
    setText(value)
    onChange(index, parseCollector(value), value)
  }

  return (
    <EntryRow
      title={`Collector ${index + 1}`}
      value={text}
      onChange={(e) => handleChange(e.target.value)}
      placeholder="198.51.100.10:6343"
      error={invalid ? 'Enter an address, optionally with a port' : null}
      className="font-mono"
      suffix={(
        <Button variant="ghost" size="icon" onClick={() => onRemove(index)} aria-label="Remove Collector" title="Remove Collector" className="text-muted-foreground hover:text-destructive">
          <Trash2 className="h-4 w-4" />
        </Button>
      )}
    />
  )
}

export function FlowSettings({ flows, onChange }: FlowSettingsShape) {
  const [collectorKeys, setCollectorKeys] = useState(() => (flows.collectors ?? []).map((_, i) => i))
  const collectors = flows.collectors ?? []
  const retention = flows.retention ?? {}
  const sourceInvalid = !!flows.source && !isIP(flows.source)

  const patch = (next: Partial<FlowsConfig>) => onChange({ ...flows, ...next })

  const addCollector = () => {
    setCollectorKeys((keys) => [...keys, Math.max(-1, ...keys) + 1])
    patch({ collectors: [...collectors, { address: '' }] })
  }

  const updateCollector = (index: number, collector: FlowCollector | null, text: string) => {
    const next = [...collectors]
    next[index] = collector ?? { address: text }
    patch({ collectors: next })
  }

  const removeCollector = (index: number) => {
    setCollectorKeys((keys) => keys.filter((_, i) => i !== index))
    patch({ collectors: collectors.filter((_, i) => i !== index) })
  }

  return (
    <>
      <PreferencesGroup title="Traffic Flows (sFlow)" description="Sampled flows appear in Monitoring › Network › Flows">
        <SwitchRow title="sFlow Sampling" subtitle="Adds a sampling rule to nftables" checked={!!flows.enabled} onCheckedChange={(v) => patch({ enabled: v })} />
      </PreferencesGroup>

      {flows.enabled && (
        <>
          <PreferencesGroup>
            <IntegerEntryRow title="Sampling Rate" value={flows.sampling ?? 1000} min={1} max={1048576} onValueChange={(v) => patch({ sampling: v })} className="font-mono" suffix={<span className="text-xs text-muted-foreground">1 in N</span>} />
            <EntryRow title="Source Address" value={flows.source ?? ''} onChange={(e) => patch({ source: e.target.value.trim() || undefined })} placeholder="Route towards each collector" error={sourceInvalid ? 'Enter an IP address' : null} className="font-mono" />
          </PreferencesGroup>

          <PreferencesGroup title="Sampled Interfaces" description="Empty samples every interface">
            <div className="px-4 py-3">
              <TagInput values={flows.interfaces ?? []} onChange={(v) => patch({ interfaces: v })} placeholder="wan" mono />
            </div>
          </PreferencesGroup>

          <PreferencesGroup
            title="Collectors"
            description="Default port 6343; empty keeps flows only on this router"
            header={(
              <Button variant="ghost" size="icon" onClick={addCollector} aria-label="Add Collector" title="Add Collector">
                <Plus className="h-4 w-4" />
              </Button>
            )}
          >
            {collectors.map((collector, index) => (
              <CollectorRow key={collectorKeys[index]} index={index} collector={collector} onChange={updateCollector} onRemove={removeCollector} />
            ))}
          </PreferencesGroup>

          <PreferencesGroup title="Local History">
            <IntegerEntryRow title="Retention" value={retention.hours ?? 72} min={1} onValueChange={(v) => patch({ retention: { ...retention, hours: v } })} className="font-mono" suffix={<span className="text-xs text-muted-foreground">h</span>} />
            <IntegerEntryRow title="Maximum Flows" value={retention.max_flows ?? 200000} min={1} onValueChange={(v) => patch({ retention: { ...retention, max_flows: v } })} className="font-mono" />
          </PreferencesGroup>
        </>
      )}
    </>
  )
}
