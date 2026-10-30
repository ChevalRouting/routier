import { CopyButton, EntryRow } from 'cheval-ui'

type KeyRowShape = {
  title: string
  value: string
  onChange: (v: string) => void
  placeholder?: string
  help?: React.ReactNode
}

export interface WgPeer {
  name: string
  private_key: string
  public_key: string
  preshared_key: string
  preshared_key_file: string
  endpoint: string
  allowed_ips: string[]
  keepalive: number
}

export interface WgIface {
  friend?: string
  listen_port: number
  private_key: string
  private_key_file: string
  table: string
  addresses: string[]
  mtu: number
  pre_up: string[]
  post_up: string[]
  pre_down: string[]
  post_down: string[]
  peers: WgPeer[]
  allow_inbound?: boolean
}

export type WgMap = Record<string, WgIface>

export function emptyWg(): WgIface {
  return {
    listen_port: 51820,
    private_key: '',
    private_key_file: '',
    table: 'auto',
    addresses: [],
    mtu: 0,
    pre_up: [],
    post_up: [],
    pre_down: [],
    post_down: [],
    peers: [],
  }
}

export function emptyPeer(): WgPeer {
  return { name: '', private_key: '', public_key: '', preshared_key: '', preshared_key_file: '', endpoint: '', allowed_ips: [], keepalive: 0 }
}

export function truncateKey(key: string): string {
  if (!key || key.length <= 16) return key || '-'
  return key.slice(0, 8) + '…' + key.slice(-6)
}

export function KeyRow({ title, value, onChange, placeholder, help }: KeyRowShape) {
  return (
    <EntryRow
      title={title}
      type="password"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder ?? 'Base64 key'}
      className="font-mono"
      help={help}
      suffix={value ? <CopyButton text={value} /> : undefined}
    />
  )
}
