import { useState, useEffect, useRef } from 'react'
import { Input } from './input'

type InputProps = React.ComponentProps<typeof Input>

interface NumberInputProps extends Omit<InputProps, 'value' | 'onChange' | 'type'> {
  value?: number | null
  onChange: (value: number | undefined) => void
}

export function NumberInput({ value, onChange, ...props }: NumberInputProps) {
  const [text, setText] = useState(() => (value != null ? String(value) : ''))
  const lastExternal = useRef(value)

  useEffect(() => {
    if (lastExternal.current !== value) {
      lastExternal.current = value
      setText(value != null ? String(value) : '')
    }
  }, [value])

  return (
    <Input
      {...props}
      type="text"
      inputMode="numeric"
      value={text}
      onChange={(e) => {
        const s = e.target.value
        setText(s)
        const n = parseFloat(s)
        onChange(s === '' || isNaN(n) ? undefined : n)
      }}
    />
  )
}
