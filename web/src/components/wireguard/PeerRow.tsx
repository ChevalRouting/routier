import { api } from '@/lib/client'
import { checkCIDR } from '@/lib/validate'
import { Badge, Button, ComboRow, CopyButton, EntryRow, PreferencesGroup, Segmented, TagInput } from 'cheval-ui'
import { NumberRow } from '@/components/ui/NumberRow'
import { Download } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { ExportModal } from './ExportModal'
import { KeyRow, WgPeer, truncateKey } from './shared'

type PeerSummaryShape = { peer: WgPeer }

type PeerActionsShape = {
  peer: WgPeer
  serverPubKey: string
  serverPort: number
  serverAddresses: string[]
  ifaceName: string
  onChange: (updated: WgPeer) => void
}

type PeerBodyShape = { peer: WgPeer; onChange: (updated: WgPeer) => void }

export function PeerSummary({ peer }: PeerSummaryShape) {
  return (
    <div className="flex min-w-0 flex-1 items-center gap-2">
      <span className="min-w-0 flex-1 truncate text-sm font-medium">
        {peer.name || <span className="text-xs text-muted-foreground">Unnamed peer</span>}
      </span>
      <span className="hidden shrink-0 font-mono text-xs text-muted-foreground sm:block">
        {truncateKey(peer.public_key)}
      </span>
      {peer.endpoint && (
        <span className="hidden shrink-0 text-xs text-muted-foreground md:block">{peer.endpoint}</span>
      )}
      {(peer.allowed_ips ?? []).length > 0 && (
        <Badge variant="outline" className="hidden shrink-0 text-xs lg:flex">
          {(peer.allowed_ips ?? []).length === 1 ? peer.allowed_ips[0] : `${(peer.allowed_ips ?? []).length} routes`}
        </Badge>
      )}
    </div>
  )
}

export function PeerActions({ peer, serverPubKey, serverPort, serverAddresses, ifaceName, onChange }: PeerActionsShape) {
  const [showExport, setShowExport] = useState(false)

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => setShowExport(true)}
        className="h-7 shrink-0 gap-1 px-2 text-xs"
        title="Export Client Config"
      >
        <Download className="h-3 w-3" />
        Export…
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

export function PeerBody({ peer, onChange }: PeerBodyShape) {
  const [pskMode, setPskMode] = useState<'inline' | 'file'>(peer.preshared_key_file ? 'file' : 'inline')

  const set = <K extends keyof WgPeer>(key: K, val: WgPeer[K]) =>
    onChange({ ...peer, [key]: val })

  const switchPskMode = (mode: 'inline' | 'file') => {
    setPskMode(mode)
    onChange({ ...peer, preshared_key: '', preshared_key_file: '' })
  }

  const generateKeys = async () => {
    try {
      const { private_key, public_key } = await api.apiWireguardKeygenPost()
      onChange({ ...peer, private_key, public_key })
    } catch {
      toast.error('Key generation failed')
    }
  }

  return (
    <div className="space-y-6">
      <PreferencesGroup>
        <EntryRow title="Name" value={peer.name} onChange={(e) => set('name', e.target.value)} placeholder="laptop" />
        <EntryRow title="Endpoint" value={peer.endpoint} onChange={(e) => set('endpoint', e.target.value)} placeholder="peer.example.com:51820" className="font-mono" />
        <ComboRow title="Allowed IPs">
          <TagInput values={peer.allowed_ips ?? []} onChange={(v) => set('allowed_ips', v)} placeholder="10.0.0.2/32" mono validate={checkCIDR} />
        </ComboRow>
        <NumberRow title="Persistent Keepalive (s)" subtitle="0 disables it" value={peer.keepalive} onChange={(v) => set('keepalive', v ?? 0)} placeholder="0" />
      </PreferencesGroup>

      <PreferencesGroup title="Client Key Pair" header={<Button variant="outline" size="sm" onClick={generateKeys}>Generate</Button>}>
        <KeyRow title="Private Key" value={peer.private_key ?? ''} onChange={(v) => set('private_key', v)} placeholder="Base64 private key" help="Stored on the router for export" />
        <EntryRow title="Public Key" value={peer.public_key} onChange={(e) => set('public_key', e.target.value)} placeholder="Base64 public key" className="font-mono" suffix={peer.public_key ? <CopyButton text={peer.public_key} /> : undefined} />
      </PreferencesGroup>

      <PreferencesGroup
        title="Preshared Key"
        header={(
          <Segmented
            value={pskMode}
            onChange={switchPskMode}
            options={[{ value: 'inline', label: 'Inline' }, { value: 'file', label: 'File' }]}
          />
        )}
      >
        {pskMode === 'inline'
          ? <KeyRow title="Key" value={peer.preshared_key} onChange={(v) => set('preshared_key', v)} placeholder="Base64 preshared key" />
          : <EntryRow title="Key File" value={peer.preshared_key_file} onChange={(e) => set('preshared_key_file', e.target.value)} className="font-mono" placeholder="/etc/wireguard/psk.key" />}
      </PreferencesGroup>
    </div>
  )
}
