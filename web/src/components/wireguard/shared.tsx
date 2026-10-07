import { CopyButton, Input, Label } from 'cheval-ui'
import { Eye, EyeOff } from 'lucide-react'
import { useState } from 'react'

type KeyFieldShape = {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  label?: string
  mono?: boolean
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

export function KeyField({
  value, onChange, placeholder, label, mono = true,
}: KeyFieldShape) {
  const [show, setShow] = useState(false)
  return (
    <div className="relative">
      {label && <Label className="text-xs mb-1.5 block">{label}</Label>}
      <div className="relative flex items-center">
        <Input
          type={show ? 'text' : 'password'}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={`pr-16 ${mono ? 'font-mono' : ''} text-sm`}
          placeholder={placeholder ?? 'Base64 key'}
        />
        <div className="absolute right-2 flex items-center gap-1">
          {value && <CopyButton text={value} />}
          <button type="button" onClick={() => setShow(!show)}
            className="text-muted-foreground hover:text-foreground transition-colors">
            {show ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
          </button>
        </div>
      </div>
    </div>
  )
}
