import type { TlscertLogEntry as LogEntry } from '@/api'
import { useEffect, useRef } from 'react'

type CertificateLogShape = { entries: LogEntry[] }

function lineOf(entry: LogEntry) {
  const domain = entry.domain ? ` ${entry.domain}` : ''
  return `${new Date(entry.time).toLocaleString()}${domain}  ${entry.message}`
}

export function CertificateLog({ entries }: CertificateLogShape) {
  const box = useRef<HTMLPreElement>(null)

  useEffect(() => {
    if (box.current) box.current.scrollTop = box.current.scrollHeight
  }, [entries.length])

  return (
    <section className="space-y-2" aria-labelledby="certificate-log-title">
      <h2 id="certificate-log-title" className="px-1 text-sm font-semibold">Log</h2>
      <pre ref={box} className="max-h-[32rem] overflow-auto whitespace-pre-wrap break-words rounded-xl bg-card p-4 font-mono text-xs [box-shadow:var(--card-shadow)]">
        {entries.length === 0 && 'No ACME activity yet'}
        {entries.map((entry, index) => (
          <div key={index} className={entry.level === 'error' ? 'text-danger' : undefined}>{lineOf(entry)}</div>
        ))}
      </pre>
    </section>
  )
}
