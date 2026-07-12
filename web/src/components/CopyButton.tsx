import { Check, Copy } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useClipboard } from '@/lib/useClipboard'

export function CopyButton({ text, className }: { text: string; className?: string }) {
  const { copied, copy } = useClipboard()
  return (
    <button
      type="button"
      onClick={() => copy(text)}
      className={cn('text-muted-foreground hover:text-foreground transition-colors', className)}
      title="Copy to clipboard"
    >
      {copied ? <Check className="h-3.5 w-3.5 text-success" /> : <Copy className="h-3.5 w-3.5" />}
    </button>
  )
}
