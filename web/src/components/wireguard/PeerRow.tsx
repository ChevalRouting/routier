import { useState } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import { Button } from 'cheval-ui'
import { Input } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { TagInput } from 'cheval-ui'
import { NumberInput } from 'cheval-ui'
import { Eye, EyeOff, Shuffle, Download } from 'lucide-react'
import { CopyButton } from 'cheval-ui'
import { checkCIDR } from '@/lib/validate'
import { Segmented } from 'cheval-ui'
import { WgPeer, truncateKey, KeyField } from './shared'
import { ExportModal } from './ExportModal'

export function PeerSummary({ peer }: { peer: WgPeer }) {
  return (
    <div className="flex min-w-0 flex-1 items-center gap-2">
      <span className="min-w-0 flex-1 truncate text-sm font-medium">
        {peer.name || <span className="text-xs italic text-muted-foreground">Unnamed peer</span>}
      </span>
      <span className="hidden shrink-0 font-mono text-xs text-muted-foreground sm:block">
        {truncateKey(peer.public_key)}
      </span>
      {peer.endpoint && (
        <span className="hidden shrink-0 text-xs text-muted-foreground md:block">{peer.endpoint}</span>
      )}
      {(peer.allowed_ips ?? []).length > 0 && (
        <Badge variant="outline" className="hidden shrink-0 text-[10px] lg:flex">
          {(peer.allowed_ips ?? []).length === 1 ? peer.allowed_ips[0] : `${(peer.allowed_ips ?? []).length} routes`}
        </Badge>
      )}
    </div>
  )
}

export function PeerActions({ peer, serverPubKey, serverPort, serverAddresses, ifaceName, onChange }: {
  peer: WgPeer
  serverPubKey: string
  serverPort: number
  serverAddresses: string[]
  ifaceName: string
  onChange: (updated: WgPeer) => void
}) {
  const [showExport, setShowExport] = useState(false)

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => setShowExport(true)}
        className="h-7 shrink-0 gap-1 px-2 text-xs"
        title="Export client config"
      >
        <Download className="h-3 w-3" />
        Export
      </Button>

      {showExport && (
        <ExportModal
          peer={peer}
          serverPubKey={serverPubKey}
          serverPort={serverPort}
          serverAddresses={serverAddresses}
          ifaceName={ifaceName}
          onClose={() => setShowExport(false)}
          onUpdatePeerPubKey={(privKey, pubKey) => {
            onChange({ ...peer, private_key: privKey, public_key: pubKey })
            setShowExport(false)
          }}
        />
      )}
    </>
  )
}

export function PeerBody({ peer, onChange }: { peer: WgPeer; onChange: (updated: WgPeer) => void }) {
  const [showPsk, setShowPsk] = useState(false)
  const [pskMode, setPskMode] = useState<'inline' | 'file'>(peer.preshared_key_file ? 'file' : 'inline')

  const set = <K extends keyof WgPeer>(key: K, val: WgPeer[K]) =>
    onChange({ ...peer, [key]: val })

  const switchPskMode = (mode: 'inline' | 'file') => {
    setPskMode(mode)
    onChange({ ...peer, preshared_key: '', preshared_key_file: '' })
  }

  return (
    <>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label className="text-xs">Peer name</Label>
          <Input value={peer.name} onChange={(e) => set('name', e.target.value)}
            placeholder="laptop, phone…" className="text-sm" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-xs">Endpoint <span className="text-muted-foreground">(optional, for initiating)</span></Label>
          <Input value={peer.endpoint} onChange={(e) => set('endpoint', e.target.value)}
            placeholder="peer.example.com:51820" className="font-mono text-sm" />
        </div>
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-xs">Client key pair</Label>
          <Button variant="outline" size="sm" onClick={async () => {
            try {
              const { private_key, public_key } = await api.apiWireguardKeygenPost()
              onChange({ ...peer, private_key, public_key })
            } catch { toast.error('Key generation failed') }
          }} className="h-6 gap-1 px-2 text-xs">
            <Shuffle className="h-3 w-3" />Generate
          </Button>
        </div>
        <div className="space-y-1.5">
          <Label className="text-[11px] text-muted-foreground">Private key <span className="italic">(stored server-side for export)</span></Label>
          <KeyField value={peer.private_key ?? ''} onChange={(v) => set('private_key', v)} placeholder="Base64 private key, optional" />
        </div>
        <div className="space-y-1.5">
          <Label className="text-[11px] text-muted-foreground">Public key</Label>
          <div className="flex items-center gap-2">
            <Input value={peer.public_key} onChange={(e) => set('public_key', e.target.value)}
              className="font-mono text-sm" placeholder="Base64 public key" />
            {peer.public_key && <CopyButton text={peer.public_key} />}
          </div>
        </div>
      </div>

      <div className="space-y-2">
        <Label className="text-xs">Preshared key <span className="text-muted-foreground">(optional)</span></Label>
        <Segmented
          value={pskMode}
          onChange={switchPskMode}
          options={[{ value: 'inline', label: 'Inline' }, { value: 'file', label: 'File path' }]}
        />
        {pskMode === 'inline' ? (
          <div className="relative flex items-center">
            <Input
              type={showPsk ? 'text' : 'password'}
              value={peer.preshared_key}
              onChange={(e) => set('preshared_key', e.target.value)}
              className="font-mono text-sm pr-16"
              placeholder="Base64 preshared key"
            />
            <div className="absolute right-2 flex items-center gap-1">
              {peer.preshared_key && <CopyButton text={peer.preshared_key} />}
              <button type="button" onClick={() => setShowPsk(!showPsk)}
                className="text-muted-foreground hover:text-foreground">
                {showPsk ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
              </button>
            </div>
          </div>
        ) : (
          <Input value={peer.preshared_key_file}
            onChange={(e) => set('preshared_key_file', e.target.value)}
            className="font-mono text-sm" placeholder="/etc/wireguard/psk.key" />
        )}
      </div>

      <div className="space-y-1.5">
        <Label className="text-xs">Allowed IPs</Label>
        <TagInput values={peer.allowed_ips ?? []} onChange={(v) => set('allowed_ips', v)}
          placeholder="10.0.0.2/32" mono validate={checkCIDR} />
      </div>

      <div className="space-y-1.5">
        <Label className="text-xs">Persistent keepalive <span className="text-muted-foreground">(seconds, 0 = off)</span></Label>
        <NumberInput min={0} value={peer.keepalive || undefined}
          onChange={(v) => set('keepalive', v ?? 0)}
          className="w-32 font-mono" placeholder="0" />
      </div>
    </>
  )
}
