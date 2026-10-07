import type { TypesFriendInfo as FriendInfo, TypesFriendStatus as FriendStatus } from '@/api'
import { InstanceRow, Tone, dotColor, useActiveInstance } from '@/components/InstanceSwitcher'
import { selfApi } from '@/lib/client'
import { useDataVersion } from '@/lib/dataVersion'
import {
  SELF,
  friendBaseUrl,
  switchInstance,
  type Instance
} from '@/lib/instance'
import { useFetch } from '@/lib/useFetch'
import { cn } from 'cheval-ui'
import { Server } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

type InstanceMenuShape = { tone?: Tone; onDone?: () => void }

export function InstanceMenu({ tone = 'sidebar', onDone }: InstanceMenuShape) {
  const navigate = useNavigate()
  const { bump } = useDataVersion()
  const active = useActiveInstance()

  const { data: friends } = useFetch<FriendInfo[]>(() => selfApi.apiFriendsGet())
  const { data: statuses } = useFetch<FriendStatus[]>(() => selfApi.apiFriendsStatusGet())

  const statusByName = new Map((statuses ?? []).map((s) => [s.name, s]))

  const select = (inst: Instance) => {
    onDone?.()
    if (inst.id === active.id) return
    switchInstance(inst)
    bump()
    navigate('/')
  }

  const selectFriend = (f: FriendInfo) => {
    select({ id: f.name, label: f.hostname || f.name, baseUrl: friendBaseUrl(f.name) })
  }

  return (
    <div className="py-1">
      <InstanceRow
        tone={tone}
        icon={<Server className="h-3.5 w-3.5 shrink-0 opacity-70" />}
        label="This Instance"
        active={active.id === 'self'}
        onClick={() => select(SELF)}
      />
      {(friends ?? []).length > 0 && (
        <div className={cn('my-1 border-t', tone === 'sheet' ? 'border-border/50' : 'border-sidebar-border/40')} />
      )}
      {(friends ?? []).map((f) => (
        <InstanceRow
          key={f.name}
          tone={tone}
          icon={<span className={cn('h-2 w-2 rounded-full shrink-0', dotColor(statusByName.get(f.name)?.reachable))} />}
          label={f.hostname || f.name}
          sub={f.hostname && f.hostname !== f.name ? f.name : undefined}
          active={active.id === f.name}
          onClick={() => selectFriend(f)}
        />
      ))}
    </div>
  )
}
