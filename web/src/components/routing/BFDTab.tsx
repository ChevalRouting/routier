import { EntityRow } from '@/components/ui/EntityRow'
import { Badge, Button, EmptyState, EntryRow, PreferencesGroup, Sheet, SwitchRow } from 'cheval-ui'
import { Plus } from 'lucide-react'
import React, { useState } from 'react'
import { BFDProfile } from './types'

type BFDProfileFormShape = {
  profile: BFDProfile
  onChange: (v: BFDProfile) => void
  onAdd?: () => void
  onDone: () => void
}

type BFDProfileRowShape = {
  profile: BFDProfile
  onEdit: () => void
  onDelete: () => void
}

type BFDTabShape = {
  profiles: BFDProfile[]
  setProfiles: (v: BFDProfile[]) => void
  onDirty: () => void
}

export function emptyBFDProfile(): BFDProfile {
  return { name: '' }
}

export function BFDProfileForm({
  profile, onChange, onAdd, onDone,
}: BFDProfileFormShape) {
  const set = (patch: Partial<BFDProfile>) => onChange({ ...profile, ...patch })

  const numField = (v: number | undefined) => v ?? ''
  const onNum = (key: keyof BFDProfile) => (e: React.ChangeEvent<HTMLInputElement>) =>
    set({ [key]: e.target.value ? Number(e.target.value) : undefined })

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        <EntryRow title="Profile Name" autoFocus value={profile.name} onChange={(e) => set({ name: e.target.value })} placeholder="fast" className="font-mono" />
        <EntryRow title="Detect Multiplier" value={numField(profile.detect_multiplier)} onChange={onNum('detect_multiplier')} placeholder="3" className="font-mono" />
        <EntryRow title="Minimum TTL" value={numField(profile.minimum_ttl)} onChange={onNum('minimum_ttl')} placeholder="254" className="font-mono" />
        <EntryRow title="Receive Interval (ms)" value={numField(profile.receive_interval)} onChange={onNum('receive_interval')} placeholder="300" className="font-mono" />
        <EntryRow title="Transmit Interval (ms)" value={numField(profile.transmit_interval)} onChange={onNum('transmit_interval')} placeholder="300" className="font-mono" />
        <SwitchRow title="Echo Mode" checked={!!profile.echo_mode} onCheckedChange={(v) => set({ echo_mode: v })} />
        <SwitchRow title="Passive Mode" checked={!!profile.passive_mode} onCheckedChange={(v) => set({ passive_mode: v })} />
      </PreferencesGroup>

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Close'}</Button>
        {onAdd && <Button onClick={onAdd}>Add Profile</Button>}
      </div>
    </div>
  )
}

export function BFDProfileRow({
  profile, onEdit, onDelete,
}: BFDProfileRowShape) {
  const timers: string[] = []
  if (profile.detect_multiplier) timers.push(`×${profile.detect_multiplier}`)
  if (profile.receive_interval) timers.push(`rx ${profile.receive_interval}ms`)
  if (profile.transmit_interval) timers.push(`tx ${profile.transmit_interval}ms`)
  return (
    <EntityRow title={profile.name} onOpen={onEdit} onRemove={onDelete} removeLabel="Remove Profile">
      {timers.length > 0
        ? timers.map((t) => <Badge key={t} variant="outline" className="font-mono">{t}</Badge>)
        : <span className="text-xs text-muted-foreground">Default timers</span>}
      {profile.echo_mode && <Badge variant="secondary">echo</Badge>}
      {profile.passive_mode && <Badge variant="secondary">passive</Badge>}
    </EntityRow>
  )
}

export function BFDTab({
  profiles, setProfiles, onDirty,
}: BFDTabShape) {
  const [openIdx, setOpenIdx] = useState<number | null>(null)
  const [formDraft, setFormDraft] = useState<BFDProfile | null>(null)

  const update = (next: BFDProfile[]) => { setProfiles(next); onDirty() }

  const handleAddNew = () => { setFormDraft(emptyBFDProfile()); setOpenIdx(-1) }
  const handleCommitNew = () => {
    if (!formDraft) return
    update([...profiles, formDraft])
    setOpenIdx(null); setFormDraft(null)
  }
  const handleCloseSheet = () => { setOpenIdx(null); setFormDraft(null) }
  const remove = (idx: number) => {
    update(profiles.filter((_, i) => i !== idx))
    if (openIdx === idx) handleCloseSheet()
  }

  const isAdding = openIdx === -1
  const openProfile = openIdx !== null && openIdx >= 0 ? profiles[openIdx] : null
  const sheetTitle = isAdding ? 'Add BFD Profile' : (openProfile?.name || 'BFD Profile')

  return (
    <div className="space-y-4">

      {profiles.length === 0 ? (
        <EmptyState
          title="No BFD Profiles"
          action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5"><Plus className="h-4 w-4" />Add Profile…</Button>}
        />
      ) : (
        <PreferencesGroup
          title="Profiles"
          header={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5"><Plus className="h-4 w-4" />Add Profile…</Button>}
        >
          {profiles.map((p, idx) => (
            <BFDProfileRow
              key={idx}
              profile={p}
              onEdit={() => setOpenIdx(idx)}
              onDelete={() => remove(idx)}
            />
          ))}
        </PreferencesGroup>
      )}

      <Sheet open={openIdx !== null} onClose={handleCloseSheet} title={sheetTitle}>
        {openIdx !== null && (isAdding ? formDraft : openProfile) && (
          <BFDProfileForm
            profile={isAdding ? formDraft! : openProfile!}
            onChange={isAdding ? setFormDraft : (u) => update(profiles.map((p, i) => i === openIdx ? u : p))}
            onAdd={isAdding ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
          />
        )}
      </Sheet>
    </div>
  )
}

