import { ComboRow, NumberInput } from 'cheval-ui'

interface NumberRowProps {
  title: string
  subtitle?: string
  value?: number
  onChange: (value: number | undefined) => void
  placeholder?: string
}

export function NumberRow({ title, subtitle, value, onChange, placeholder }: NumberRowProps) {
  return (
    <ComboRow title={title} subtitle={subtitle}>
      <NumberInput value={value || undefined} onChange={onChange} placeholder={placeholder} className="font-mono" />
    </ComboRow>
  )
}
