import { useCallback, useState } from 'react'

export function useClipboard(resetMs = 1800) {
  const [copied, setCopied] = useState(false)
  const copy = useCallback((text: string) => {
    void navigator.clipboard
      .writeText(text)
      .then(() => {
        setCopied(true)
        setTimeout(() => setCopied(false), resetMs)
      })
      .catch(() => {})
  }, [resetMs])
  return { copied, copy }
}
