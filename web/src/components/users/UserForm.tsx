import { activeBaseUrl } from '@/lib/instance'
import { getToken } from '@/lib/utils'
import { Button, EntryRow, Input, PreferencesGroup, SwitchRow, TagInput } from 'cheval-ui'
import { useState } from 'react'
import { toast } from 'sonner'

type UserFieldsShape = { user: UserConfig; onChange: (u: UserConfig) => void }

export interface UserConfig {
  uid: number
  shell: string
  home: string
  groups: string[]
  ssh_keys: string[]
  system: boolean
  password_hash?: string
}

export type UsersMap = Record<string, UserConfig>

export function defaultUser(): UserConfig {
  return { uid: 0, shell: '/bin/ash', home: '', groups: ['wheel'], ssh_keys: [], system: false }
}

export function isAdmin(user: UserConfig): boolean {
  return (user.groups ?? []).includes('wheel')
}

async function hashPassword(password: string): Promise<string> {
  const res = await fetch(activeBaseUrl() + '/api/auth/password-hash', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${getToken() ?? ''}` },
    body: JSON.stringify({ password }),
  })
  if (!res.ok) throw new Error('failed to hash password')
  const data = await res.json()
  return data.result?.hash ?? data.hash
}

export function UserFields({ user, onChange }: UserFieldsShape) {
  const [pw, setPw] = useState('')
  const [hashing, setHashing] = useState(false)

  const set = <K extends keyof UserConfig>(key: K, val: UserConfig[K]) => onChange({ ...user, [key]: val })

  const setPassword = async () => {
    if (!pw) return
    setHashing(true)
    try {
      set('password_hash', await hashPassword(pw))
      setPw('')
      toast.success('Password set, apply to activate')
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to set password')
    } finally {
      setHashing(false)
    }
  }

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        <EntryRow title="UID (0 = auto)" value={user.uid || ''} onChange={(e) => set('uid', Number(e.target.value) || 0)} placeholder="0" className="font-mono" />
        <EntryRow title="Shell" value={user.shell} onChange={(e) => set('shell', e.target.value)} placeholder="/bin/ash" className="font-mono" />
        <EntryRow title="Home directory" value={user.home} onChange={(e) => set('home', e.target.value)} placeholder="/home/username" className="font-mono" />
        <SwitchRow title="System account" subtitle="Create as a system user" checked={user.system} onCheckedChange={(v) => set('system', v)} />
        <SwitchRow
          title="Routier administrator"
          subtitle="Adds the wheel group: can sign in to this UI (verified against the system password) and run doas."
          checked={isAdmin(user)}
          onCheckedChange={(v) => set('groups', v
            ? [...new Set([...(user.groups ?? []), 'wheel'])]
            : (user.groups ?? []).filter((g) => g !== 'wheel'))}
        />
      </PreferencesGroup>

      <PreferencesGroup title="Password" description="Stored as a hash and applied to the system on save. A user with no password cannot sign in.">
        <div className="space-y-2 px-4 py-3">
          <p className="text-xs text-muted-foreground">
            {user.password_hash ? 'A password is set. Enter a new one to change it.' : 'No password set.'}
          </p>
          <div className="flex gap-2">
            <Input type="password" value={pw} onChange={(e) => setPw(e.target.value)} placeholder="New password" autoComplete="new-password" />
            <Button variant="outline" disabled={!pw || hashing} onClick={setPassword} className="shrink-0">Set</Button>
          </div>
        </div>
      </PreferencesGroup>

      <PreferencesGroup title="Groups">
        <div className="px-4 py-3">
          <TagInput values={user.groups ?? []} onChange={(v) => set('groups', v)} placeholder="Add group…" />
        </div>
      </PreferencesGroup>

      <PreferencesGroup title="SSH keys" description="Authorized keys for SSH access.">
        <div className="px-4 py-3">
          <TagInput values={user.ssh_keys ?? []} onChange={(v) => set('ssh_keys', v)} placeholder="ssh-ed25519 AAAA…" mono />
        </div>
      </PreferencesGroup>
    </div>
  )
}
