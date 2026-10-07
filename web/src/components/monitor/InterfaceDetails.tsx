import type { SystemInterfaceStatus } from '@/api'
import { api } from '@/lib/client'
import { Card, CardContent, CardHeader, CardTitle, Dialog, Spinner } from 'cheval-ui'
import { Fragment, useEffect, useState } from 'react'

type ValueShape = { value: unknown }

type FieldsShape = { fields: Record<string, unknown> }

type SectionShape = { title: string; fields: Record<string, unknown> }

type InterfaceDetailsShape = { name: string; onClose: () => void }

type InterfaceStatusShape = { name: string }

const LABELS: Record<string, string> = {
  ifname: 'Interface', ifindex: 'Index', mtu: 'MTU', operstate: 'Operational state',
  address: 'MAC address', broadcast: 'Broadcast', master: 'Master', link: 'Parent link',
  link_index: 'Parent index', link_type: 'Link type', qdisc: 'Queue discipline',
  txqlen: 'TX queue length', flags: 'Flags', group: 'Group', promiscuity: 'Promiscuity',
  allmulti: 'All multicast', num_tx_queues: 'TX queues', num_rx_queues: 'RX queues',
  info_kind: 'Type', info_slave_kind: 'Member type', mii_status: 'Link status',
  lacp: 'LACP', lacp_rate: 'LACP rate', active_aggregator: 'Active aggregator',
  aggregator_id: 'Aggregator ID', xmit_hash_policy: 'Transmit hash policy',
  actor_churn: 'Actor churn', partner_churn: 'Partner churn', link_failures: 'Link failures',
  speed: 'Speed (Mbps)', duplex: 'Duplex', rx: 'RX', tx: 'TX',
}

function label(key: string): string {
  return LABELS[key] ?? key.replace(/_/g, ' ').replace(/^./, (char) => char.toUpperCase())
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function Value({ value }: ValueShape) {
  if (isRecord(value)) return <Fields fields={value} />
  if (Array.isArray(value)) {
    if (value.every((item) => !isRecord(item) && !Array.isArray(item))) return <>{value.map(String).join(', ') || '–'}</>
    return <div className="space-y-2">{value.map((item, index) => <div key={index}><Value value={item} /></div>)}</div>
  }
  return <>{value === null || value === undefined || value === '' ? '–' : typeof value === 'boolean' ? (value ? 'Yes' : 'No') : String(value)}</>
}

function Fields({ fields }: FieldsShape) {
  return <dl className="grid grid-cols-[minmax(7rem,auto)_minmax(0,1fr)] gap-x-4 gap-y-2 text-xs">
    {Object.entries(fields).map(([key, value]) => <Fragment key={key}>
      <dt className="text-muted-foreground break-words">{label(key)}</dt>
      <dd className="font-mono break-words min-w-0"><Value value={value} /></dd>
    </Fragment>)}
  </dl>
}

function Section({ title, fields }: SectionShape) {
  if (Object.keys(fields).length === 0) return null
  return <Card><CardHeader><CardTitle className="text-sm">{title}</CardTitle></CardHeader><CardContent><Fields fields={fields} /></CardContent></Card>
}

export function InterfaceDetails({ name, onClose }: InterfaceDetailsShape) {
  return <Dialog open onClose={onClose} title={`${name}, Interface status`} className="max-w-3xl"
    description="Live kernel state · refreshes every 5 seconds"
  >
    <InterfaceStatus key={name} name={name} />
  </Dialog>
}

function InterfaceStatus({ name }: InterfaceStatusShape) {
  const [data, setData] = useState<SystemInterfaceStatus | null>(null)
  const [error, setError] = useState('')
  useEffect(() => {
    let stopped = false
    const controller = new AbortController()
    let timer: ReturnType<typeof setTimeout>
    const refresh = async () => {
      try {
        const result = await api.apiSystemInterfacesNameGet({ name }, { signal: controller.signal })
        if (!stopped) { setData(result); setError('') }
      } catch (err) {
        if (!stopped) setError(err instanceof Error ? err.message : 'Failed to load interface status')
      } finally {
        if (!stopped) timer = setTimeout(refresh, 5000)
      }
    }
    void refresh()
    return () => { stopped = true; controller.abort(); clearTimeout(timer) }
  }, [name])

  const { linkinfo, stats64, stats, ...link } = data?.link ?? {}
  const info = isRecord(linkinfo) ? linkinfo : {}
  const { info_data, info_slave_data, ...kind } = info
  const counters = stats64 ?? stats
  const bond = data?.bond
  const { slaves, ...bondStatus } = bond ?? {}

  return <div className="space-y-4">
      {error && <p role="alert" className="text-sm text-destructive">{error}{data ? ', showing last successful update.' : ''}</p>}
      {!data && !error && <Spinner />}
      {data && <>
        <Section title="Link status" fields={{ ...link, ...kind }} />
        {data.addr && <Card><CardHeader><CardTitle className="text-sm">Addresses</CardTitle></CardHeader>
          <CardContent><pre className="overflow-x-auto whitespace-pre-wrap break-words rounded-md bg-muted p-3 font-mono text-xs">{data.addr.trimEnd()}</pre></CardContent></Card>}
        {isRecord(info_data) && <Section title={`${String(info.info_kind ?? 'Interface')} details`} fields={info_data} />}
        {isRecord(info_slave_data) && <Section title={`${String(info.info_slave_kind ?? 'Interface')} member details`} fields={info_slave_data} />}
        {bond && <>
          <Section title={`Bonding · ${bond.healthy ? 'Healthy' : 'Degraded'}`} fields={bondStatus} />
          {(slaves ?? []).map((member) => <Section key={member.name} title={`Member ${member.name}`} fields={{ ...member }} />)}
        </>}
        {isRecord(counters) && <Section title="Counters" fields={counters} />}
        <details className="text-xs"><summary className="cursor-pointer text-muted-foreground">Raw link details</summary><pre className="mt-2 overflow-x-auto rounded-md bg-muted p-3">{JSON.stringify(data.link, null, 2)}</pre></details>
      </>}
    </div>
}
