import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { Button, ComboRow, CopyButton, Dialog, EntryRow, PreferencesGroup, Row, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { KeyRow, WgPeer } from './shared'
import { CodeBlock } from '@/components/ui/CodeBlock'

type VrrpShape = { vips?: string[] }

const dhcpTokens = ['dhcp', 'dhcp4', 'dhcp6', 'slaac']

interface ExportIface {
  addresses?: string[]
}

interface ExportHA {
  vrrp?: VrrpShape[]
}

function serverIPs(ifaces: Record<string, ExportIface> | null, ha: ExportHA | null): string[] {
  const raw = [
    ...Object.values(ifaces ?? {}).flatMap((i) => i.addresses ?? []),
    ...(ha?.vrrp ?? []).flatMap((v) => v.vips ?? []),
  ]

  const ips = raw
    .map((a) => a.split('/')[0])
    .filter((ip) => ip && !dhcpTokens.includes(ip))

  return Array.from(new Set(ips))
}

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

  const { data: ifaces } = useFetch<Record<string, ExportIface>>(
    () => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<Record<string, ExportIface>>,
  )
  const { data: ha } = useFetch<ExportHA>(
    () => api.apiConfigSectionGet({ section: 'ha' }) as Promise<ExportHA>,
  )
  const hostIPs = serverIPs(ifaces, ha)

  const selectHost = (ip: string) => {
    const port = endpoint.match(/:(\d+)$/)?.[1] ?? String(serverPort || 51820)
    const host = ip.includes(':') ? `[${ip}]` : ip
    setEndpoint(`${host}:${port}`)
  }

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
      clientPrivKey ? `PrivateKey = ${clientPrivKey}` : '# PrivateKey = <your private key>',
    ]
    if (clientAddresses) lines.push(`Address = ${clientAddresses}`)
    if (dns) lines.push(`DNS = ${dns}`)
    lines.push('')
    lines.push('[Peer]')
    lines.push(`PublicKey = ${serverPubKey || '<server public key>'}`)
    if (peer.preshared_key) lines.push(`PresharedKey = ${peer.preshared_key}`)
    lines.push(`Endpoint = ${endpoint}`)
    lines.push(`AllowedIPs = ${(peer.allowed_ips ?? []).length ? peer.allowed_ips.join(', ') : '0.0.0.0/0, ::/0'}`)
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
      title="Export Peer Config"
      description={<>{peer.name ? `“${peer.name}”` : 'Unnamed peer'} on interface <span className="font-mono">{ifaceName}</span></>}
      className="max-w-2xl"
      footer={
        <>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button variant="suggested" onClick={downloadConf}>Download</Button>
        </>
      }
    >
      <div className="space-y-6">
        <PreferencesGroup
          title="Client Key Pair"
          header={(
            <Button variant="outline" size="sm" onClick={generateClientKeys} disabled={generating}>
              {generating ? 'Generating…' : 'Generate New Keys'}
            </Button>
          )}
          description={keysChanged ? 'Downloading saves the new key pair to this peer' : undefined}
        >
          <KeyRow title="Private Key" value={clientPrivKey} onChange={setClientPrivKey} placeholder="Base64 private key" />
          <EntryRow
            title="Public Key"
            value={clientPubKey}
            onChange={(e) => setClientPubKey(e.target.value)}
            className="font-mono"
            placeholder="Base64 public key"
            help="Stored on the router as this peer's key"
            suffix={clientPubKey ? <CopyButton text={clientPubKey} /> : undefined}
          />
        </PreferencesGroup>

        <PreferencesGroup title="Client Interface">
          <EntryRow title="Addresses" value={clientAddresses} onChange={(e) => setClientAddresses(e.target.value)} placeholder="10.0.0.2/32" className="font-mono" />
          <EntryRow title="DNS" value={dns} onChange={(e) => setDns(e.target.value)} placeholder="1.1.1.1" className="font-mono" />
        </PreferencesGroup>

        <PreferencesGroup title="Server Peer">
          <EntryRow title="Endpoint" value={endpoint} onChange={(e) => setEndpoint(e.target.value)} placeholder="vpn.example.com:51820" className="font-mono" help="Add the hostname before the colon" />
          {hostIPs.length > 0 && (
            <ComboRow title="Use Address">
              <Select value="" onValueChange={selectHost}>
                <SelectTrigger><SelectValue placeholder="Interface or VRRP address" /></SelectTrigger>
                <SelectContent>
                  {hostIPs.map((ip) => <SelectItem key={ip} value={ip} className="font-mono">{ip}</SelectItem>)}
                </SelectContent>
              </Select>
            </ComboRow>
          )}
          <Row title="Public Key">
            <span className="truncate font-mono text-sm">{serverPubKey || 'Set a private key on the interface'}</span>
            {serverPubKey && <CopyButton text={serverPubKey} />}
          </Row>
          <Row title="Allowed IPs" subtitle="Inherited from the peer">
            <span className="font-mono text-sm">{(peer.allowed_ips ?? []).length ? peer.allowed_ips.join(', ') : '0.0.0.0/0, ::/0'}</span>
          </Row>
        </PreferencesGroup>

        <PreferencesGroup title="Config Preview">
          <CodeBlock className="max-h-52 rounded-none">{conf}</CodeBlock>
        </PreferencesGroup>
      </div>
    </Dialog>
  )
}

