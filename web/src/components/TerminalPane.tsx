import { useEffect, useRef } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { getToken } from '@/lib/utils'
import '@xterm/xterm/css/xterm.css'

interface TerminalPaneProps {
  wsPath: string
  className?: string
  onOpen?: () => void
  onClose?: (clean: boolean) => void
  quiet?: boolean
}

export function TerminalPane({ wsPath, className, onOpen, onClose, quiet }: TerminalPaneProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const onOpenRef = useRef(onOpen)
  const onCloseRef = useRef(onClose)
  const quietRef = useRef(quiet)
  onOpenRef.current = onOpen
  onCloseRef.current = onClose
  quietRef.current = quiet

  useEffect(() => {
    const el = containerRef.current
    if (!el) return

    let unmounting = false

    const term = new Terminal({
      theme: {
        background: '#09090b',
        foreground: '#e4e4e7',
        cursor: '#a1a1aa',
        selectionBackground: '#3f3f46',
      },
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
      fontSize: 13,
      lineHeight: 1.4,
      cursorBlink: true,
      convertEol: true,
      scrollback: 5000,
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(el)
    fitAddon.fit()

    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const token = getToken() ?? ''
    const sep = wsPath.includes('?') ? '&' : '?'
    const wsUrl = `${proto}//${window.location.host}${wsPath}${sep}token=${encodeURIComponent(token)}`

    let ws: WebSocket | null = null

    const sendResize = () => {
      fitAddon.fit()
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
    }

    const openTimer = setTimeout(() => {
      if (unmounting) return
      ws = new WebSocket(wsUrl)
      ws.binaryType = 'arraybuffer'

      ws.onmessage = (e) => {
        if (e.data instanceof ArrayBuffer) {
          term.write(new Uint8Array(e.data))
        } else {
          term.write(e.data as string)
        }
      }

      ws.onclose = (e) => {
        if (!quietRef.current) term.write('\r\n\x1b[2m[connection closed]\x1b[0m\r\n')
        if (!unmounting) onCloseRef.current?.(e.code === 1000 || e.code === 1005)
      }

      ws.onerror = () => {
        if (!quietRef.current) term.write('\r\n\x1b[31m[connection error]\x1b[0m\r\n')
      }

      ws.onopen = () => {
        sendResize()
        onOpenRef.current?.()
      }
    }, 0)

    term.onData((data) => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }))
      }
    })

    const ro = new ResizeObserver(sendResize)
    ro.observe(el)

    return () => {
      unmounting = true
      clearTimeout(openTimer)
      ro.disconnect()
      ws?.close()
      term.dispose()
    }
  }, [wsPath])

  return (
    <div className={className} style={{ height: '100%', background: '#09090b', padding: '10px 12px', boxSizing: 'border-box', display: 'flex', flexDirection: 'column' }}>
      <div ref={containerRef} style={{ flex: 1, minHeight: 0 }} />
    </div>
  )
}
