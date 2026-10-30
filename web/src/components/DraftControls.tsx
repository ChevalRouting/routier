import { useEffect, useId, useRef } from 'react'
import { useDraftActions } from '@/lib/draftActions'

interface DraftControlsProps {
  isDirty: boolean
  saving: boolean
  disabled?: boolean
  onClick: () => unknown
  onCancel?: () => unknown
}

export function DraftControls(props: DraftControlsProps) {
  const id = useId()
  const register = useDraftActions()
  const latest = useRef(props)
  latest.current = props

  const hasDiscard = !!props.onCancel

  useEffect(() => {
    register(id, {
      isDirty: props.isDirty,
      saving: props.saving,
      disabled: props.disabled,
      save: () => latest.current.onClick(),
      discard: hasDiscard ? () => latest.current.onCancel?.() : undefined,
    })
    return () => register(id, null)
  }, [id, register, props.isDirty, props.saving, props.disabled, hasDiscard])

  return null
}
