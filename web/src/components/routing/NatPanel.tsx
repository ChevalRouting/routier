import { useCallback, useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import type { NatSpec, NatKind } from '@/api'
import { useFetch } from '@/lib/useFetch'
import { useDataRefresh } from '@/lib/dataVersion'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/Spinner'
import { EmptyState } from '@/components/EmptyState'
import { VarPickerInput } from '@/components/VarPickerInput'
import { Plus, Trash2 } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const KIND_LABEL: Record<NatKind, string> = {
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

export default function NatPanel({ onStateChange }: { onStateChange?: (s: NatSaveState) => void }) {
  const { data, isLoading } = useFetch<NatSpec[]>(() => api.apiNatGet())
  const { data: vars } = useFetch(() => api.apiConfigNftablesVarsGet())
  const [specs, setSpecs] = useState<NatSpec[]>([])
  const [initialized, setInitialized] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)

  useDataRefresh(() => { setInitialized(false); setDirty(false) })

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

  const update = (next: NatSpec[]) => { setSpecs(next); setDirty(true) }
  const setSpec = (i: number, patch: Partial<NatSpec>) =>
    update(specs.map((s, j) => (j === i ? { ...s, ...patch } : s)))
  const add = (kind: NatKind) =>
    update([...specs, kind === 'dnat' ? { kind, proto: 'tcp' } : { kind }])
  const remove = (i: number) => update(specs.filter((_, j) => j !== i))

  const save = useCallback(async () => {
    setSaving(true)
    try {
      await api.apiNatPut({ NatSpec: specs })
      setDirty(false)
      toast.success('NAT rules saved')
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save NAT rules')
    } finally {
      setSaving(false)
    }
  }, [specs])

  const cancel = useCallback(() => { setSpecs(data ?? []); setDirty(false) }, [data])

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

      {specs.length === 0 ? (
        <EmptyState
          title="No NAT rules"
          message="Masquerade, SNAT and port-forward shortcuts write tagged rules into the firewall chains."
          action={addButtons}
        />
      ) : (
        <>
        {addButtons}
        <div className="space-y-2">
          {specs.map((s, i) => (
            <div key={i} className="border rounded-md p-3 space-y-3">
              <div className="flex items-center justify-between">
                <Badge variant="secondary" className="text-[11px]">{KIND_LABEL[s.kind]}</Badge>
                <Button variant="ghost" size="icon" className="h-6 w-6 hover:text-destructive" onClick={() => remove(i)}>
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>

              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {(s.kind === 'masquerade' || s.kind === 'snat') && (
                  <div className="space-y-1">
                    <Label className="text-[11px]">Out interface</Label>
                    <VarPickerInput value={s.out ?? ''} onChange={(v) => setSpec(i, { out: v })} vars={ifaceVars} placeholder="$wan_interfaces" mono />
                  </div>
                )}
                {s.kind === 'snat' && (
                  <div className="space-y-1">
                    <Label className="text-[11px]">SNAT to</Label>
                    <Input value={s.to ?? ''} onChange={(e) => setSpec(i, { to: e.target.value })} placeholder="203.0.113.5" className="font-mono text-sm h-8" />
                  </div>
                )}
                {(s.kind === 'masquerade' || s.kind === 'snat') && (
                  <>
                    <div className="space-y-1">
                      <Label className="text-[11px]">Source (optional)</Label>
                      <VarPickerInput value={s.source ?? ''} onChange={(v) => setSpec(i, { source: v })} vars={addrVars} placeholder="$lan_network" mono />
                    </div>
                    {s.source ? (
                      <div className="space-y-1">
                        <Label className="text-[11px]">Source family</Label>
                        <Select value={s.family || 'ip'} onValueChange={(v) => setSpec(i, { family: v })}>
                          <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
                          <SelectContent>
                            <SelectItem value="ip">ip</SelectItem>
                            <SelectItem value="ip6">ip6</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    ) : null}
                  </>
                )}

                {s.kind === 'dnat' && (
                  <>
                    <div className="space-y-1">
                      <Label className="text-[11px]">In interface</Label>
                      <VarPickerInput value={s.in ?? ''} onChange={(v) => setSpec(i, { in: v })} vars={ifaceVars} placeholder="$wan_interfaces" mono />
                    </div>
                    <div className="space-y-1">
                      <Label className="text-[11px]">Protocol</Label>
                      <Select value={s.proto || 'tcp'} onValueChange={(v) => setSpec(i, { proto: v })}>
                        <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="tcp">tcp</SelectItem>
                          <SelectItem value="udp">udp</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-1">
                      <Label className="text-[11px]">Dest port</Label>
                      <Input value={s.dport ?? ''} onChange={(e) => setSpec(i, { dport: e.target.value })} placeholder="443" className="font-mono text-sm h-8" />
                    </div>
                    <div className="space-y-1">
                      <Label className="text-[11px]">Forward to</Label>
                      <Input value={s.to ?? ''} onChange={(e) => setSpec(i, { to: e.target.value })} placeholder="10.0.0.5:8443" className="font-mono text-sm h-8" />
                    </div>
                  </>
                )}

                <div className="space-y-1">
                  <Label className="text-[11px]">Comment</Label>
                  <Input value={s.comment ?? ''} onChange={(e) => setSpec(i, { comment: e.target.value })} placeholder="optional" className="text-sm h-8" />
                </div>
              </div>
            </div>
          ))}
        </div>
        </>
      )}
    </div>
  )
}
