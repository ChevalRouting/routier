import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { Trash2 } from 'lucide-react'
import { useState } from 'react'

type ListenBuilderShape = {
  value: string[]
  options: string[]
  onChange: (v: string[]) => void
}

export function ListenBuilder({ value, options, onChange }: ListenBuilderShape) {
  const [pick, setPick] = useState('')

  const add = (entry: string) => {
    if (!entry || value.includes(entry)) return
    onChange([...value, entry])
    setPick('')
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-2">
        {value.map((v) => (
          <span key={v} className="flex items-center gap-1 rounded-md bg-muted/40 px-2 py-1 font-mono text-xs">
            {v}
            <button type="button" onClick={() => onChange(value.filter((x) => x !== v))} className="text-muted-foreground hover:text-foreground">
              <Trash2 className="h-3 w-3" />
            </button>
          </span>
        ))}
        {value.length === 0 && <span className="text-xs text-muted-foreground">No listen address set</span>}
      </div>
      <div className="flex gap-2">
        <Select value={pick} onValueChange={add}>
          <SelectTrigger className="h-8 w-64 text-xs">
            <SelectValue placeholder="Add an interface reference" />
          </SelectTrigger>
          <SelectContent>
            {options.filter((o) => !value.includes(o)).map((o) => (
              <SelectItem key={o} value={o} className="font-mono text-xs">{o}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <p className="text-xs text-muted-foreground">
        <code>iface(name)</code> and <code>vips(name)</code> resolve at render time, so they survive an interface being renumbered.
      </p>
    </div>
  )
}
