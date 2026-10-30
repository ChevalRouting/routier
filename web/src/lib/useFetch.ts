import { useState, useEffect, useCallback, useRef } from 'react'
import { useDataVersion } from './dataVersion'

type UseFetchShape<T> = {
    data: T | null
    isLoading: boolean
    error: string | null
  }

export function useFetch<T>(fn: () => Promise<T>) {
  const fnRef = useRef(fn)
  fnRef.current = fn
  const { version, track } = useDataVersion()
  const requestRef = useRef(0)
  const mounted = useRef(false)

  const [state, setState] = useState<UseFetchShape<T>>({ data: null, isLoading: true, error: null })

  const load = useCallback(async (invalidate = false) => {
    const request = ++requestRef.current
    setState((previous) => ({ ...previous, isLoading: invalidate || previous.data === null, error: null }))
    try {
      const result = await fnRef.current()
      if (request !== requestRef.current) return
      setState({ data: result, isLoading: false, error: null })
    } catch (e) {
      if (request !== requestRef.current) return
      setState((previous) => ({ ...previous, isLoading: false, error: (e as Error).message }))
    }
  }, [])

  useEffect(() => {
    const refresh = mounted.current
    mounted.current = true
    const done = track()
    void load(!refresh).finally(done)
    return () => { requestRef.current += 1; done() }
  }, [load, version, track])

  const { isLoading } = state
  return { data: isLoading ? null : state.data, latest: state.data, hasData: state.data !== null, isLoading, error: state.error, reload: load }
}
