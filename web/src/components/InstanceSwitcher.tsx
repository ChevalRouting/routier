import { useState, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Server, Handshake, Check, ChevronsUpDown } from 'lucide-react'
import { selfApi } from '@/lib/client'
import {
  SELF, switchInstance, getActiveInstance, subscribeInstance, friendBaseUrl,
  type Instance,
} from '@/lib/instance'
import { useDataVersion } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { cn } from 'cheval-ui'
import type { TypesFriendInfo as FriendInfo, TypesFriendStatus as FriendStatus } from '@/api'

type Tone = 'sidebar' | 'sheet'

export function useActiveInstance(): Instance {
  const [inst, setInst] = useState(getActiveInstance)
  useEffect(() => subscribeInstance(setInst), [])
  return inst
}

function dotColor(reachable?: boolean): string {
  if (reachable === undefined) return 'bg-muted-foreground/40'
  return reachable ? 'bg-success' : 'bg-danger'
}

export function InstanceMenu({ tone = 'sidebar', onDone }: { tone?: Tone; onDone?: () => void }) {
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

export function InstanceSwitcher({ collapsed }: { collapsed: boolean }) {
  const active = useActiveInstance()
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false) }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const isFriend = active.id !== 'self'
  const Icon = isFriend ? Handshake : Server

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        title={collapsed ? active.label : undefined}
        className={cn(
          'flex items-center rounded-md text-[13px] font-medium transition-colors w-full',
          'text-sidebar-foreground/80 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground',
          isFriend && 'bg-info/15 text-info hover:bg-info/20 hover:text-info',
          collapsed ? 'justify-center py-2 h-9' : 'gap-2 px-2.5 py-1.5',
        )}
      >
        <Icon className="h-3.5 w-3.5 shrink-0" />
        {!collapsed && (
          <>
            <span className="truncate">{active.label}</span>
            <ChevronsUpDown className="ml-auto h-3 w-3 opacity-50" />
          </>
        )}
      </button>

      {open && (
        <div
          className={cn(
            'absolute z-50 mt-1 rounded-md border border-sidebar-border bg-sidebar shadow-lg',
            collapsed ? 'left-full ml-1 top-0 w-48' : 'left-0 right-0',
          )}
        >
          <InstanceMenu onDone={() => setOpen(false)} />
        </div>
      )}
    </div>
  )
}

interface InstanceRowProps {
  tone: Tone
  icon: React.ReactNode
  label: string
  sub?: string
  active: boolean
  onClick: () => void
}

function InstanceRow({ tone, icon, label, sub, active, onClick }: InstanceRowProps) {
  const tones = tone === 'sheet'
    ? {
        active: 'text-primary',
        idle: 'text-foreground/80 hover:bg-muted/60 hover:text-foreground',
      }
    : {
        active: 'text-sidebar-accent-foreground',
        idle: 'text-sidebar-foreground/70 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground',
      }

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex items-center gap-2 px-2.5 py-2 text-[13px] w-full text-left transition-colors',
        active ? tones.active : tones.idle,
      )}
    >
      {icon}
      <span className="truncate">{label}</span>
      {sub && <span className="truncate text-[11px] text-muted-foreground font-mono">{sub}</span>}
      {active && <Check className="ml-auto h-3.5 w-3.5 shrink-0" />}
    </button>
  )
}
