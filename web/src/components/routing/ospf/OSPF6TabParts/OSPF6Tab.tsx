import { OSPF6AreasPanel, OSPF6GeneralPanel, OSPF6InterfacesPanel } from '@/components/routing/ospf/OSPF6Tab'
import { OSPF6Config, OSPF6SubTab } from '@/components/routing/types'
import { Button, EmptyState, PreferencesGroup, SwitchRow } from 'cheval-ui'
import { Network } from 'lucide-react'

type OSPF6TabShape = {
  section: OSPF6SubTab
  ospf6: OSPF6Config
  setOSPF6: (v: OSPF6Config) => void
  enabled: boolean
  setEnabled: (v: boolean) => void
  onDirty: () => void
  ifaceNames: string[]
}

export function OSPF6Tab({ section, ospf6, setOSPF6, enabled, setEnabled, onDirty, ifaceNames }: OSPF6TabShape) {
  const enable = (value: boolean) => {
    setEnabled(value)
    onDirty()
  }

  if (!enabled && section !== 'general') {
    return (
      <EmptyState
        icon={<Network />}
        title="OSPFv3 Disabled"
        action={<Button variant="suggested" onClick={() => enable(true)}>Enable OSPFv3</Button>}
      />
    )
  }

  return (
    <div className="space-y-6">
      {section === 'general' && (
        <PreferencesGroup>
          <SwitchRow title="OSPFv3" checked={enabled} onCheckedChange={enable} />
        </PreferencesGroup>
      )}
      {enabled && section === 'general' && <OSPF6GeneralPanel ospf6={ospf6} setOSPF6={setOSPF6} onDirty={onDirty} />}
      {enabled && section === 'areas' && (
        <OSPF6AreasPanel areas={ospf6.areas ?? []} onChange={(areas) => setOSPF6({ ...ospf6, areas })} onDirty={onDirty} />
      )}
      {enabled && section === 'interfaces' && (
        <OSPF6InterfacesPanel
          interfaces={ospf6.interfaces ?? {}}
          onChange={(interfaces) => setOSPF6({ ...ospf6, interfaces })}
          ifaceNames={ifaceNames}
          areaIds={(ospf6.areas ?? []).map((a) => a.id).filter(Boolean)}
          onDirty={onDirty}
        />
      )}
    </div>
  )
}
