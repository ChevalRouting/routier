import { SectionCard as SharedSectionCard } from 'cheval-ui'

type ConfigSectionsShape = {
  data: Record<string, unknown> | null | undefined
  defaultOpen?: boolean
}

export { ValueNode } from 'cheval-ui'

export function SectionCard(props: React.ComponentProps<typeof SharedSectionCard>) {
  return <SharedSectionCard {...props} hideNullValues />
}

const SECTION_ORDER = [
  'version', 'hostname', 'interfaces', 'tunnels', 'routing', 'wireguard',
  'nftables', 'sysctl', 'dns', 'users', 'services', 'logging',
]

export function orderedSections(data: Record<string, unknown> | null | undefined): [string, unknown][] {
  if (!data) return []
  return [
    ...SECTION_ORDER.filter((k) => k in data),
    ...Object.keys(data).filter((k) => !SECTION_ORDER.includes(k)),
  ].map((k) => [k, data[k]] as [string, unknown])
}

export function ConfigSections({
  data,
  defaultOpen = true,
}: ConfigSectionsShape) {
  const sections = orderedSections(data)
  if (sections.length === 0) return <div className="text-sm text-muted-foreground">empty</div>
  return (
    <div className="md:columns-2 [column-gap:0.75rem]">
      {sections.map(([k, v]) => (
        <div key={k} className="mb-3 break-inside-avoid">
          <SectionCard name={k} value={v} defaultOpen={defaultOpen} />
        </div>
      ))}
    </div>
  )
}
