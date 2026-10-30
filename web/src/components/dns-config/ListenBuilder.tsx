import { TagInput } from 'cheval-ui'

type ListenBuilderShape = {
  value: string[]
  options: string[]
  onChange: (v: string[]) => void
}

export function ListenBuilder({ value, options, onChange }: ListenBuilderShape) {
  return <TagInput values={value} onChange={onChange} suggestions={options} placeholder="iface(eth0)" mono />
}
