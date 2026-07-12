import { useState, useEffect } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Dialog } from '@/components/ui/dialog'
import { Shuffle, Download } from 'lucide-react'
import { CopyButton } from '@/components/CopyButton'
import { WgPeer, KeyField } from './shared'

export interface ExportModalProps {
  peer: WgPeer
  serverPubKey: string
  serverPort: number
  serverAddresses: string[]
  ifaceName: string
  onClose: () => void
  onUpdatePeerPubKey: (privKey: string, pubKey: string) => void
}

export function ExportModal({ peer, serverPubKey, serverPort, serverAddresses: _serverAddresses, ifaceName, onClose, onUpdatePeerPubKey }: ExportModalProps) {
  const [clientPrivKey, setClientPrivKey] = useState(peer.private_key ?? '')
  const [clientPubKey, setClientPubKey] = useState(peer.public_key)
  const [clientAddresses, setClientAddresses] = useState('')
  const [dns, setDns] = useState('')
  const [endpoint, setEndpoint] = useState(() => serverPort ? `:${serverPort}` : ':51820')
  const [generating, setGenerating] = useState(false)
  const [keysChanged, setKeysChanged] = useState(false)

  useEffect(() => {
    if (!clientPrivKey || clientPrivKey.length < 40) return
    api.apiWireguardPubkeyPost({ body: clientPrivKey })
      .then((r) => setClientPubKey(r.public_key ?? ''))
      .catch(() => {})
  }, [clientPrivKey])

  const generateClientKeys = async () => {
    setGenerating(true)
    try {
      const { private_key, public_key } = await api.apiWireguardKeygenPost()
      setClientPrivKey(private_key ?? '')
      setClientPubKey(public_key ?? '')
      setKeysChanged(true)
    } catch {
      toast.error('Key generation failed')
    } finally {
      setGenerating(false)
    }
  }

  const buildConf = () => {
    const lines: string[] = [
      '[Interface]',
      clientPrivKey ? `PrivateKey = ${clientPrivKey}` : `# PrivateKey = <your private key>`,
    ]
    if (clientAddresses) lines.push(`Address = ${clientAddresses}`)
    if (dns) lines.push(`DNS = ${dns}`)
    lines.push('')
    lines.push('[Peer]')
    lines.push(`PublicKey = ${serverPubKey || '<server public key>'}`)
    if (peer.preshared_key) lines.push(`PresharedKey = ${peer.preshared_key}`)
    lines.push(`Endpoint = ${endpoint}`)
    lines.push(`AllowedIPs = ${peer.allowed_ips.length ? peer.allowed_ips.join(', ') : '0.0.0.0/0, ::/0'}`)
    if (peer.keepalive) lines.push(`PersistentKeepalive = ${peer.keepalive}`)
    return lines.join('\n')
  }

  const downloadConf = () => {
    const conf = buildConf()
    const blob = new Blob([conf], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${ifaceName}-${peer.name || 'peer'}.conf`
    a.click()
    URL.revokeObjectURL(url)

    if (keysChanged || clientPrivKey !== peer.private_key || clientPubKey !== peer.public_key) {
      onUpdatePeerPubKey(clientPrivKey, clientPubKey)
      toast.success('Peer keys saved to config')
    }
  }

  const conf = buildConf()

  return (
    <Dialog
      open
      onClose={onClose}
      title="Export peer config"
      description={<>{peer.name ? `“${peer.name}”` : 'Unnamed peer'} on interface <span className="font-mono">{ifaceName}</span></>}
      className="max-w-2xl"
      footer={
        <>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={downloadConf} className="gap-2">
            <Download className="h-4 w-4" />
            Download .conf
          </Button>
        </>
      }
    >
      <div className="space-y-5">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-semibold">Client key pair</Label>
              <Button variant="outline" size="sm" onClick={generateClientKeys} disabled={generating} className="gap-1.5 h-7 text-xs">
                <Shuffle className="h-3.5 w-3.5" />
                {generating ? 'Generating…' : 'Generate new keys'}
              </Button>
            </div>
            <div className="space-y-2">
              <KeyField
                label="Client Private Key"
                value={clientPrivKey}
                onChange={setClientPrivKey}
                placeholder="Leave blank if managing keys yourself"
              />
              <div className="space-y-1">
                <Label className="text-xs text-muted-foreground">Client public key (stored on server as this peer's key)</Label>
                <div className="flex items-center gap-2">
                  <Input
                    value={clientPubKey}
                    onChange={(e) => { setClientPubKey(e.target.value) }}
                    className="font-mono text-xs"
                    placeholder="Base64 public key"
                  />
                  {clientPubKey && <CopyButton text={clientPubKey} />}
                </div>
                {keysChanged && (
                  <p className="text-xs text-warning">
                    Downloading will save the new key pair to this peer in the config.
                  </p>
                )}
              </div>
            </div>
          </div>

          <Separator />

          <div className="space-y-3">
            <Label className="text-sm font-semibold">Client interface</Label>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label className="text-xs">Client address(es)</Label>
                <Input value={clientAddresses} onChange={(e) => setClientAddresses(e.target.value)}
                  placeholder="10.0.0.2/32" className="font-mono text-sm" />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs">DNS <span className="text-muted-foreground">(optional)</span></Label>
                <Input value={dns} onChange={(e) => setDns(e.target.value)}
                  placeholder="1.1.1.1" className="font-mono text-sm" />
              </div>
            </div>
          </div>

          <Separator />

          <div className="space-y-3">
            <Label className="text-sm font-semibold">Server peer</Label>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label className="text-xs">Server endpoint</Label>
                <Input value={endpoint} onChange={(e) => setEndpoint(e.target.value)}
                  placeholder="vpn.example.com:51820" className="font-mono text-sm" />
                <p className="text-[11px] text-muted-foreground">hostname:port, add your hostname before the colon</p>
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs">Server public key</Label>
                <div className="flex items-center gap-2">
                  <Input value={serverPubKey} readOnly className="font-mono text-xs bg-muted" placeholder="(not derivable, set private key on interface)" />
                  {serverPubKey && <CopyButton text={serverPubKey} />}
                </div>
              </div>
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Allowed IPs (routes sent through tunnel)</Label>
              <Input
                value={peer.allowed_ips.length ? peer.allowed_ips.join(', ') : '0.0.0.0/0, ::/0'}
                readOnly className="font-mono text-xs bg-muted"
              />
              <p className="text-[11px] text-muted-foreground">Inherited from peer's AllowedIPs. Edit on the peer to change.</p>
            </div>
          </div>

          <Separator />

          <div className="space-y-2">
            <Label className="text-sm font-semibold">Config preview</Label>
            <pre className="bg-zinc-950 text-zinc-100 font-mono text-xs p-3 rounded-md overflow-auto max-h-52 whitespace-pre-wrap">
              {conf}
            </pre>
          </div>
      </div>
    </Dialog>
  )
}

