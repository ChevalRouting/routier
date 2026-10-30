import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'

export function useHasParam(key: string): boolean {
  const [params] = useSearchParams()
  const [present] = useState(() => params.has(key))
  return present
}
