import type { Network, SimpleDNS } from '@/components/simple/types'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { ZoneEditor, type DnsZone } from '@/pages/DnsConfig'
import { AccordionList, PageHeader, PreferencesGroup, SaveButton, Spinner, SwitchRow, TagInput } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

const emptyDNS: SimpleDNS = {
  enabled: false,
  upstreams: [],
  networks: [],
  allow_wan: false,
  wan_allow_from: [],
  dnssec: true,
  cache: true,
  prefetch: true,
  serve_expired: true,
  zones: [],
}

export default function SimpleDns() {
  const { data, isLoading, reload } = useFetch<SimpleDNS>(() => api.apiConfigSectionGet({ section: 'dns' }) as Promise<SimpleDNS>)
  const { data: networks } = useFetch<Network[]>(() => api.apiConfigSectionGet({ section: 'networks' }) as Promise<Network[]>)
  const [value, setValue] = useState<SimpleDNS>(emptyDNS)
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (data) {
      setValue({
        ...emptyDNS,
        ...data,
        upstreams: data.upstreams ?? [],
        networks: data.networks ?? [],
        wan_allow_from: data.wan_allow_from ?? [],
        zones: data.zones ?? [],
      })
      setDirty(false)
    }
  }, [data])

  if (isLoading) return <Spinner />

  const update = (patch: Partial<SimpleDNS>) => {
    setValue((current) => ({ ...current, ...patch }))
    setDirty(true)
  }
  const toggleNetwork = (name: string, enabled: boolean) => update({
    networks: enabled ? [...new Set([...value.networks, name])] : value.networks.filter((network) => network !== name),
  })
  const save = async () => {
    setSaving(true)
    try {
      await api.apiConfigSectionPut({ section: 'dns', body: value as unknown as object })
      toast.success('DNS configuration staged')
      setDirty(false)
      reload()
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save DNS configuration')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader title="DNS" description="Provide secure, cached name resolution to devices on your local networks." />
      <div className="max-w-3xl space-y-5">
        <PreferencesGroup title="DNS service">
          <SwitchRow title="Provide DNS to local devices" subtitle="Run Routier's local forwarding DNS server." checked={value.enabled} onCheckedChange={(enabled) => update({ enabled })} />
        </PreferencesGroup>

        {value.enabled && (
          <>
            <PreferencesGroup title="Upstream resolvers" description="Queries not answered locally are sent to these servers.">
              <div className="px-4 py-3"><TagInput values={value.upstreams} onChange={(upstreams) => update({ upstreams })} placeholder="1.1.1.1" mono /></div>
              <SwitchRow title="Validate DNSSEC" subtitle="Reject responses with invalid DNSSEC signatures." checked={value.dnssec} onCheckedChange={(dnssec) => update({ dnssec })} />
            </PreferencesGroup>

            <PreferencesGroup title="Local networks" description="Only devices on selected networks can query this DNS server.">
              {(networks ?? []).length === 0 && <p className="px-4 py-3 text-sm text-muted-foreground">Create a local network before enabling DNS.</p>}
              {(networks ?? []).map((network) => (
                <SwitchRow
                  key={network.id || network.name}
                  title={network.name}
                  subtitle={network.addresses.join(', ') || 'No router address'}
                  checked={value.networks.includes(network.name)}
                  onCheckedChange={(enabled) => toggleNetwork(network.name, enabled)}
                />
              ))}
            </PreferencesGroup>

            <PreferencesGroup title="WAN access" description="Expose DNS on the Internet interface only to explicitly allowed sources.">
              <SwitchRow title="Allow DNS queries from WAN" subtitle="Use this for trusted remote resolvers or monitoring systems. Never allow the whole Internet." checked={value.allow_wan} onCheckedChange={(allow_wan) => update({ allow_wan })} />
              {value.allow_wan && (
                <div className="space-y-2 px-4 py-3">
                  <div>
                    <p className="text-sm font-medium">Allowed source addresses</p>
                    <p className="text-xs text-muted-foreground">Only these public IP addresses or CIDR ranges can query port 53 on WAN.</p>
                  </div>
                  <TagInput values={value.wan_allow_from} onChange={(wan_allow_from) => update({ wan_allow_from })} placeholder="203.0.113.10/32" mono />
                </div>
              )}
            </PreferencesGroup>

            <PreferencesGroup title="Cache">
              <SwitchRow title="Cache responses" subtitle="Reuse recent answers to reduce latency and upstream traffic." checked={value.cache} onCheckedChange={(cache) => update({ cache })} />
              <SwitchRow title="Prefetch popular records" subtitle="Refresh frequently used answers before they expire." checked={value.prefetch} disabled={!value.cache} onCheckedChange={(prefetch) => update({ prefetch })} />
              <SwitchRow title="Serve expired records" subtitle="Keep names working temporarily when upstream resolvers are unavailable." checked={value.serve_expired} disabled={!value.cache} onCheckedChange={(serve_expired) => update({ serve_expired })} />
            </PreferencesGroup>

            <PreferencesGroup title="Hosted DNS zones" description="Serve authoritative local or public zones using the same zone model as Advanced mode.">
              <div className="p-3 sm:p-4">
                <AccordionList
                  items={value.zones}
                  getId={(zone, index) => `${zone.name || 'new'}-${index}`}
                  addLabel="Add zone"
                  onAdd={() => update({ zones: [...value.zones, { name: '' }] })}
                  emptyTitle="No hosted zones"
                  emptyMessage="Add a zone to host authoritative DNS records on this router."
                  renderSummary={(zone) => <span className="font-mono text-sm">{zone.name || 'New zone'}</span>}
                  onRemove={(zone) => update({ zones: value.zones.filter((item) => item !== zone) })}
                  renderBody={(zone) => <ZoneEditor zone={zone} onChange={(next: DnsZone) => update({ zones: value.zones.map((item) => item === zone ? next : item) })} onRemove={() => update({ zones: value.zones.filter((item) => item !== zone) })} zones={value.zones} onZonesChange={(next: DnsZone[]) => update({ zones: next })} />}
                />
              </div>
            </PreferencesGroup>
          </>
        )}

        <div className="flex justify-end">
          <SaveButton isDirty={dirty} saving={saving} onClick={save} onCancel={() => {
            setValue({
              ...emptyDNS,
              ...data,
              upstreams: data?.upstreams ?? [],
              networks: data?.networks ?? [],
              wan_allow_from: data?.wan_allow_from ?? [],
              zones: data?.zones ?? [],
            })
            setDirty(false)
          }} />
        </div>
      </div>
    </div>
  )
}
