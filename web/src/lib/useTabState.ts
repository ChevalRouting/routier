import { useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'

export function useTabState<T extends string>(key: string, defaultValue: T): [T, (v: T) => void] {
  const [params, setParams] = useSearchParams()
  const value = (params.get(key) as T) || defaultValue

  const set = useCallback(
    (v: T) => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          const childPrefix = `${key}.`
          for (const k of Array.from(next.keys())) {
            if (k.startsWith(childPrefix)) next.delete(k)
          }
          if (v === defaultValue) {
            next.delete(key)
          } else {
            next.set(key, v)
          }
          return next
        },
        { replace: true },
      )
    },
    [key, defaultValue, setParams],
  )

  return [value, set]
}
