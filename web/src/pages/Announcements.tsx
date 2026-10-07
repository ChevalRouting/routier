import type { TypesAnnouncement as Announcement, TypesAnnouncementLevelEnum as AnnouncementLevel } from '@/api'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Textarea, Badge, Button, EmptyState, Label, PageHeader, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Spinner, Switch } from 'cheval-ui'
import { Megaphone, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

type AnnouncementFormShape = {
  initial?: Announcement
  submitLabel: string
  onSubmit: (input: AnnouncementInput) => Promise<void>
  onCancel?: () => void
}

const LEVELS: AnnouncementLevel[] = ['info', 'warning', 'danger']

type AnnouncementInput = Pick<Announcement, 'message' | 'level' | 'enabled' | 'dismissible'>

function AnnouncementForm({ initial, submitLabel, onSubmit, onCancel }: AnnouncementFormShape) {
  const [message, setMessage] = useState(initial?.message ?? '')
  const [level, setLevel] = useState<AnnouncementLevel>(initial?.level ?? 'info')
  const [enabled, setEnabled] = useState(initial?.enabled ?? true)
  const [dismissible, setDismissible] = useState(initial?.dismissible ?? true)
  const [busy, setBusy] = useState(false)

  const submit = async () => {
    if (!message.trim()) { toast.error('Message is required'); return }
    setBusy(true)
    try {
      await onSubmit({ message: message.trim(), level, enabled, dismissible })
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="space-y-3 rounded-md bg-card p-3 shadow-[var(--card-shadow)]">
      <Textarea
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        placeholder="Maintenance window tonight 22:00-23:00 UTC"
        className="w-full min-h-16 rounded-md border border-border bg-transparent p-2 text-sm outline-none focus-visible:ring-1 focus-visible:ring-ring"
      />
      <div className="flex flex-wrap items-center gap-4">
        <div className="flex items-center gap-2">
          <Label className="text-xs">Level</Label>
          <Select value={level} onValueChange={(v) => setLevel(v as AnnouncementLevel)}>
            <SelectTrigger className="h-8 w-28 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent>
              {LEVELS.map((l) => <SelectItem key={l} value={l}>{l}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
        <div className="flex items-center gap-2">
          <Switch checked={enabled} onCheckedChange={setEnabled} />
          <span className="text-sm">Enabled</span>
        </div>
        <div className="flex items-center gap-2">
          <Switch checked={dismissible} onCheckedChange={setDismissible} />
          <span className="text-sm">Dismissible</span>
        </div>
        <div className="ml-auto flex gap-2">
          {onCancel && <Button variant="ghost" size="sm" onClick={onCancel}>Cancel</Button>}
          <Button size="sm" onClick={submit} disabled={busy}>{submitLabel}</Button>
        </div>
      </div>
    </div>
  )
}

export default function Announcements() {
  const { data, isLoading, reload } = useFetch<Announcement[]>(() => api.apiAnnouncementsGet())
  const [adding, setAdding] = useState(false)

  const create = async (input: AnnouncementInput) => {
    try {
      await api.apiAnnouncementsPost({ TypesAnnouncement: input as Announcement })
      toast.success('Announcement created')
      setAdding(false)
      reload()
    } catch (e) {
      toast.error((e as Error).message)
    }
  }

  const save = async (id: number, input: AnnouncementInput) => {
    try {
      await api.apiAnnouncementsIdPut({ id, TypesAnnouncement: input as Announcement })
      toast.success('Saved')
      reload()
    } catch (e) {
      toast.error((e as Error).message)
    }
  }

  const remove = async (id: number) => {
    try {
      await api.apiAnnouncementsIdDelete({ id })
      toast.success('Deleted')
      reload()
    } catch (e) {
      toast.error((e as Error).message)
    }
  }

  if (isLoading) return <Spinner />

  const items = data ?? []

  return (
    <div className="space-y-6">
      <PageHeader
        title="Announcements"
        description="Banner messages shown at the top of every page"
        action={!adding && (
          <Button variant="outline" size="sm" onClick={() => setAdding(true)} className="gap-1.5">
            <Plus className="h-4 w-4" />New
          </Button>
        )}
      />

      {adding && <AnnouncementForm submitLabel="Create" onSubmit={create} onCancel={() => setAdding(false)} />}

      {items.length === 0 && !adding ? (
        <EmptyState
          icon={<Megaphone />}
          title="No announcements"
          message="Create a banner to notify users (maintenance windows, notices, hostname, etc.)."
          action={<Button variant="outline" size="sm" onClick={() => setAdding(true)} className="gap-2"><Plus className="h-4 w-4" />New announcement</Button>}
        />
      ) : (
        <div className="space-y-4">
          {items.map((a) => (
            <div key={a.id} className="space-y-2">
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-[10px]">{a.level}</Badge>
                {!a.enabled && <Badge variant="secondary" className="text-[10px]">disabled</Badge>}
                <Button variant="ghost" size="icon" className="ml-auto h-7 w-7 hover:text-destructive" onClick={() => remove(a.id)}>
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
              <AnnouncementForm initial={a} submitLabel="Save" onSubmit={(input) => save(a.id, input)} />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
