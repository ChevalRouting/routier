import type { ReactNode } from 'react'
import { VLANChip } from 'cheval-ui'
import type { TypesSwitchPort as SwitchPort } from '@/api'
import { desiredPort, portLink, type SwitchDefinition } from '@/lib/switches'

type PortStatusShape = {
  ports: string[]
  live: SwitchPort[]
  definition: SwitchDefinition | undefined
  pending: string[]
}

type StatusRowShape = { label: string; children: ReactNode }

type PortSummaryShape = { port: SwitchPort | undefined; name: string }

function StatusRow({ label, children }: StatusRowShape) {
  return (
    <div className="flex items-center justify-between gap-4 py-1.5">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="flex flex-wrap justify-end gap-1 text-right">{children}</dd>
    </div>
  )
}

function liveMode(port: SwitchPort): string {
  return port.mode === 'bond-member' ? `member of ${port.bond}` : port.mode
}

function PortSummary({ port, name }: PortSummaryShape) {
  return (
    <div className="flex items-center justify-between gap-4 py-1.5">
      <span className="font-mono">{port?.label ? `${name} (${port.label})` : name}</span>
      <span className="text-muted-foreground">{port ? `${portLink(port)}, ${liveMode(port)}` : 'not reported'}</span>
    </div>
  )
}

export function PortStatus({ ports, live, definition, pending }: PortStatusShape) {
  const byName = new Map(live.map((port) => [port.name, port]))

  if (ports.length > 1) {
    return (
      <div className="divide-y divide-border rounded-md border border-border px-3 text-xs">
        {ports.map((name) => <PortSummary key={name} name={name} port={byName.get(name)} />)}
      </div>
    )
  }

  const port = byName.get(ports[0])

  return (
    <dl className="divide-y divide-border rounded-md border border-border px-3 text-xs">
      {port ? (
        <>
          {port.label && <StatusRow label="Name on Switch"><span className="font-mono">{port.label}</span></StatusRow>}
          <StatusRow label="Link"><span className="tabular-nums">{portLink(port)}</span></StatusRow>
          <StatusRow label="Mode">{liveMode(port)}</StatusRow>
          {port.bridge && <StatusRow label="Untagged"><VLANChip vid={port.pvid} /></StatusRow>}
          {port.tagged.length > 0 && <StatusRow label="Tagged">{port.tagged.map((vid) => <VLANChip key={vid} vid={vid} tagged />)}</StatusRow>}
          {port.comment && <StatusRow label="Comment">{port.comment}</StatusRow>}
        </>
      ) : (
        <StatusRow label="Link">not reported by the switch</StatusRow>
      )}
      <StatusRow label="Configured">{desiredPort(ports[0], definition) || 'not managed'}</StatusRow>
      {pending.includes(ports[0]) && <StatusRow label="On Apply"><span className="text-warning">changes</span></StatusRow>}
    </dl>
  )
}
