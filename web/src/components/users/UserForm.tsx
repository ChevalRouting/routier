import { activeBaseUrl } from '@/lib/instance'
import { getToken } from '@/lib/utils'
import { Button, ComboRow, EntryRow, PreferencesGroup, SwitchRow, TagInput } from 'cheval-ui'
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
    if (!pw || hashing) return
    setHashing(true)
    try {
      set('password_hash', await hashPassword(pw))
      setPw('')
      toast.success('Password updated, save the draft and apply to activate')
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to set password')
    } finally {
      setHashing(false)
    }
  }

  return (
    <div className="space-y-5">
      <PreferencesGroup title="Account">
        <EntryRow title="UID" type="number" min={0} step={1} help="Use 0 to assign a UID automatically" value={user.uid || ''} onChange={(e) => set('uid', Number(e.target.value) || 0)} placeholder="0" />
        <EntryRow title="Shell" value={user.shell} onChange={(e) => set('shell', e.target.value)} placeholder="/bin/ash" />
        <EntryRow title="Home Directory" value={user.home} onChange={(e) => set('home', e.target.value)} placeholder="/home/username" />
        <SwitchRow title="System Account" subtitle="Create as a system user" checked={user.system} onCheckedChange={(v) => set('system', v)} />
        <SwitchRow
          title="Routier Administrator"
          subtitle="Adds the wheel group: UI sign-in and doas"
          checked={isAdmin(user)}
          onCheckedChange={(v) => set('groups', v
            ? [...new Set([...(user.groups ?? []), 'wheel'])]
            : (user.groups ?? []).filter((g) => g !== 'wheel'))}
        />
      </PreferencesGroup>

      <PreferencesGroup title="Password" description="Save the draft and apply to activate a password change">
        <EntryRow
          title="New Password"
          type="password"
          value={pw}
          onChange={(e) => setPw(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') setPassword() }}
          placeholder="Enter a new password"
          autoComplete="new-password"
          disabled={hashing}
          help={user.password_hash ? 'A password is set; leave empty to keep it' : 'No password is set'}
          suffix={<Button variant="outline" disabled={!pw || hashing} onClick={setPassword}>{hashing ? 'Setting…' : 'Set Password'}</Button>}
        />
      </PreferencesGroup>

      <PreferencesGroup title="Access">
        <ComboRow title="Groups" subtitle="Additional groups for this account">
          <TagInput values={user.groups ?? []} onChange={(v) => set('groups', v)} placeholder="Add group…" />
        </ComboRow>
        <ComboRow title="SSH Keys" subtitle="Public keys allowed to sign in">
          <TagInput values={user.ssh_keys ?? []} onChange={(v) => set('ssh_keys', v)} placeholder="ssh-ed25519 AAAA…" mono />
        </ComboRow>
      </PreferencesGroup>
    </div>
  )
}
