import { useState } from 'react'
import { Input } from '@/components/ui/input'

interface VarPickerInputProps {
  value: string
  onChange: (v: string) => void
  vars: string[]
  placeholder?: string
  mono?: boolean
  className?: string
  wrapperClassName?: string
  prefix?: string
  label?: string
}

export function VarPickerInput({
  value, onChange, vars, placeholder, mono, className, wrapperClassName, prefix = '$', label = 'Available variables',
}: VarPickerInputProps) {
  const [open, setOpen] = useState(false)
  return (
    <div className={`relative ${wrapperClassName ?? ''}`}>
      <Input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onFocus={() => vars.length > 0 && setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        placeholder={placeholder}
        className={`${mono ? 'font-mono' : ''} text-sm ${className ?? ''}`}
      />
      {open && (
        <div className="absolute top-full mt-1 left-0 z-50 bg-popover border rounded-md shadow-lg p-2 max-h-44 overflow-auto w-max max-w-xs">
          <p className="text-[10px] text-muted-foreground mb-1.5 px-0.5">{label}</p>
          <div className="flex flex-wrap gap-1">
            {vars.map((v) => (
              <button
                key={v}
                type="button"
                onMouseDown={(e) => { e.preventDefault(); onChange(prefix + v); setOpen(false) }}
                className="text-[11px] font-mono px-1.5 py-0.5 rounded bg-muted hover:bg-accent text-foreground"
              >
                {prefix}{v}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
