import { Label, TagInput } from 'cheval-ui'

type ExtraDirectivesShape = {
  label?: string
  values?: string[]
  onChange: (v: string[] | undefined) => void
  placeholder?: string
}

export function ExtraDirectives({ label = 'Additional directives', values, onChange, placeholder }: ExtraDirectivesShape) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <TagInput
        values={values ?? []}
        onChange={(v) => onChange(v.length ? v : undefined)}
        placeholder={placeholder ?? 'e.g. timers throttle spf 200 400 2000'}
        mono
      />
      <p className="text-[11px] text-muted-foreground">Raw FRR directives, rendered verbatim into this block.</p>
    </div>
  )
}
