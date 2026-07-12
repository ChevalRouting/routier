import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

interface SpinnerProps {
  className?: string
  size?: 'sm' | 'md' | 'lg'
}

const SIZE = { sm: 'h-4 w-4', md: 'h-5 w-5', lg: 'h-6 w-6' }

export function Spinner({ size = 'md', className }: SpinnerProps) {
  return (
    <div className={cn('flex items-center justify-center h-64', className)}>
      <Loader2 className={cn(SIZE[size], 'animate-spin text-muted-foreground/50')} />
    </div>
  )
}
