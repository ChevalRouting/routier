import type { TypesFriendInfo as FriendInfo, TypesFriendStatus as FriendStatus } from '@/api'
import { FriendDetailView } from '@/components/ha/FriendDetailView'
import { AddFriendSheet, FriendCard } from '@/components/ha/FriendsPanel'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Button, EmptyState, PageHeader, Sheet, Spinner } from 'cheval-ui'
import { Handshake, Plus, RefreshCw, Send } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

export function FriendsPanel() {
  const { data: friends, isLoading, reload } = useFetch<FriendInfo[]>(() => api.apiFriendsGet())
  const { data: statuses, reload: reloadStatus } = useFetch<FriendStatus[]>(() => api.apiFriendsStatusGet())
  const [adding, setAdding] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [selected, setSelected] = useState<string | null>(null)
  const [lastReload, setLastReload] = useState(() => Date.now())

  useEffect(() => {
    const tick = () => { reloadStatus(); setLastReload(Date.now()) }
    const id = setInterval(tick, 10000)
    return () => clearInterval(id)
  }, [reloadStatus])

  const statusByName = new Map((statuses ?? []).map((s) => [s.name, s]))
  const reloadAll = () => { reload(); reloadStatus(); setLastReload(Date.now()) }

  const syncAll = async () => {
    setSyncing(true)
    try {
      const res = await api.apiFriendsSyncPost()
      const failed = res.filter((r) => r.error)
      if (failed.length) failed.forEach((r) => toast.error(`${r.name}: ${r.error}`))
      else toast.success('HA cluster synced')
    } catch (err) {
      toast.error((err as Error).message)
    } finally { setSyncing(false); reloadAll() }
  }

  if (selected) {
    return <FriendDetailView name={selected} onBack={() => { setSelected(null); reloadAll() }} />
  }

  if (isLoading) return <Spinner />

  const list = friends ?? []
  const hasHA = list.some((f) => f.ha)

  return (
    <div className="space-y-6">
      <PageHeader
        title="Friends"
        description="Mutually-authenticated peer routers for liveness, HA and WireGuard"
        action={
          <div className="flex items-center gap-2">
            <span className="hidden text-xs text-muted-foreground sm:inline">Updated {new Date(lastReload).toLocaleTimeString()}</span>
            <Button variant="outline" size="sm" className="gap-1.5" onClick={reloadAll}>
              <RefreshCw className="h-3.5 w-3.5" />Refresh
            </Button>
            {hasHA && (
              <Button variant="outline" size="sm" className="gap-1.5" onClick={syncAll} disabled={syncing}>
                <Send className="h-3.5 w-3.5" />{syncing ? 'Syncing…' : 'Sync HA'}
              </Button>
            )}
            {list.length > 0 && (
              <Button size="sm" className="gap-1.5" onClick={() => setAdding(true)}>
                <Plus className="h-3.5 w-3.5" />Add friend
              </Button>
            )}
          </div>
        }
      />

      {list.length === 0 ? (
        <EmptyState className="max-w-2xl mx-auto" icon={<Handshake />} title="No friends"
          message="Add a peer router to monitor it, form an HA cluster, or derive WireGuard tunnels."
          action={<Button size="sm" className="gap-1.5" onClick={() => setAdding(true)}><Plus className="h-3.5 w-3.5" />Add first friend</Button>} />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {list.map((f) => (
            <FriendCard key={f.name} info={f} status={statusByName.get(f.name)} onOpen={() => setSelected(f.name)} onChanged={reloadAll} />
          ))}
        </div>
      )}

      {adding && (
        <Sheet open onClose={() => setAdding(false)} title="Add friend">
          <AddFriendSheet onClose={() => setAdding(false)} onAdded={reloadAll} />
        </Sheet>
      )}
    </div>
  )
}
