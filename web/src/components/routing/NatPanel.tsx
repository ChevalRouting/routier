import type { NatKind, NatSpec } from '@/api'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { Badge, Button, Card, EmptyState, Spinner } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'
import { NatBody } from './NatPanelParts/NatBody'

type NatSummaryShape = { spec: NatSpec }

type NatPanelShape = { onStateChange?: (s: NatSaveState) => void }

export const KIND_LABEL: Record<NatKind, string> = {
  masquerade: 'Masquerade',
  snat: 'SNAT',
  dnat: 'Port forward (DNAT)',
}

export interface NatSaveState {
  isDirty: boolean
  saving: boolean
  save: () => void
  cancel: () => void
}

export function NatSummary({ spec }: NatSummaryShape) {
  const detail =
    spec.kind === 'dnat'
      ? `${spec.in || 'any'} ${spec.proto || 'tcp'}/${spec.dport || '?'} → ${spec.to || '?'}`
      : spec.kind === 'snat'
        ? `${spec.source || 'any'} → ${spec.to || '?'}${spec.out ? ` out ${spec.out}` : ''}`
        : `out ${spec.out || 'any'}${spec.source ? ` from ${spec.source}` : ''}`
  return (
    <div className="flex min-w-0 flex-1 items-center gap-3">
      <Badge variant="secondary" className="shrink-0 text-[11px]">{KIND_LABEL[spec.kind]}</Badge>
      <span className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">{detail}</span>
      {spec.comment && <span className="hidden shrink-0 truncate text-xs text-muted-foreground sm:block">{spec.comment}</span>}
    </div>
  )
}

export default function NatPanel({ onStateChange }: NatPanelShape) {
  const { data, isLoading, reload } = useFetch<NatSpec[]>(() => api.apiNatGet())
  const { data: vars } = useFetch(() => api.apiConfigNftablesVarsGet())
  const [specs, setSpecs] = useState<NatSpec[]>([])
  const [initialized, setInitialized] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const revision = useRef(0)

  useDataRefresh(() => { revision.current += 1; setInitialized(false); setDirty(false) })

  useEffect(() => {
    if (!isLoading && !initialized) {
      setSpecs(data ?? [])
      setInitialized(true)
    }
  }, [isLoading, initialized, data])

  const ifaceVars = useMemo(
    () => (vars ?? []).map((v) => v.name).filter((n) => n.endsWith('_interfaces') || n === 'interfaces' || n === 'wireguard' || n === 'tunnels'),
    [vars],
  )
  const addrVars = useMemo(
    () => (vars ?? []).map((v) => v.name).filter((n) => n.includes('_address') || n.includes('_network')),
    [vars],
  )

  const update = (next: NatSpec[]) => { revision.current += 1; setSpecs(next); setDirty(true) }
  const setSpec = (i: number, patch: Partial<NatSpec>) =>
    update(specs.map((s, j) => (j === i ? { ...s, ...patch } : s)))
  const add = (kind: NatKind) =>
    update([...specs, kind === 'dnat' ? { kind, proto: 'tcp' } : { kind }])
  const remove = (i: number) => update(specs.filter((_, j) => j !== i))

  const save = useCallback(async () => {
    const savedRevision = revision.current
    setSaving(true)
    try {
      await api.apiNatPut({ NatSpec: specs })
      if (revision.current === savedRevision) setDirty(false)
      toast.success('NAT rules staged; apply to activate')
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save NAT rules')
    } finally {
      setSaving(false)
    }
  }, [specs])

  const cancel = useCallback(() => { revision.current += 1; setInitialized(false); setDirty(false); reload(true) }, [reload])

  useEffect(() => {
    onStateChange?.({ isDirty: dirty, saving, save, cancel })
  }, [dirty, saving, save, cancel, onStateChange])

  if (isLoading) return <Spinner />

  const addButtons = (
    <div className="flex gap-2">
      {(['masquerade', 'snat', 'dnat'] as NatKind[]).map((k) => (
        <Button key={k} variant="outline" size="sm" className="gap-1 h-8 text-xs" onClick={() => add(k)}>
          <Plus className="h-3 w-3" />{KIND_LABEL[k]}
        </Button>
      ))}
    </div>
  )

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground max-w-2xl">
        Shortcuts that write tagged nftables rules into the firewall chains (masquerade/SNAT in postrouting,
        port-forwards in prerouting). They show up on the Firewall page and can be edited or removed there too.
      </p>

      {addButtons}
      {specs.length === 0 ? (
        <EmptyState
          title="No NAT rules"
          message="Masquerade, SNAT and port-forward shortcuts write tagged rules into the firewall chains."
        />
      ) : (
        <div className="grid gap-3">
          {specs.map((spec, index) => (
            <Card key={index} className="space-y-4 p-4">
              <div className="flex items-center gap-3">
                <NatSummary spec={spec} />
                <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive" onClick={() => remove(index)}>
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
              <NatBody spec={spec} ifaceVars={ifaceVars} addrVars={addrVars} onChange={(patch) => setSpec(index, patch)} />
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}

export { NatBody } from './NatPanelParts/NatBody'
