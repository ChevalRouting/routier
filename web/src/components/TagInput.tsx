import { useState } from 'react'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { X, Plus } from 'lucide-react'

interface TagInputProps {
  values?: string[]
  onChange: (values: string[]) => void
  placeholder?: string
  mono?: boolean
  validate?: (value: string) => string | null
  suggestions?: string[]
}

export default function TagInput({ values = [], onChange, placeholder = 'Add…', mono = false, validate, suggestions }: TagInputProps) {
  const [input, setInput] = useState('')
  const liveError = validate && input.trim() ? validate(input.trim()) : null

  const add = (raw?: string) => {
    const v = (raw ?? input).trim()
    if (!v || (validate && validate(v))) return
    if (!values.includes(v)) {
      onChange([...values, v])
      setInput('')
    }
  }

  const openSuggestions = (suggestions ?? []).filter((s) => !values.includes(s))

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-1.5 min-h-6">
        {values.map((v) => {
          const invalid = validate ? validate(v) : null
          return (
            <Badge key={v} variant={invalid ? 'destructive' : 'secondary'} className="gap-1 pr-1" title={invalid ?? undefined}>
              <span className={mono ? 'font-mono' : ''}>{v}</span>
              <button
                type="button"
                onClick={() => onChange(values.filter((x) => x !== v))}
                className="hover:text-destructive ml-0.5"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          )
        })}
        {values.length === 0 && <span className="text-sm text-muted-foreground italic">None</span>}
      </div>
      <div className="flex gap-2">
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); add() } }}
          placeholder={placeholder}
          aria-invalid={liveError ? true : undefined}
          className={mono ? 'font-mono text-sm' : 'text-sm'}
        />
        <Button type="button" variant="outline" size="sm" onClick={() => add()} disabled={!input.trim() || !!liveError} className="gap-1 shrink-0">
          <Plus className="h-3.5 w-3.5" />Add
        </Button>
      </div>
      {openSuggestions.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {openSuggestions.map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => add(s)}
              className={`text-[11px] px-1.5 py-0.5 rounded bg-muted hover:bg-accent text-foreground ${mono ? 'font-mono' : ''}`}
            >
              + {s}
            </button>
          ))}
        </div>
      )}
      {liveError && <p className="text-xs text-destructive">{liveError}</p>}
    </div>
  )
}
