import type { KeyboardEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import { StatusChip, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from 'cheval-ui'
import type { TypesSwitchStatus as SwitchStatus } from '@/api'
import type { SwitchDefinitions } from '@/lib/switches'
import { switchHealth } from './switchStatus'

type SwitchesTableShape = { definitions: SwitchDefinitions; statuses: SwitchStatus[] }

export function SwitchesTable({ definitions, statuses }: SwitchesTableShape) {
  const byName = new Map(statuses.map((status) => [status.name, status]))
  const names = [...new Set([...Object.keys(definitions), ...byName.keys()])].sort()
  const navigate = useNavigate()

  const handleOpen = (name: string) => navigate(`/switches/${encodeURIComponent(name)}`)

  const handleKey = (event: KeyboardEvent, name: string) => {
    if (event.key !== 'Enter' && event.key !== ' ') return
    event.preventDefault()
    handleOpen(name)
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Status</TableHead>
          <TableHead>Switch</TableHead>
          <TableHead>Model</TableHead>
          <TableHead>Uplink</TableHead>
          <TableHead>Last Read</TableHead>
          <TableHead className="w-8" />
        </TableRow>
      </TableHeader>
      <TableBody>
        {names.map((name) => {
          const status = byName.get(name)
          const definition = definitions[name]
          const health = definition ? switchHealth(status) : { intent: 'warning' as const, label: 'removed in draft' }
          return (
            <TableRow key={name} role="link" tabIndex={0} aria-label={`Open ${name}`} className="cursor-pointer" onClick={() => handleOpen(name)} onKeyDown={(event) => handleKey(event, name)}>
              <TableCell><StatusChip intent={health.intent} label={health.label} /></TableCell>
              <TableCell data-title>
                <div className="font-medium">{name}</div>
                <div className="font-mono text-xs text-muted-foreground">{definition?.host ?? status?.host}</div>
              </TableCell>
              <TableCell className="text-xs">{status?.model.name ?? definition?.model}</TableCell>
              <TableCell className="font-mono text-xs">{definition?.interface ?? status?.interface} {definition?.uplink ?? status?.uplink}</TableCell>
              <TableCell className="text-xs text-muted-foreground">{status?.read_at ? new Date(status.read_at * 1000).toLocaleString() : '-'}</TableCell>
              <TableCell><ChevronRight className="h-4 w-4 text-muted-foreground" /></TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}
