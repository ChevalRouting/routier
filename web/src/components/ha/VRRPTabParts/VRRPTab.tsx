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
  const rows: Row[] = instances.map((v) => ({ v }))

  const update = (next: VRRPInstance[]) => { setInstances(next); onDirty() }

  const setInstance = (index: number, v: VRRPInstance) =>
    update(instances.map((it, i) => (i === index ? v : it)))

  const removeInstance = (index: number) => update(instances.filter((_, i) => i !== index))

  const addInstance = () => update([...instances, emptyInstance(ifaceNames[0] ?? '')])

  if (ifaceNames.length === 0) {
    return (
      <EmptyState className="py-12" title="No Interfaces"
        message="Add an interface first" />
    )
  }

  return (
    <AccordionList
      items={rows.map((r, i) => ({ ...r, index: i }))}
      getId={(it) => String(it.index)}
      addLabel="Add Instance"
      onAdd={addInstance}
      onRemove={(it) => removeInstance(it.index)}
      emptyTitle="No VRRP Instances"
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
