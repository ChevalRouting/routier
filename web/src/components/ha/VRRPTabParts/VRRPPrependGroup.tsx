import { VRRPBGPPrepend } from '@/components/ha/types'
import { checkCIDR, checkIP } from '@/lib/validate'
import { ComboRow, EntryRow, PreferencesGroup, SwitchRow, TagInput } from 'cheval-ui'
import { useState } from 'react'

type VRRPPrependGroupShape = {
  prepend?: VRRPBGPPrepend
  onChange: (prepend: VRRPBGPPrepend | undefined) => void
}

const DEFAULT_COUNT = 3

const MAX_COUNT = 10

function countError(text: string): string | null {
  if (!text) return null
  const count = Number(text)
  return /^[0-9]+$/.test(text) && count >= 1 && count <= MAX_COUNT ? null : `Enter a count from 1 to ${MAX_COUNT}`
}

export function VRRPPrependGroup({ prepend, onChange }: VRRPPrependGroupShape) {
  const [countText, setCountText] = useState(prepend?.count ? String(prepend.count) : '')
  const handleEnabled = (on: boolean) => onChange(on ? {} : undefined)
  const handleCount = (e: React.ChangeEvent<HTMLInputElement>) => {
    const text = e.target.value.trim()
    setCountText(text)
    if (countError(text)) return
    const count = Number(text)
    onChange({ ...prepend, count: !text || count === DEFAULT_COUNT ? undefined : count })
  }
  const handleNeighbors = (neighbors: string[]) => onChange({ ...prepend, neighbors: neighbors.length ? neighbors : undefined })
  const handlePrefixes = (prefixes: string[]) => onChange({ ...prepend, prefixes: prefixes.length ? prefixes : undefined })

  return (
    <PreferencesGroup title="BGP While not Master" description="Prepends the local AS while BACKUP or FAULT so peers prefer the master">
      <SwitchRow title="Prepend Routes" checked={!!prepend} onCheckedChange={handleEnabled} />
      {prepend && (
        <>
          <EntryRow title="Prepend Count" inputMode="numeric" value={countText} placeholder={String(DEFAULT_COUNT)} error={countError(countText)} onChange={handleCount} />
          <ComboRow title="Prefixes" subtitle="Empty prepends every route">
            <TagInput values={prepend.prefixes ?? []} onChange={handlePrefixes} placeholder="203.0.113.0/24" mono validate={checkCIDR} />
          </ComboRow>
          <ComboRow title="Neighbors" subtitle="Empty applies to every eBGP neighbor">
            <TagInput values={prepend.neighbors ?? []} onChange={handleNeighbors} placeholder="192.0.2.1" mono validate={checkIP} />
          </ComboRow>
        </>
      )}
    </PreferencesGroup>
  )
}
