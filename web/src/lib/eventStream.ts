import { activeBaseUrl } from './instance'
import { getToken } from './utils'

export class StreamError extends Error {
  constructor(public status: number) {
    super(`Stream failed (HTTP ${status})`)
  }
}

function frameData(frame: string): string {
  return frame
    .split('\n')
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trimStart())
    .join('\n')
}

export async function readEventStream(path: string, signal: AbortSignal, onData: (data: string) => void, onOpen?: () => void): Promise<void> {
  const response = await fetch(`${activeBaseUrl()}${path}`, {
    headers: { Authorization: `Bearer ${getToken() ?? ''}` },
    signal,
  })
  if (!response.ok || !response.body) throw new StreamError(response.status)
  onOpen?.()

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (!signal.aborted) {
    const { done, value } = await reader.read()
    if (done) return
    buffer += decoder.decode(value, { stream: true })
    const frames = buffer.split('\n\n')
    buffer = frames.pop() ?? ''
    for (const frame of frames) {
      const data = frameData(frame)
      if (data) onData(data)
    }
  }
}
