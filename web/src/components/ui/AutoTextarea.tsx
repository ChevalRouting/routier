import { cn, Textarea } from 'cheval-ui'
import { useLayoutEffect, useRef } from 'react'

interface AutoTextareaProps extends Omit<React.TextareaHTMLAttributes<HTMLTextAreaElement>, 'onChange'> {
  value: string
  onChange: (value: string) => void
}

export function AutoTextarea({ value, onChange, className, onKeyDown, ...props }: AutoTextareaProps) {
  const ref = useRef<HTMLTextAreaElement>(null)

  useLayoutEffect(() => {
    const el = ref.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${el.scrollHeight}px`
  })

  return (
    <Textarea
      ref={ref}
      rows={1}
      value={value}
      onChange={(e) => onChange(e.target.value.replace(/\n/g, ''))}
      onKeyDown={(e) => {
        if (e.key === 'Enter') e.preventDefault()
        onKeyDown?.(e)
      }}
      className={cn('min-h-8 resize-none overflow-hidden break-all px-3 py-1.5 leading-5', className)}
      {...props}
    />
  )
}
