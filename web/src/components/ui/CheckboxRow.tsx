import { useId } from 'react'
import { Row } from 'cheval-ui'
import { Checkbox } from '@/components/ui/Checkbox'

type CheckboxRowProps = {
  title: string
  subtitle?: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  disabled?: boolean
}

export function CheckboxRow({ title, subtitle, ...props }: CheckboxRowProps) {
  const id = useId()

  return (
    <Row title={<label htmlFor={id}>{title}</label>} subtitle={subtitle}>
      <Checkbox id={id} aria-label={title} {...props} />
    </Row>
  )
}
