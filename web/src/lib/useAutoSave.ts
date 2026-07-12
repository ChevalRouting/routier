import { useCallback, useRef } from 'react'
import { api } from '@/lib/client'
import { toast } from 'sonner'

export function useAutoSave(section: string, debounceMs = 700) {
  const timerRef = useRef<ReturnType<typeof setTimeout>>()

  const saveNow = useCallback(async (data: unknown) => {
    clearTimeout(timerRef.current)
    try {
      await api.apiConfigSectionPut({ section, body: data as object })
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save')
    }
  }, [section])

  const save = useCallback((data: unknown) => {
    clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => saveNow(data), debounceMs)
  }, [saveNow, debounceMs])

  return { save, saveNow }
}
