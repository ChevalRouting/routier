import type { TypesAnnouncement as Announcement, TypesAnnouncementLevelEnum as AnnouncementLevel } from '@/api'
import { api } from '@/lib/client'
import { useDataVersion } from '@/lib/dataVersion'
import { X } from 'lucide-react'
import { useEffect, useState } from 'react'

const DISMISS_KEY = 'routier:dismissed-announcements'

const LEVEL_CLASS: Record<AnnouncementLevel, string> = {
  info: 'bg-info/15 text-info border-info/30',
  warning: 'bg-warning/15 text-warning border-warning/30',
  danger: 'bg-danger/15 text-danger border-danger/30',
}

function loadDismissed(): Record<string, number> {
  try { return JSON.parse(localStorage.getItem(DISMISS_KEY) || '{}') as Record<string, number> } catch { return {} }
}

export function AnnouncementBanner() {
  const [items, setItems] = useState<Announcement[]>([])
  const [dismissed, setDismissed] = useState<Record<string, number>>(loadDismissed)
  const { version } = useDataVersion()

  useEffect(() => {
    let active = true
    const load = () => api.apiAnnouncementsActiveGet().then((a) => { if (active) setItems(a ?? []) }).catch(() => {})
    load()
    const id = setInterval(load, 60_000)
    return () => { active = false; clearInterval(id) }
  }, [version])

  const dismiss = (a: Announcement) => {
    const next = { ...dismissed, [a.id]: a.updated_at }
    setDismissed(next)
    try { localStorage.setItem(DISMISS_KEY, JSON.stringify(next)) } catch {  }
  }

  const visible = items.filter((a) => !(a.dismissible && dismissed[a.id] === a.updated_at))
  if (visible.length === 0) return null

  return (
    <div className="shrink-0">
      {visible.map((a) => (
        <div key={a.id} className={`flex items-center gap-3 border-b px-4 py-2 text-sm ${LEVEL_CLASS[a.level] ?? LEVEL_CLASS.info}`}>
          <span className="flex-1 min-w-0 break-words">{a.message}</span>
          {a.dismissible && (
            <button type="button" onClick={() => dismiss(a)} className="shrink-0 opacity-70 hover:opacity-100" aria-label="Dismiss">
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
      ))}
    </div>
  )
}
