import { useState, useEffect } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import SaveButton from '@/components/SaveButton'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Sheet } from '@/components/ui/sheet'
import TagInput from '@/components/TagInput'
import { Plus, Trash2, Users } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { PreferencesGroup, PreferencesColumns, EntryRow, SwitchRow } from '@/components/Preferences'
import { EmptyState } from '@/components/EmptyState'
import { Pagination, usePagination } from '@/components/Pagination'
import { ReloadButton } from '@/components/ReloadButton'
import { Spinner } from '@/components/Spinner'

interface UserConfig {
  uid: number
  shell: string
  home: string
  groups: string[]
  ssh_keys: string[]
  system: boolean
}

type UsersMap = Record<string, UserConfig>

function defaultUser(): UserConfig {
  return { uid: 0, shell: '/bin/sh', home: '', groups: [], ssh_keys: [], system: false }
}

interface UserFormProps {
  user: UserConfig
  onChange: (u: UserConfig) => void
  name?: string
  onNameChange?: (n: string) => void
  onAdd?: () => void
  onDone: () => void
}

function UserForm({ user, onChange, name, onNameChange, onAdd, onDone }: UserFormProps) {
  const set = <K extends keyof UserConfig>(key: K, val: UserConfig[K]) =>
    onChange({ ...user, [key]: val })

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        {onNameChange !== undefined && (
          <EntryRow
            title="Username"
            autoFocus
            value={name ?? ''}
            onChange={(e) => onNameChange(e.target.value)}
            placeholder="deploy"
            className="font-mono"
          />
        )}
        <EntryRow
          title="UID (0 = auto)"
          value={user.uid || ''}
          onChange={(e) => set('uid', Number(e.target.value) || 0)}
          placeholder="0"
          className="font-mono"
        />
        <EntryRow
          title="Shell"
          value={user.shell}
          onChange={(e) => set('shell', e.target.value)}
          placeholder="/bin/sh"
          className="font-mono"
        />
        <EntryRow
          title="Home directory"
          value={user.home}
          onChange={(e) => set('home', e.target.value)}
          placeholder="/home/username"
          className="font-mono"
        />
        <SwitchRow
          title="System account"
          subtitle="Create as a system user"
          checked={user.system}
          onCheckedChange={(v) => set('system', v)}
        />
      </PreferencesGroup>

      <PreferencesGroup title="Groups">
        <div className="px-4 py-3">
          <TagInput values={user.groups ?? []} onChange={(v) => set('groups', v)} placeholder="Add group…" />
        </div>
      </PreferencesGroup>

      <PreferencesGroup title="SSH keys">
        <div className="px-4 py-3">
          <TagInput values={user.ssh_keys ?? []} onChange={(v) => set('ssh_keys', v)} placeholder="ssh-ed25519 AAAA…" mono />
        </div>
      </PreferencesGroup>

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add user</Button>}
      </div>
    </div>
  )
}

interface UserRowProps {
  username: string
  user: UserConfig
  onEdit: () => void
  onDelete: () => void
}

function UserRow({ username, user, onEdit, onDelete }: UserRowProps) {
  return (
    <div
      onClick={onEdit}
      className="flex items-start gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors"
    >
      <span className="font-mono font-semibold text-sm w-28 shrink-0 truncate pt-0.5">{username}</span>
      <div className="flex flex-wrap gap-1 flex-1 min-w-0">
        {user.system && <Badge variant="neutral" className="text-xs">system</Badge>}
        {(user.groups ?? []).map((g) => (
          <Badge key={g} variant="outline" className="text-xs font-mono">{g}</Badge>
        ))}
        {(user.ssh_keys ?? []).length > 0 && (
          <Badge variant="outline" className="text-xs">
            {(user.ssh_keys ?? []).length} key{(user.ssh_keys ?? []).length !== 1 ? 's' : ''}
          </Badge>
        )}
        {user.shell && user.shell !== '/bin/sh' && (
          <Badge variant="outline" className="text-xs font-mono">{user.shell}</Badge>
        )}
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

const NEW_KEY = '__new__'

export default function ConfigUsers() {
  const { data, isLoading, reload } = useFetch<UsersMap>(
    () => api.apiConfigSectionGet({ section: 'users' }) as Promise<UsersMap>
  )
  const [users, setUsers] = useState<UsersMap | null>(null)
  const [newName, setNewName] = useState('')
  const [openSheet, setOpenSheet] = useState<string | null>(null)
  const [formDraft, setFormDraft] = useState<UserConfig | null>(null)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('users')

  useEffect(() => {
    if (data && users === null) setUsers(data)
  }, [data, users])

  const current: UsersMap = users ?? (data as UsersMap | null) ?? {}

  const update = (username: string, u: UserConfig) => { setUsers({ ...current, [username]: u }); markDirty() }

  const deleteUser = (username: string) => {
    const next = { ...current }
    delete next[username]
    setUsers(next)
    if (openSheet === username) setOpenSheet(null)
    markDirty()
  }

  const handleAddNew = () => {
    setFormDraft(defaultUser()); setNewName(''); setOpenSheet(NEW_KEY)
  }

  const handleCommitNew = () => {
    const n = newName.trim()
    if (!n) { toast.error('Please enter a username'); return }
    if (current[n]) { toast.error(`User "${n}" already exists`); return }
    setUsers({ ...current, [n]: formDraft ?? defaultUser() }); markDirty()
    setOpenSheet(null); setFormDraft(null)
  }

  const handleCloseSheet = () => { setOpenSheet(null); setFormDraft(null) }

  const names = Object.keys(current)
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(names, 12)

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      <PageHeader title="Users" description="Manage system user accounts" action={
        <div className="flex items-center gap-2">
          <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(current)} onCancel={() => { setUsers(null); reset() }} />
          <ReloadButton onClick={() => { setUsers(null); reload() }} />
        </div>
      } />

      {names.length === 0 ? (
        <div className="w-full max-w-2xl mx-auto">
          <EmptyState
            icon={<Users />}
            title="No users"
            message="Define a user account to manage its shell, groups and SSH keys."
            action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add user</Button>}
          />
        </div>
      ) : (
        <PreferencesColumns
          title="Users"
          header={
            <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
              <Plus className="h-4 w-4" />Add
            </Button>
          }
        >
          {pageItems.map((username) => (
            <UserRow
              key={username}
              username={username}
              user={current[username]}
              onEdit={() => setOpenSheet(username)}
              onDelete={() => deleteUser(username)}
            />
          ))}
        </PreferencesColumns>
      )}

      <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="users" />

      <Sheet
        open={!!openSheet}
        onClose={handleCloseSheet}
        title={openSheet === NEW_KEY ? 'New User' : (openSheet ?? '')}
      >
        {openSheet && (openSheet === NEW_KEY ? formDraft : current[openSheet]) && (
          <UserForm
            user={openSheet === NEW_KEY ? formDraft! : current[openSheet]}
            onChange={openSheet === NEW_KEY ? setFormDraft : (u) => update(openSheet, u)}
            name={openSheet === NEW_KEY ? newName : undefined}
            onNameChange={openSheet === NEW_KEY ? setNewName : undefined}
            onAdd={openSheet === NEW_KEY ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
          />
        )}
      </Sheet>
    </div>
  )
}
