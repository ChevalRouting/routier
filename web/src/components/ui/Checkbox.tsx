import type { ButtonHTMLAttributes } from 'react'
import { Check } from 'lucide-react'
import { cn } from '@/lib/utils'

interface CheckboxProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'onChange' | 'onClick'> {
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}

export function Checkbox({ checked, onCheckedChange, className, ...props }: CheckboxProps) {
  const handleClick = () => onCheckedChange(!checked)

  return (
    <button
      {...props}
      type="button"
      role="checkbox"
      aria-checked={checked}
      onClick={handleClick}
      className={cn('inline-flex h-4 w-4 shrink-0 items-center justify-center rounded border border-input focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50', checked && 'border-primary bg-primary text-primary-foreground', className)}
    >
      {checked && <Check className="h-3 w-3" aria-hidden="true" />}
    </button>
  )
}
