import { useMemo } from 'react'
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer,
} from 'recharts'
import { cn } from '@/lib/utils'

export interface ChartSeries {
  dataKey: string
  label: string
  color: string
}

interface Props {
  data: Array<Record<string, number | null>>
  series: ChartSeries[]
  height?: number
  yFormatter?: (v: number) => string
  className?: string
  mini?: boolean
}

function fmtTick(ts: number, range: number): string {
  const d = new Date(ts * 1000)
  if (range <= 7200) return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  if (range <= 172800) return d.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

function fmtFull(ts: number): string {
  return new Date(ts * 1000).toLocaleString([], {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
}

function CustomTooltip({ active, payload, label, yFormatter }: {
  active?: boolean
  payload?: Array<{ dataKey: string; name: string; value: number | null; color: string }>
  label?: number
  yFormatter?: (v: number) => string
}) {
  if (!active || !payload?.length) return null
  return (
    <div className="bg-popover text-popover-foreground border border-border rounded-lg shadow-lg px-3 py-2 text-xs space-y-1.5">
      <p className="text-muted-foreground">{label != null ? fmtFull(label) : ''}</p>
      {payload.map((entry) => (
        <div key={entry.dataKey} className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full shrink-0" style={{ background: entry.color }} />
          <span className="text-muted-foreground">{entry.name}:</span>
          <span className="font-medium tabular-nums ml-auto">
            {entry.value != null ? (yFormatter ? yFormatter(entry.value) : entry.value) : '-'}
          </span>
        </div>
      ))}
    </div>
  )
}

export function TimeSeriesChart({ data, series, height, yFormatter, className, mini = false }: Props) {
  const range = useMemo(() => {
    if (data.length < 2) return 3600
    const first = data[0].ts as number
    const last = data[data.length - 1].ts as number
    return last - first
  }, [data])

  const filtered = useMemo(
    () => data.filter((d) => series.some((s) => d[s.dataKey] != null)),
    [data, series],
  )

  const yAxisWidth = useMemo(() => {
    if (mini || !yFormatter) return 48
    const maxVal = Math.max(0, ...filtered.flatMap((d) =>
      series.map((s) => (d[s.dataKey] as number | null) ?? 0),
    ))
    const label = yFormatter(maxVal)
    try {
      const ctx = document.createElement('canvas').getContext('2d')!
      ctx.font = '10px system-ui'
      return Math.ceil(ctx.measureText(label).width) + 12
    } catch {
      return Math.max(48, label.length * 7)
    }
  }, [filtered, series, yFormatter, mini])

  const sizeClass = height == null ? 'h-full' : ''

  if (filtered.length === 0) {
    return (
      <div
        className={cn('flex items-center justify-center text-xs text-muted-foreground/60', sizeClass, className)}
        style={height != null ? { height } : undefined}
      >
        No data collected yet
      </div>
    )
  }

  const margin = mini
    ? { top: 0, right: 0, bottom: 0, left: 0 }
    : { top: 12, right: 2, bottom: 0, left: 0 }

  return (
    <div className={cn('w-full min-w-0', sizeClass, className)} style={height != null ? { height } : undefined}>
      <ResponsiveContainer width="100%" height={height ?? '100%'} minWidth={0}>
        <AreaChart data={filtered} margin={margin}>
          <defs>
            {series.map((s) => (
              <linearGradient key={s.dataKey} id={`tsc-${s.dataKey}`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%"  stopColor={s.color} stopOpacity={mini ? 0.25 : 0.18} />
                <stop offset="95%" stopColor={s.color} stopOpacity={0} />
              </linearGradient>
            ))}
          </defs>

          {!mini && (
            <CartesianGrid
              strokeDasharray="3 3"
              stroke="currentColor"
              strokeOpacity={0.07}
              vertical={false}
            />
          )}

          <XAxis
            dataKey="ts"
            tickFormatter={(v) => fmtTick(v as number, range)}
            tick={{ fontSize: 10, fill: 'currentColor', opacity: 0.45 }}
            axisLine={false}
            tickLine={false}
            minTickGap={80}
            hide={mini}
          />

          <YAxis
            tickFormatter={yFormatter}
            tick={{ fontSize: 10, fill: 'currentColor', opacity: 0.45 }}
            axisLine={false}
            tickLine={false}
            width={yAxisWidth}
            tickCount={4}
            hide={mini}
          />

          {!mini && (
            <Tooltip
              content={<CustomTooltip yFormatter={yFormatter} />}
              cursor={{ stroke: 'currentColor', strokeOpacity: 0.08, strokeWidth: 1 }}
            />
          )}

          {series.map((s) => (
            <Area
              key={s.dataKey}
              type="monotone"
              dataKey={s.dataKey}
              name={s.label}
              stroke={s.color}
              strokeWidth={mini ? 1 : 1.5}
              fill={`url(#tsc-${s.dataKey})`}
              dot={false}
              activeDot={mini ? false : { r: 3, strokeWidth: 0, fill: s.color }}
              connectNulls={false}
            />
          ))}
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
