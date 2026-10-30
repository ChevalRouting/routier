import { useState } from 'react'
import { api } from './client'
import { useDataRefresh } from './dataVersion'
import { useFetch } from './useFetch'
import { usePageSave } from './usePageSave'
import type { SwitchDefinition, SwitchDefinitions } from './switches'

export function useSwitchDefinitions() {
  const staged = useFetch(() => api.apiConfigSectionGet({ section: 'switches' }) as Promise<SwitchDefinitions | null>)
  const [draft, setDraft] = useState<SwitchDefinitions | null>(null)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('switches')
  useDataRefresh(() => setDraft(null))

  const definitions: SwitchDefinitions = draft ?? staged.latest ?? {}

  const setDefinition = (name: string, definition: SwitchDefinition | null) => {
    const next = { ...definitions }
    if (definition) next[name] = definition
    else delete next[name]
    setDraft(next)
    markDirty()
  }

  const removeDefinition = async (name: string) => {
    const next = { ...definitions }
    delete next[name]
    const saved = await save(next)
    if (saved) setDraft(null)
    return saved
  }

  const saveDraft = async () => {
    await save(definitions)
    setDraft(null)
  }

  const discard = () => {
    setDraft(null)
    reset()
    void staged.reload(true)
  }

  return { definitions, setDefinition, removeDefinition, isDirty, saving, saveDraft, discard, isLoading: !staged.hasData && staged.isLoading, error: staged.error }
}
