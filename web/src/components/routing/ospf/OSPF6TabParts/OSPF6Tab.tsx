import { OSPF6AreasPanel, OSPF6GeneralPanel, OSPF6InterfacesPanel } from '@/components/routing/ospf/OSPF6Tab'
import { OSPF6Config, OSPF6SubTab } from '@/components/routing/types'
import { Label, SectionNav, Switch, useTabState } from 'cheval-ui'

type OSPF6TabShape = {
  ospf6: OSPF6Config; setOSPF6: (v: OSPF6Config) => void
  enabled: boolean; setEnabled: (v: boolean) => void
  onDirty: () => void; ifaceNames: string[]
}

type SubTabsShape = { key: OSPF6SubTab; label: string }

export function OSPF6Tab({ ospf6, setOSPF6, enabled, setEnabled, onDirty, ifaceNames }: OSPF6TabShape) {
  const [subTab, setSubTab] = useTabState<OSPF6SubTab>('routing.ospf6', 'general')
  const subTabs: SubTabsShape[] = [
    { key: 'general', label: 'General' },
    { key: 'areas', label: ospf6.areas?.length ? `Areas (${ospf6.areas.length})` : 'Areas' },
    { key: 'interfaces', label: Object.keys(ospf6.interfaces ?? {}).length ? `Interfaces (${Object.keys(ospf6.interfaces ?? {}).length})` : 'Interfaces' },
  ]
  return (
    <div className="space-y-5">
      <p className="text-sm text-muted-foreground">
        OSPFv3 is the link-state IGP for IPv6, distributing IPv6 routes across areas much like OSPF
        does for IPv4.
      </p>
      <div className="flex items-center justify-between gap-2">
        <Label htmlFor="ospf6-enable" className="cursor-pointer">Enable OSPFv3</Label>
        <Switch id="ospf6-enable" checked={enabled} onCheckedChange={(v) => { setEnabled(v); onDirty() }} />
      </div>
      {!enabled ? (
        <p className="text-sm text-muted-foreground italic">OSPFv3 is disabled. Enable it above to configure.</p>
      ) : (
        <SectionNav items={subTabs} active={subTab} onChange={setSubTab}>
          {subTab === 'general' && <OSPF6GeneralPanel ospf6={ospf6} setOSPF6={setOSPF6} onDirty={onDirty} />}
          {subTab === 'areas' && (
            <OSPF6AreasPanel areas={ospf6.areas ?? []} onChange={(areas) => setOSPF6({ ...ospf6, areas })} onDirty={onDirty} />
          )}
          {subTab === 'interfaces' && (
            <OSPF6InterfacesPanel
              interfaces={ospf6.interfaces ?? {}}
              onChange={(interfaces) => setOSPF6({ ...ospf6, interfaces })}
              ifaceNames={ifaceNames}
              areaIds={(ospf6.areas ?? []).map((a) => a.id).filter(Boolean)}
              onDirty={onDirty}
            />
          )}
        </SectionNav>
      )}
    </div>
  )
}
