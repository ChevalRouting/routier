import { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react'

type DataVersionProviderShape = { children: React.ReactNode }

interface DataVersionContextValue {
  version: number
  bump: () => void
}

const DataVersionContext = createContext<DataVersionContextValue>({ version: 0, bump: () => {} })

export function DataVersionProvider({ children }: DataVersionProviderShape) {
  const [version, setVersion] = useState(0)
  const bump = useCallback(() => setVersion((v) => v + 1), [])
  return (
    <DataVersionContext.Provider value={{ version, bump }}>
      {children}
    </DataVersionContext.Provider>
  )
}

export function useDataVersion() {
  return useContext(DataVersionContext)
}

export function useDataRefresh(callback: () => void) {
  const { version } = useDataVersion()
  const cbRef = useRef(callback)
  cbRef.current = callback
  const first = useRef(true)
  useEffect(() => {
    if (first.current) { first.current = false; return }
    cbRef.current()
  }, [version])
}
