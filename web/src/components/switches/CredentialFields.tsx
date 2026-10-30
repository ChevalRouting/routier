import { EntryRow } from 'cheval-ui'
import type { TypesCredentialField as CredentialField } from '@/api'

type CredentialFieldsShape = {
  fields: CredentialField[]
  values: Record<string, string>
  onChange: (key: string, value: string) => void
}

export function CredentialFields({ fields, values, onChange }: CredentialFieldsShape) {
  return (
    <>
      {fields.map((field) => (
        <EntryRow
          key={field.key}
          title={field.label}
          type={field.type === 'password' ? 'password' : 'text'}
          value={values[field.key] ?? ''}
          placeholder={field.default_value}
          help={field.help || undefined}
          onChange={(e) => onChange(field.key, e.target.value)}
        />
      ))}
    </>
  )
}

export function credentialDefaults(fields: CredentialField[]): Record<string, string> {
  return Object.fromEntries(fields.filter((field) => field.type !== 'password' && field.default_value).map((field) => [field.key, field.default_value]))
}
