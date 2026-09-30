import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Button, EntryRow, Label, PageHeader, PreferencesGroup, Spinner, TagInput } from 'cheval-ui'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import type { Internet } from '@/components/simple/types'
import type { TypesSystemNic as SystemNic } from '@/api'

export default function SimpleInternet() {
  const { data, isLoading, reload } = useFetch<Internet | null>(() =>
    api.apiConfigSectionGet({ section: 'internet' }) as Promise<Internet | null>,
  )
  const { data: nics } = useFetch<SystemNic[]>(() => api.apiSystemNicsGet())
  const [value, setValue] = useState<Internet>({ interface: 'wan', select: '', mode: 'dhcp', ipv6: 'slaac' })
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (data) setValue(data)
  }, [data])

  if (isLoading) return <Spinner />

  const update = (patch: Partial<Internet>) => setValue((current) => ({ ...current, ...patch }))
  const save = async () => {
    setSaving(true)
    try {
      await api.apiConfigSectionPut({ section: 'internet', body: value as unknown as object })
      toast.success('Internet configuration staged')
      reload()
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save Internet configuration')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Internet" description="Configure the connection to your modem or upstream network." />
      <div className="max-w-2xl space-y-4">
        <PreferencesGroup title="Connection">
          <div className="flex items-center justify-between gap-4 px-4 py-3">
            <div>
              <p className="text-sm font-medium">Internet port</p>
              <p className="text-xs text-muted-foreground">The physical port connected to the upstream network.</p>
            </div>
            <Select
              value={value.select}
              onValueChange={(select) => update({ select })}
            >
              <SelectTrigger className="w-52"><SelectValue placeholder="Select a port" /></SelectTrigger>
              <SelectContent>
                {!nics?.some((nic) => selectorFor(nic) === value.select) && value.select && <SelectItem value={value.select}>{value.select}</SelectItem>}
                {(nics ?? []).filter((nic) => nic.physical !== false).map((nic) => (
                  <SelectItem key={nic.name} value={selectorFor(nic)}>{nic.name}{nic.carrier ? ' (connected)' : ''}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center justify-between gap-4 px-4 py-3">
            <label className="text-sm font-medium">IPv4 assignment</label>
            <Select value={value.mode} onValueChange={(mode) => update({ mode: mode as Internet['mode'] })}>
              <SelectTrigger className="w-56"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="dhcp">Automatic (DHCP)</SelectItem>
                <SelectItem value="static">Static IPv4</SelectItem>
                <SelectItem value="disabled">Disabled</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center justify-between gap-4 px-4 py-3">
            <label className="text-sm font-medium">IPv6 assignment</label>
            <Select value={value.ipv6} onValueChange={(ipv6) => update({ ipv6: ipv6 as Internet['ipv6'] })}>
              <SelectTrigger className="w-56"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="slaac">Automatic (SLAAC)</SelectItem>
                <SelectItem value="dhcp6">Automatic (DHCPv6)</SelectItem>
                <SelectItem value="static">Static IPv6</SelectItem>
                <SelectItem value="disabled">Disabled</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {(value.mode === 'static' || value.ipv6 === 'static') && (
            <div className="flex items-center justify-between gap-4 px-4 py-3">
              <div>
                <Label>Addresses</Label>
                <p className="text-xs text-muted-foreground">Static IPv4 and IPv6 addresses with prefix.</p>
              </div>
              <div className="w-64 max-w-full">
                <TagInput values={value.addresses ?? []} onChange={(addresses) => update({ addresses })} placeholder="192.0.2.2/24" mono />
              </div>
            </div>
          )}
          {value.mode === 'static' && (
            <EntryRow title="IPv4 gateway" value={value.gateway ?? ''} placeholder="192.0.2.1" onChange={(event) => update({ gateway: event.target.value })} />
          )}
          {value.ipv6 === 'static' && (
            <EntryRow title="IPv6 gateway" value={value.gateway_v6 ?? ''} placeholder="2001:db8::1" onChange={(event) => update({ gateway_v6: event.target.value })} />
          )}
          <div className="flex items-center justify-between gap-4 px-4 py-3">
            <div>
              <Label>DNS servers</Label>
              <p className="text-xs text-muted-foreground">Resolvers the router uses. Add one address at a time.</p>
            </div>
            <div className="w-64 max-w-full">
              <TagInput values={value.dns ?? []} onChange={(dns) => update({ dns })} placeholder="1.1.1.1" mono />
            </div>
          </div>
          {value.vips && value.vips.length > 0 && (
            <div className="flex items-center justify-between gap-4 px-4 py-3">
              <div>
                <p className="text-sm font-medium">Failover addresses</p>
                <p className="text-xs text-muted-foreground">Shared with your other router (VRRP). Managed in Advanced mode.</p>
              </div>
              <div className="text-right font-mono text-sm">{value.vips.join(', ')}</div>
            </div>
          )}
          {value.mode !== 'static' && value.gateway && (
            <div className="flex items-center justify-between gap-4 px-4 py-3">
              <div>
                <p className="text-sm font-medium">Upstream gateway</p>
                <p className="text-xs text-muted-foreground">From the current routing configuration.</p>
              </div>
              <div className="text-right font-mono text-sm">{value.gateway}</div>
            </div>
          )}
        </PreferencesGroup>
        <div className="flex justify-end"><Button onClick={save} disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button></div>
      </div>
    </div>
  )
}

function selectorFor(nic: SystemNic): string {
  return nic.mac ? `mac(${nic.mac})` : `name=${nic.name}`
}
