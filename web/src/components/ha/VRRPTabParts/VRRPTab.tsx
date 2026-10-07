import { VRRPInstance } from '@/components/ha/types'
import { emptyInstance, Row, VRRPBody, VRRPSummary } from '@/components/ha/VRRPTab'
import { AccordionList } from '@/components/ui/AccordionList'
import { EmptyState } from 'cheval-ui'

type VRRPTabShape = {
  instances: VRRPInstance[]
  setInstances: (v: VRRPInstance[]) => void
  ifaceNames: string[]
  onDirty: () => void
}

export function VRRPTab({ instances, setInstances, ifaceNames, onDirty }: VRRPTabShape) {
  const rows: Row[] = instances.map((v) => ({ uid: `${v.interface}-${v.id}-${v.name ?? ''}`, v }))

  const update = (next: VRRPInstance[]) => { setInstances(next); onDirty() }

  const setInstance = (index: number, v: VRRPInstance) =>
    update(instances.map((it, i) => (i === index ? v : it)))

  const removeInstance = (index: number) => update(instances.filter((_, i) => i !== index))

  const addInstance = () => update([...instances, emptyInstance(ifaceNames[0] ?? '')])

  if (ifaceNames.length === 0) {
    return (
      <EmptyState className="py-12" title="No interfaces"
        message="Add an interface first, then define VRRP failover groups here." />
    )
  }

  return (
    <AccordionList
      items={rows.map((r, i) => ({ ...r, index: i }))}
      getId={(it) => `${it.uid}-${it.index}`}
      description="Virtual router failover groups managed by keepalived."
      addLabel="Add instance"
      onAdd={addInstance}
      onRemove={(it) => removeInstance(it.index)}
      emptyTitle="No VRRP instances"
      emptyMessage="Add an instance to create a failover group on an interface."
      renderSummary={(it) => <VRRPSummary v={it.v} />}
      renderBody={(it) => (
        <VRRPBody
          v={it.v}
          ifaceNames={ifaceNames}
          onChange={(v) => setInstance(it.index, v)}
        />
      )}
    />
  )
}
