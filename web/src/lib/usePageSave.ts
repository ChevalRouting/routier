import { useState, useCallback } from 'react'
import { api } from '@/lib/client'
import { toast } from 'sonner'

export function usePageSave(section: string) {
  const [isDirty, setIsDirty] = useState(false)
  const [saving, setSaving] = useState(false)

  const markDirty = useCallback(() => setIsDirty(true), [])

  const save = useCallback(async (data: unknown) => {
    setSaving(true)
    try {
      await api.apiConfigSectionPut({ section, body: data as object })
      setIsDirty(false)
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save')
    } finally {
      setSaving(false)
    }
  }, [section])

  const reset = useCallback(() => setIsDirty(false), [])

  return { isDirty, markDirty, save, saving, reset }
}
