import { fmtBitrate } from 'cheval-ui'

export const LIVE_WINDOW = 60

const BAR_WIDTH = 10
const GRAPH_HEIGHT = 120

type ThroughputGraphProps = { values: number[]; color: string; label: string }

export function ThroughputGraph({ values, color, label }: ThroughputGraphProps) {
  const peak = Math.max(0, ...values)
  const scale = peak > 0 ? GRAPH_HEIGHT / peak : 0
  const offset = LIVE_WINDOW - values.length

  return (
    <div className="space-y-1">
      <div className="flex justify-between text-xs text-muted-foreground tabular-nums">
        <span>{fmtBitrate(peak)}</span>
        <span>{LIVE_WINDOW} s</span>
      </div>
      <svg
        role="img"
        aria-label={label}
        viewBox={`0 0 ${LIVE_WINDOW * BAR_WIDTH} ${GRAPH_HEIGHT}`}
        preserveAspectRatio="none"
        className="block h-32 w-full rounded-md bg-muted/40"
      >
        {values.map((value, i) => {
          const height = Math.max(value > 0 ? 1 : 0, value * scale)
          return <rect key={i} x={(offset + i) * BAR_WIDTH} y={GRAPH_HEIGHT - height} width={BAR_WIDTH - 1} height={height} fill={color} />
        })}
      </svg>
    </div>
  )
}
