import { createContext, useContext } from 'react'

export interface DraftAction {
  isDirty: boolean
  saving: boolean
  disabled?: boolean
  save: () => unknown
  discard?: () => unknown
}

export type RegisterDraftAction = (id: string, action: DraftAction | null) => void

export const DraftActionsContext = createContext<RegisterDraftAction>(() => {})
export const useDraftActions = () => useContext(DraftActionsContext)
