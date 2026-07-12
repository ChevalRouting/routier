import type { TypesDiffLine as DiffLine } from '@/api'

export type ViewLine = DiffLine | { type: 'hunk'; text: string }

const CONTEXT = 4

export function withContext(raw: DiffLine[]): ViewLine[] {
  const show = new Array(raw.length).fill(false)
  for (let i = 0; i < raw.length; i++) {
    if (raw[i].type !== 'same') {
      const lo = Math.max(0, i - CONTEXT)
      const hi = Math.min(raw.length - 1, i + CONTEXT)
      for (let k = lo; k <= hi; k++) show[k] = true
    }
  }
  const result: ViewLine[] = []
  let i = 0
  while (i < raw.length) {
    if (show[i]) {
      result.push(raw[i])
      i++
    } else {
      let skip = 0
      while (i < raw.length && !show[i]) { skip++; i++ }
      result.push({ type: 'hunk', text: `··· ${skip} unchanged line${skip !== 1 ? 's' : ''} ···` })
    }
  }
  return result
}

export function DiffView({ lines }: { lines: ViewLine[] }) {
  return (
    <pre className="text-xs font-mono leading-relaxed">
      {lines.map((line, idx) => {
        if (line.type === 'hunk') {
          return (
            <div key={idx} className="text-muted-foreground/60 py-0.5 select-none">
              {line.text}
            </div>
          )
        }
        if (line.type === 'add') {
          return (
            <div key={idx} className="bg-green-500/10 text-green-700 dark:text-green-400">
              <span className="select-none text-green-600 dark:text-green-500 mr-1">+</span>
              {line.text}
            </div>
          )
        }
        if (line.type === 'remove') {
          return (
            <div key={idx} className="bg-red-500/10 text-red-700 dark:text-red-400">
              <span className="select-none text-red-600 dark:text-red-500 mr-1">-</span>
              {line.text}
            </div>
          )
        }
        return (
          <div key={idx} className="text-muted-foreground">
            <span className="select-none mr-1"> </span>
            {line.text}
          </div>
        )
      })}
    </pre>
  )
}
