import { PreferencesGroup, TagInput } from 'cheval-ui'

type ExtraDirectivesShape = {
  label?: string
  values?: string[]
  onChange: (v: string[] | undefined) => void
  placeholder?: string
}

export function ExtraDirectives({ label = 'Additional Directives', values, onChange, placeholder }: ExtraDirectivesShape) {
  return (
    <PreferencesGroup title={label} description="Rendered verbatim">
      <div className="px-4 py-3">
        <TagInput
          values={values ?? []}
          onChange={(v) => onChange(v.length ? v : undefined)}
          placeholder={placeholder ?? 'timers throttle spf 200 400 2000'}
          mono
        />
      </div>
    </PreferencesGroup>
  )
}
