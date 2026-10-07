import { useState, useEffect, useCallback, useRef } from 'react'
import { useDataVersion } from './dataVersion'

type UseFetchShape<T> = {
    data: T | null
    version: number
    isLoading: boolean
    error: string | null
  }

export function useFetch<T>(fn: () => Promise<T>) {
  const fnRef = useRef(fn)
  fnRef.current = fn
  const { version } = useDataVersion()
  const versionRef = useRef(version)
  versionRef.current = version
  const requestRef = useRef(0)

  const [state, setState] = useState<UseFetchShape<T>>({ data: null, version, isLoading: true, error: null })

  const load = useCallback(async (invalidate = false) => {
    const request = ++requestRef.current
    const requestedVersion = versionRef.current
    setState((previous) => ({ ...previous, version: requestedVersion, isLoading: invalidate || previous.data === null, error: null }))
    try {
      const result = await fnRef.current()
      if (request !== requestRef.current || requestedVersion !== versionRef.current) return
      setState({ data: result, version: requestedVersion, isLoading: false, error: null })
    } catch (e) {
      if (request !== requestRef.current || requestedVersion !== versionRef.current) return
      setState((previous) => ({ ...previous, version: requestedVersion, isLoading: false, error: (e as Error).message }))
    }
  }, [])

  useEffect(() => {
    void load(true)
    return () => { requestRef.current += 1 }
  }, [load, version])

  const isLoading = state.isLoading || state.version !== version
  return { data: isLoading ? null : state.data, hasData: state.data !== null, isLoading, error: state.error, reload: load }
}
