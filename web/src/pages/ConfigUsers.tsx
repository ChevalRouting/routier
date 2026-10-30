import { defaultUser, isAdmin, type UsersMap } from '@/components/users/UserForm'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Badge, Button, EmptyState, EntryRow, PageHeader, Pagination, ReloadButton, Spinner, usePagination, PreferencesGroup } from 'cheval-ui'
import { ChevronRight, KeyRound, Users } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'

type UserRowShape = { username: string; user: UsersMap[string] }

type ConfigUsersShape = { embedded?: boolean }

function UserRow({ username, user }: UserRowShape) {
  const keyCount = (user.ssh_keys ?? []).length
  return (
    <Link to={`/system/users/${encodeURIComponent(username)}`} className="flex items-center gap-3 px-4 py-3 hover:bg-accent/50 transition-colors">
      <span className="w-28 shrink-0 truncate font-mono text-sm font-semibold">{username}</span>
      <div className="flex min-w-0 flex-1 flex-wrap gap-1">
        {isAdmin(user) && <Badge variant="default">admin</Badge>}
        {user.system && <Badge variant="neutral">system</Badge>}
        {!user.password_hash && username !== 'routier' && <Badge variant="outline" className="text-muted-foreground">no password</Badge>}
        {keyCount > 0 && <Badge variant="outline" className="gap-1"><KeyRound className="h-3 w-3" />{keyCount}</Badge>}
      </div>
      <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
    </Link>
  )
}

export default function ConfigUsers({ embedded = false }: ConfigUsersShape = {}) {
  const navigate = useNavigate()
  const { data, isLoading, reload } = useFetch<UsersMap>(() => api.apiConfigSectionGet({ section: 'users' }) as Promise<UsersMap>)
  const [newName, setNewName] = useState('')
  const [adding, setAdding] = useState(false)

  const current: UsersMap = data ?? {}
  const names = Object.keys(current)
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(names, 15)

  const add = async () => {
    const name = newName.trim()
    if (adding) return
    if (!name) { toast.error('Please enter a username'); return }
    if (current[name]) { toast.error(`User "${name}" already exists`); return }
    setAdding(true)
    try {
      await api.apiConfigSectionPut({ section: 'users', body: { ...current, [name]: defaultUser() } })
      navigate(`/system/users/${encodeURIComponent(name)}`)
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to add user')
      setAdding(false)
    }
  }

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      {embedded ? <div className="flex items-center justify-between gap-3"><span /><ReloadButton onClick={() => reload()} /></div> : <PageHeader title="Users" action={
        <ReloadButton onClick={reload} />
      } />}

      <PreferencesGroup title="Add User" className="max-w-3xl">
        <EntryRow
          title="Username"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter' && !adding) add() }}
          placeholder="username"
          autoComplete="off"
          disabled={adding}
          suffix={<Button onClick={add} disabled={adding || !newName.trim()}>{adding ? 'Adding…' : 'Add User'}</Button>}
        />
      </PreferencesGroup>

      {names.length === 0 ? (
        <div className="mx-auto w-full max-w-2xl">
          <EmptyState icon={<Users />} title="No Users" />
        </div>
      ) : (
        <PreferencesGroup title="Users">
          {pageItems.map((username) => (
            <UserRow key={username} username={username} user={current[username]} />
          ))}
        </PreferencesGroup>
      )}

      <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="users" />
    </div>
  )
}
