import { Textarea } from 'cheval-ui'
import type { ReactNode } from 'react'

type PemFieldShape = {
  id: string
  label: string
  value: string
  placeholder: string
  help?: ReactNode
  onChange: (value: string) => void
}

export function PemField({ id, label, value, placeholder, help, onChange }: PemFieldShape) {
  return (
    <div className="flex min-w-0 flex-col gap-1.5 px-4 py-3">
      <label htmlFor={id} className="text-sm font-medium">{label}</label>
      <Textarea
        id={id}
        value={value}
        rows={5}
        spellCheck={false}
        placeholder={placeholder}
        className="font-mono text-xs"
        onChange={(event) => onChange(event.target.value)}
      />
      {help && <p className="text-xs text-muted-foreground">{help}</p>}
    </div>
  )
}
