import { Card, CardContent, CardHeader, CardTitle, fmtBitrate, fmtBytes, fmtPps } from 'cheval-ui'
import { ThroughputGraph } from './ThroughputGraph'

export type DirectionSample = { bps: number; pps: number; total: number }

type DirectionCardProps = { title: string; color: string; samples: DirectionSample[] }

type DirectionFigure = { label: string; value: string }

function figures(samples: DirectionSample[]): DirectionFigure[] {
  const rates = samples.map((s) => s.bps)
  const last = samples[samples.length - 1]
  const average = rates.length > 0 ? rates.reduce((sum, r) => sum + r, 0) / rates.length : 0
  return [
    { label: 'Current', value: fmtBitrate(last?.bps ?? 0) },
    { label: 'Average', value: fmtBitrate(average) },
    { label: 'Minimum', value: fmtBitrate(rates.length > 0 ? Math.min(...rates) : 0) },
    { label: 'Maximum', value: fmtBitrate(rates.length > 0 ? Math.max(...rates) : 0) },
    { label: 'Packets', value: fmtPps(last?.pps ?? 0) },
    { label: 'Total', value: fmtBytes(last?.total ?? 0) },
  ]
}

export function DirectionCard({ title, color, samples }: DirectionCardProps) {
  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-sm">
          <span className="h-2 w-2 rounded-full" style={{ background: color }} />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <ThroughputGraph values={samples.map((s) => s.bps)} color={color} label={`${title} throughput`} />
        <dl className="grid grid-cols-3 gap-x-4 gap-y-3">
          {figures(samples).map((figure) => (
            <div key={figure.label} className="min-w-0">
              <dt className="text-xs text-muted-foreground">{figure.label}</dt>
              <dd className="truncate text-sm font-medium tabular-nums">{figure.value}</dd>
            </div>
          ))}
        </dl>
      </CardContent>
    </Card>
  )
}
