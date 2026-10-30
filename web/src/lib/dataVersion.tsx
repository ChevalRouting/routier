import { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react'

type DataVersionProviderShape = { children: React.ReactNode }

interface DataVersionContextValue {
  version: number
  settled: number
  bump: () => void
  track: () => () => void
}

const DataVersionContext = createContext<DataVersionContextValue>({ version: 0, settled: 0, bump: () => {}, track: () => () => {} })

export function DataVersionProvider({ children }: DataVersionProviderShape) {
  const [version, setVersion] = useState(0)
  const [settled, setSettled] = useState(0)
  const versionRef = useRef(0)
  const pending = useRef(0)

  const settle = useCallback(() => {
    if (pending.current === 0) setSettled(versionRef.current)
  }, [])

  const bump = useCallback(() => {
    versionRef.current += 1
    setVersion(versionRef.current)
  }, [])

  const track = useCallback(() => {
    pending.current += 1
    let done = false
    return () => {
      if (done) return
      done = true
      pending.current -= 1
      settle()
    }
  }, [settle])

  useEffect(() => { settle() }, [version, settle])

  return (
    <DataVersionContext.Provider value={{ version, settled, bump, track }}>
      {children}
    </DataVersionContext.Provider>
  )
}

export function useDataVersion() {
  return useContext(DataVersionContext)
}

export function useDataRefresh(callback: () => void) {
  const { settled } = useDataVersion()
  const cbRef = useRef(callback)
  cbRef.current = callback
  const first = useRef(true)
  useEffect(() => {
    if (first.current) { first.current = false; return }
    cbRef.current()
  }, [settled])
}
