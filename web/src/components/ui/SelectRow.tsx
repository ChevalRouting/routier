import { ComboRow, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'

const NONE = '__none__'

interface SelectRowProps {
  title: string
  subtitle?: string
  value?: string
  onChange: (value: string | undefined) => void
  options: string[]
  noneLabel?: string
  placeholder?: string
  mono?: boolean
}

export function SelectRow({ title, subtitle, value, onChange, options, noneLabel, placeholder, mono = true }: SelectRowProps) {
  return (
    <ComboRow title={title} subtitle={subtitle}>
      <Select value={value || (noneLabel ? NONE : '')} onValueChange={(v) => onChange(v === NONE ? undefined : v)}>
        <SelectTrigger className={mono ? 'font-mono' : undefined}><SelectValue placeholder={placeholder} /></SelectTrigger>
        <SelectContent>
          {noneLabel && <SelectItem value={NONE}>{noneLabel}</SelectItem>}
          {options.map((option) => <SelectItem key={option} value={option}>{option}</SelectItem>)}
        </SelectContent>
      </Select>
    </ComboRow>
  )
}
