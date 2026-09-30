import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { PageHeader, SaveButton, Button, Spinner } from 'cheval-ui'
import { ArrowLeft, Trash2 } from 'lucide-react'
import { UserFields, type UserConfig, type UsersMap } from '@/components/users/UserForm'

function BackLink() {
  return (
    <Link to="/users" className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
      <ArrowLeft className="h-4 w-4" />Users
    </Link>
  )
}

export default function UserDetail() {
  const { name = '' } = useParams()
  const navigate = useNavigate()
  const { data, isLoading } = useFetch<UsersMap>(() => api.apiConfigSectionGet({ section: 'users' }) as Promise<UsersMap>)
  const [users, setUsers] = useState<UsersMap | null>(null)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('users')

  useEffect(() => { if (data && users === null) setUsers(data) }, [data, users])

  if (isLoading || users === null) return <Spinner />

  const user = users[name]
  if (!user) {
    return (
      <div className="space-y-4">
        <BackLink />
        <p className="text-sm text-muted-foreground">User &quot;{name}&quot; not found.</p>
      </div>
    )
  }

  const update = (u: UserConfig) => { setUsers({ ...users, [name]: u }); markDirty() }

  const remove = async () => {
    const next = { ...users }
    delete next[name]
    try {
      await api.apiConfigSectionPut({ section: 'users', body: next })
      toast.success(`User "${name}" removed, apply to activate`)
      navigate('/users')
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to remove user')
    }
  }

  return (
    <div className="space-y-6">
      <BackLink />
      <PageHeader title={name} description="System user account" action={
        <div className="flex items-center gap-2">
          <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(users)} onCancel={() => { setUsers(data ?? {}); reset() }} />
          <Button variant="outline" onClick={remove} className="gap-1.5 text-destructive hover:text-destructive">
            <Trash2 className="h-4 w-4" />Delete
          </Button>
        </div>
      } />
      <div className="max-w-2xl">
        <UserFields user={user} onChange={update} />
      </div>
    </div>
  )
}
