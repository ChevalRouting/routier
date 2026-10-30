import type { ConfigRouteMapEntry, RenderImportedRouteMap } from '@/api'
import { Badge, Button, Dialog, PreferencesGroup, Row } from 'cheval-ui'

type ImportedRouteMapDialogShape = {
  map: RenderImportedRouteMap
  onClose: () => void
}

type ClauseRowsShape = {
  title: string
  clauses: Record<string, string> | undefined
}

type EntryGroupShape = { entry: ConfigRouteMapEntry }

function ClauseRows({ title, clauses }: ClauseRowsShape) {
  return Object.entries(clauses ?? {}).map(([key, value]) => (
    <Row key={`${title}-${key}`} title={`${title} ${key}`}>
      <span className="font-mono text-xs">{value}</span>
    </Row>
  ))
}

function flowOf(entry: ConfigRouteMapEntry): string[] {
  const flow: string[] = []
  if (entry.call) flow.push(`call ${entry.call}`)
  if (entry.on_match) flow.push(`on-match ${entry.on_match}`)
  if (entry._continue) flow.push(`continue ${entry._continue}`)
  return flow
}

function EntryGroup({ entry }: EntryGroupShape) {
  const flow = flowOf(entry)
  const empty = !Object.keys(entry.match ?? {}).length && !Object.keys(entry.set ?? {}).length && flow.length === 0
  const action = <Badge variant="secondary" className="font-mono">{entry.action}</Badge>

  return (
    <PreferencesGroup title={`Sequence ${entry.seq}`} header={action}>
      <ClauseRows title="match" clauses={entry.match} />
      <ClauseRows title="set" clauses={entry.set} />
      {flow.map((step) => <Row key={step} title={step} />)}
      {empty && <Row title={entry.action === 'deny' ? 'Deny Every Route' : 'Permit Every Route'} />}
    </PreferencesGroup>
  )
}

export function ImportedRouteMapDialog({ map, onClose }: ImportedRouteMapDialogShape) {
  const entries = [...map.entries].sort((a, b) => a.seq - b.seq)

  return (
    <Dialog
      open
      onClose={onClose}
      title={`Route Map ${map.name}`}
      description={`Imported from ${map.sources.join(', ')}, edit it on the friend that shares it`}
      className="max-w-2xl"
      footer={<Button variant="outline" onClick={onClose}>Close</Button>}
    >
      <div className="space-y-4">
        {entries.length === 0
          ? <p className="text-sm text-muted-foreground">No entries</p>
          : entries.map((entry) => <EntryGroup key={entry.seq} entry={entry} />)}
      </div>
    </Dialog>
  )
}
