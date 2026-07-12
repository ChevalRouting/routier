import { useState, useEffect, useCallback, useRef } from 'react'
import { useDataVersion } from './dataVersion'

export function useFetch<T>(fn: () => Promise<T>) {
  const fnRef = useRef(fn)
  fnRef.current = fn

  const [data, setData] = useState<T | null>(null)
  const dataRef = useRef<T | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const { version } = useDataVersion()

  const load = useCallback(async () => {
    const silent = dataRef.current !== null
    if (!silent) setIsLoading(true)
    setError(null)
    try {
      const result = await fnRef.current()
      dataRef.current = result
      setData(result)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      if (!silent) setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load, version])

  return { data, isLoading, error, reload: load }
}
