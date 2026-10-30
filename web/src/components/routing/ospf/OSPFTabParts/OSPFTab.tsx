import { OSPFAreasPanel, OSPFGeneralPanel, OSPFInterfacesPanel } from '@/components/routing/ospf/OSPFTab'
import { OSPFConfig, OSPFSubTab } from '@/components/routing/types'
import { Button, EmptyState, PreferencesGroup, SwitchRow } from 'cheval-ui'
import { Network } from 'lucide-react'

type OSPFTabShape = {
  section: OSPFSubTab
  ospf: OSPFConfig
  setOSPF: (v: OSPFConfig) => void
  enabled: boolean
  setEnabled: (v: boolean) => void
  onDirty: () => void
  ifaceNames: string[]
}

export function OSPFTab({
  section, ospf, setOSPF, enabled, setEnabled, onDirty, ifaceNames,
}: OSPFTabShape) {
  const enable = (value: boolean) => {
    setEnabled(value)
    onDirty()
  }

  if (!enabled && section !== 'general') {
    return (
      <EmptyState
        icon={<Network />}
        title="OSPF Disabled"
        action={<Button variant="suggested" onClick={() => enable(true)}>Enable OSPF</Button>}
      />
    )
  }

  return (
    <div className="space-y-6">
      {section === 'general' && (
        <PreferencesGroup>
          <SwitchRow title="OSPF" checked={enabled} onCheckedChange={enable} />
        </PreferencesGroup>
      )}
      {enabled && section === 'general' && (
        <OSPFGeneralPanel ospf={ospf} setOSPF={setOSPF} onDirty={onDirty} />
      )}
      {enabled && section === 'areas' && (
        <OSPFAreasPanel
          areas={ospf.areas ?? []}
          onChange={(areas) => setOSPF({ ...ospf, areas })}
          onDirty={onDirty}
        />
      )}
      {enabled && section === 'interfaces' && (
        <OSPFInterfacesPanel
          interfaces={ospf.interfaces ?? {}}
          onChange={(interfaces) => setOSPF({ ...ospf, interfaces })}
          ifaceNames={ifaceNames}
          areaIds={(ospf.areas ?? []).map((a) => a.id).filter(Boolean)}
          onDirty={onDirty}
        />
      )}
    </div>
  )
}
