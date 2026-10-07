import { useState, useCallback, useRef } from 'react'
import { api } from '@/lib/client'
import { toast } from 'sonner'
import { useDataRefresh } from './dataVersion'

export function usePageSave(section: string) {
  const [isDirty, setIsDirty] = useState(false)
  const [saving, setSaving] = useState(false)

  const editRevision = useRef(0)
  const savingRef = useRef(false)
  const markDirty = useCallback(() => {
    editRevision.current += 1
    setIsDirty(true)
  }, [])
  const reset = useCallback(() => {
    editRevision.current += 1
    setIsDirty(false)
  }, [])
  useDataRefresh(reset)

  const save = useCallback(async (data: unknown) => {
    if (savingRef.current) return false
    savingRef.current = true
    const savedRevision = editRevision.current
    setSaving(true)
    try {
      await api.apiConfigSectionPut({ section, body: data as object })
      if (editRevision.current === savedRevision) setIsDirty(false)
      return true
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save')
      return false
    } finally {
      savingRef.current = false
      setSaving(false)
    }
  }, [section])

  return { isDirty, markDirty, save, saving, reset }
}
