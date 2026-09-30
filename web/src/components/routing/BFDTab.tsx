import React, { useState } from 'react'
import { Label } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { Sheet } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { PreferencesGroup, PreferencesColumns, EntryRow, SwitchRow } from 'cheval-ui'
import { BFDProfile } from './types'

export function emptyBFDProfile(): BFDProfile {
  return { name: '' }
}

export function BFDProfileForm({
  profile, onChange, onAdd, onDone,
}: {
  profile: BFDProfile
  onChange: (v: BFDProfile) => void
  onAdd?: () => void
  onDone: () => void
}) {
  const set = (patch: Partial<BFDProfile>) => onChange({ ...profile, ...patch })

  const numField = (v: number | undefined) => v ?? ''
  const onNum = (key: keyof BFDProfile) => (e: React.ChangeEvent<HTMLInputElement>) =>
    set({ [key]: e.target.value ? Number(e.target.value) : undefined })

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        <EntryRow title="Profile name" autoFocus value={profile.name} onChange={(e) => set({ name: e.target.value })} placeholder="fast" className="font-mono" />
        <EntryRow title="Detect multiplier" value={numField(profile.detect_multiplier)} onChange={onNum('detect_multiplier')} placeholder="3" className="font-mono" />
        <EntryRow title="Minimum TTL" value={numField(profile.minimum_ttl)} onChange={onNum('minimum_ttl')} placeholder="254" className="font-mono" />
        <EntryRow title="Receive interval (ms)" value={numField(profile.receive_interval)} onChange={onNum('receive_interval')} placeholder="300" className="font-mono" />
        <EntryRow title="Transmit interval (ms)" value={numField(profile.transmit_interval)} onChange={onNum('transmit_interval')} placeholder="300" className="font-mono" />
        <SwitchRow title="Echo mode" checked={!!profile.echo_mode} onCheckedChange={(v) => set({ echo_mode: v })} />
        <SwitchRow title="Passive mode" checked={!!profile.passive_mode} onCheckedChange={(v) => set({ passive_mode: v })} />
      </PreferencesGroup>

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add Profile</Button>}
      </div>
    </div>
  )
}

export function BFDProfileRow({
  profile, onEdit, onDelete,
}: {
  profile: BFDProfile
  onEdit: () => void
  onDelete: () => void
}) {
  const timers: string[] = []
  if (profile.detect_multiplier) timers.push(`×${profile.detect_multiplier}`)
  if (profile.receive_interval) timers.push(`rx ${profile.receive_interval}ms`)
  if (profile.transmit_interval) timers.push(`tx ${profile.transmit_interval}ms`)
  return (
    <div onClick={onEdit} className="flex items-center gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors">
      <span className="font-mono font-semibold text-sm w-40 shrink-0 truncate">
        {profile.name || <span className="text-muted-foreground italic font-normal">unnamed</span>}
      </span>
      <div className="flex flex-wrap items-center gap-1.5 flex-1 min-w-0">
        {timers.length > 0
          ? timers.map((t) => <Badge key={t} variant="outline" className="text-xs font-mono">{t}</Badge>)
          : <span className="text-xs text-muted-foreground italic">default timers</span>}
        {profile.echo_mode && <Badge variant="secondary" className="text-xs">echo</Badge>}
        {profile.passive_mode && <Badge variant="secondary" className="text-xs">passive</Badge>}
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={(e) => { e.stopPropagation(); onDelete() }}
        className="h-7 w-7 p-0 hover:text-destructive shrink-0"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

export function BFDTab({
  profiles, setProfiles, onDirty,
}: {
  profiles: BFDProfile[]
  setProfiles: (v: BFDProfile[]) => void
  onDirty: () => void
}) {
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
      <p className="text-sm text-muted-foreground">
        BFD (Bidirectional Forwarding Detection) profiles define failure-detection timers. Reference a profile
        from a BGP neighbor to enable sub-second failover. Profiles are shared across the main table and all VRFs.
      </p>

      <div className="flex items-center justify-between">
        <Label className="text-sm font-semibold">Profiles</Label>
        <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
          <Plus className="h-3.5 w-3.5" />Add Profile
        </Button>
      </div>

      {profiles.length === 0 ? (
        <EmptyState
          title="No BFD profiles"
          message="Profiles define failure-detection timers referenced by BGP and OSPF neighbors."
          action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add profile</Button>}
        />
      ) : (
        <PreferencesColumns>
          {profiles.map((p, idx) => (
            <BFDProfileRow
              key={idx}
              profile={p}
              onEdit={() => setOpenIdx(idx)}
              onDelete={() => remove(idx)}
            />
          ))}
        </PreferencesColumns>
      )}

      <Sheet open={openIdx !== null} onClose={handleCloseSheet} title={sheetTitle} className="max-w-xl">
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

