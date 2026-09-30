import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Button, Input, Badge } from 'cheval-ui'
import { Plus, Users, KeyRound, ChevronRight } from 'lucide-react'
import { PageHeader, PreferencesColumns, EmptyState } from 'cheval-ui'
import { Pagination, usePagination } from 'cheval-ui'
import { ReloadButton } from 'cheval-ui'
import { Spinner } from 'cheval-ui'
import { defaultUser, isAdmin, type UsersMap } from '@/components/users/UserForm'

function UserRow({ username, user }: { username: string; user: UsersMap[string] }) {
  const keyCount = (user.ssh_keys ?? []).length
  return (
    <Link to={`/users/${encodeURIComponent(username)}`} className="flex items-center gap-3 px-4 py-3 hover:bg-accent/50 transition-colors">
      <span className="w-28 shrink-0 truncate font-mono text-sm font-semibold">{username}</span>
      <div className="flex min-w-0 flex-1 flex-wrap gap-1">
        {isAdmin(user) && <Badge variant="default" className="text-xs">admin</Badge>}
        {user.system && <Badge variant="neutral" className="text-xs">system</Badge>}
        {!user.password_hash && username !== 'routier' && <Badge variant="outline" className="text-xs text-muted-foreground">no password</Badge>}
        {keyCount > 0 && <Badge variant="outline" className="gap-1 text-xs"><KeyRound className="h-3 w-3" />{keyCount}</Badge>}
      </div>
      <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
    </Link>
  )
}

export default function ConfigUsers() {
  const navigate = useNavigate()
  const { data, isLoading, reload } = useFetch<UsersMap>(() => api.apiConfigSectionGet({ section: 'users' }) as Promise<UsersMap>)
  const [newName, setNewName] = useState('')
  const [adding, setAdding] = useState(false)

  const current: UsersMap = data ?? {}
  const names = Object.keys(current)
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(names, 15)

  const add = async () => {
    const name = newName.trim()
    if (!name) { toast.error('Please enter a username'); return }
    if (current[name]) { toast.error(`User "${name}" already exists`); return }
    setAdding(true)
    try {
      await api.apiConfigSectionPut({ section: 'users', body: { ...current, [name]: defaultUser() } })
      navigate(`/users/${encodeURIComponent(name)}`)
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to add user')
      setAdding(false)
    }
  }

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      <PageHeader title="Users" description="System accounts. Admins (wheel) can sign in to this UI and run doas." action={
        <ReloadButton onClick={reload} />
      } />

      <div className="flex max-w-md items-center gap-2">
        <Input value={newName} onChange={(e) => setNewName(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter') add() }} placeholder="new username" className="font-mono" />
        <Button onClick={add} disabled={adding || !newName.trim()} className="shrink-0 gap-1.5"><Plus className="h-4 w-4" />Add user</Button>
      </div>

      {names.length === 0 ? (
        <div className="mx-auto w-full max-w-2xl">
          <EmptyState icon={<Users />} title="No users" message="Add a user account to manage its shell, groups, SSH keys and password." />
        </div>
      ) : (
        <PreferencesColumns title="Users">
          {pageItems.map((username) => (
            <UserRow key={username} username={username} user={current[username]} />
          ))}
        </PreferencesColumns>
      )}

      <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="users" />
    </div>
  )
}
