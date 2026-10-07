import { Card, CardContent, CopyButton, Input, Label } from 'cheval-ui'
import { useEffect, useState } from 'react'

type BinaryRowShape = { bits: string; prefix: number; is4: boolean }

type ResultTableShape = { rows: CalcRow[]; prefix: number; is4: boolean }

type ToolInputShape = { label: string; hint?: string }

type ErrorNoteShape = { message: string }

type TABSShape = { key: Tab; label: string }

export type Tab = 'subnet' | 'reverse' | 'range'

export function useDebounced<T>(value: T, delay = 250): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(id)
  }, [value, delay])
  return debounced
}

export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : 'invalid input'
}

export function formatBits(raw: string, prefix: number, is4: boolean): [string, string | null] {
  const group = is4 ? 8 : 16
  let out = ''
  let splitAt = -1
  for (let i = 0; i < raw.length; i++) {
    if (i > 0 && i % group === 0) out += '.'
    if (i === prefix) {
      splitAt = out.length
      out += ' '
    }
    out += raw[i]
  }
  if (splitAt < 0) return [out, null]
  return [out.slice(0, splitAt), out.slice(splitAt + 1)]
}

export function BinaryRow({ bits, prefix, is4 }: BinaryRowShape) {
  const [net, host] = formatBits(bits, prefix, is4)
  return (
    <span className="font-mono text-xs whitespace-pre">
      <span className="text-primary">{net}</span>
      {host !== null && <span className="text-muted-foreground">{' ' + host}</span>}
    </span>
  )
}

export interface CalcRow {
  label: string
  value: string
  bits?: string
}

export function ResultTable({ rows, prefix, is4 }: ResultTableShape) {
  return (
    <Card>
      <CardContent className="overflow-x-auto p-0">
        <table className="w-full text-sm">
          <tbody>
            {rows.map((row) => (
              <tr key={row.label} className="border-b border-border last:border-0">
                <td className="py-2 pl-4 pr-3 align-top font-medium text-muted-foreground w-28 whitespace-nowrap">{row.label}</td>
                <td className="py-2 pr-3 align-top font-mono whitespace-nowrap">
                  <span className="inline-flex items-center gap-1.5">
                    {row.value}
                    <CopyButton text={row.value} />
                  </span>
                </td>
                <td className="py-2 pr-4 align-top">
                  {row.bits && <BinaryRow bits={row.bits} prefix={prefix} is4={is4} />}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </CardContent>
    </Card>
  )
}

export function ToolInput({ label, hint, ...props }: ToolInputShape & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs">{label}</Label>
      <Input className="font-mono" spellCheck={false} autoComplete="off" {...props} />
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  )
}

export function ErrorNote({ message }: ErrorNoteShape) {
  return <p className="text-sm text-destructive">{message}</p>
}

export const TABS: TABSShape[] = [
  { key: 'subnet', label: 'Subnet calculator' },
  { key: 'reverse', label: 'Reverse DNS' },
  { key: 'range', label: 'Range to CIDR' },
]
