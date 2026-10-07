import { OSPFAreasPanel, OSPFGeneralPanel, OSPFInterfacesPanel } from '@/components/routing/ospf/OSPFTab'
import { OSPFConfig, OSPFSubTab } from '@/components/routing/types'
import { Label, SectionNav, Switch, useTabState } from 'cheval-ui'

type OSPFTabShape = {
  ospf: OSPFConfig
  setOSPF: (v: OSPFConfig) => void
  enabled: boolean
  setEnabled: (v: boolean) => void
  onDirty: () => void
  ifaceNames: string[]
}

type SubTabsShape = { key: OSPFSubTab; label: string }

export function OSPFTab({
  ospf, setOSPF, enabled, setEnabled, onDirty, ifaceNames,
}: OSPFTabShape) {
  const [subTab, setSubTab] = useTabState<OSPFSubTab>('routing.ospf', 'general')

  const subTabs: SubTabsShape[] = [
    { key: 'general', label: 'General' },
    { key: 'areas', label: ospf.areas?.length ? `Areas (${ospf.areas.length})` : 'Areas' },
    { key: 'interfaces', label: Object.keys(ospf.interfaces ?? {}).length ? `Interfaces (${Object.keys(ospf.interfaces ?? {}).length})` : 'Interfaces' },
  ]

  return (
    <div className="space-y-5">
      <p className="text-sm text-muted-foreground">
        OSPF (Open Shortest Path First) is a link-state IGP that distributes IPv4 routes within your
        network by flooding link-state information across areas.
      </p>
      <div className="flex items-center justify-between gap-2">
        <Label htmlFor="ospf-enable" className="cursor-pointer">Enable OSPF</Label>
        <Switch id="ospf-enable" checked={enabled} onCheckedChange={(v) => { setEnabled(v); onDirty() }} />
      </div>

      {!enabled ? (
        <p className="text-sm text-muted-foreground italic">OSPF is disabled. Enable it above to configure.</p>
      ) : (
        <SectionNav items={subTabs} active={subTab} onChange={setSubTab}>
          {subTab === 'general' && (
            <OSPFGeneralPanel ospf={ospf} setOSPF={setOSPF} onDirty={onDirty} />
          )}
          {subTab === 'areas' && (
            <OSPFAreasPanel
              areas={ospf.areas ?? []}
              onChange={(areas) => setOSPF({ ...ospf, areas })}
              onDirty={onDirty}
            />
          )}
          {subTab === 'interfaces' && (
            <OSPFInterfacesPanel
              interfaces={ospf.interfaces ?? {}}
              onChange={(interfaces) => setOSPF({ ...ospf, interfaces })}
              ifaceNames={ifaceNames}
              areaIds={(ospf.areas ?? []).map((a) => a.id).filter(Boolean)}
              onDirty={onDirty}
            />
          )}
        </SectionNav>
      )}
    </div>
  )
}
