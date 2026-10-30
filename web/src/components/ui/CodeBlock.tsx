import { cn } from 'cheval-ui'
import { forwardRef } from 'react'

type CodeBlockProps = React.HTMLAttributes<HTMLPreElement>

export const CodeBlock = forwardRef<HTMLPreElement, CodeBlockProps>(({ className, ...props }, ref) => (
  <pre
    ref={ref}
    className={cn('m-0 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted p-3 font-mono text-xs text-foreground', className)}
    {...props}
  />
))

CodeBlock.displayName = 'CodeBlock'
