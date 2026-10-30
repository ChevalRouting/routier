import { AccordionList } from '@/components/ui/AccordionList'
import { api } from '@/lib/client'
import { checkCIDR, checkMTU, checkPort } from '@/lib/validate'
import { Button, ComboRow, CopyButton, EntryRow, PreferencesGroup, Segmented, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, SwitchRow, TagInput, Row } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { PeerActions, PeerBody, PeerSummary } from './PeerRow'
import { emptyPeer, KeyRow, WgIface, WgPeer } from './shared'

interface WorkingPeer { uid: string; p: WgPeer }

export interface WgFormProps {
  name: string
  iface: WgIface
  onChange: (updated: WgIface) => void
  onNameChange?: (n: string) => void
  onAdd?: () => void
  onDone: () => void
}

export function WgForm({ name, iface, onChange, onNameChange, onAdd, onDone }: WgFormProps) {
  const [peerItems, setPeerItems] = useState<WorkingPeer[]>(() => (iface.peers ?? []).map((p) => ({ uid: crypto.randomUUID(), p })))
  const [keyMode, setKeyMode] = useState<'inline' | 'file'>(iface.private_key_file ? 'file' : 'inline')
  const [tableMode, setTableMode] = useState<'off' | 'auto' | 'custom'>(
    iface.table === 'off' ? 'off' : iface.table === 'auto' ? 'auto' : 'custom'
  )
  const [generatingKey, setGeneratingKey] = useState(false)
  const [serverPubKey, setServerPubKey] = useState('')

  const set = <K extends keyof WgIface>(key: K, val: WgIface[K]) =>
    onChange({ ...iface, [key]: val })

  useEffect(() => {
    const pk = iface.private_key
    if (!pk || pk.length < 40) { setServerPubKey(''); return }
    api.apiWireguardPubkeyPost({ body: pk })
      .then((r) => setServerPubKey(r.public_key ?? ''))
      .catch(() => setServerPubKey(''))
  }, [iface.private_key])

  const generateKey = async () => {
    setGeneratingKey(true)
    try {
      const { private_key, public_key } = await api.apiWireguardKeygenPost()
      onChange({ ...iface, private_key, private_key_file: '' })
      setKeyMode('inline')
      setServerPubKey(public_key ?? '')
      toast.success('New key pair generated')
    } catch {
      toast.error('Key generation failed')
    } finally {
      setGeneratingKey(false)
    }
  }

  const switchKeyMode = (mode: 'inline' | 'file') => {
    setKeyMode(mode)
    onChange({ ...iface, private_key: '', private_key_file: '' })
    setServerPubKey('')
  }

  const switchTableMode = (mode: 'off' | 'auto' | 'custom') => {
    setTableMode(mode)
    if (mode !== 'custom') onChange({ ...iface, table: mode })
    else onChange({ ...iface, table: '' })
  }

  const commitPeers = (next: WorkingPeer[]) => {
    setPeerItems(next)
    onChange({ ...iface, peers: next.map((x) => x.p) })
  }
  const updatePeer = (uid: string, peer: WgPeer) => commitPeers(peerItems.map((x) => (x.uid === uid ? { ...x, p: peer } : x)))
  const deletePeer = (uid: string) => commitPeers(peerItems.filter((x) => x.uid !== uid))
  const addPeer = () => commitPeers([...peerItems, { uid: crypto.randomUUID(), p: emptyPeer() }])

  return (
    <>
      <PreferencesGroup>
        {onNameChange !== undefined && (
          <EntryRow title="Name" autoFocus value={name} onChange={(e) => onNameChange(e.target.value)}
            placeholder="wg0" className="font-mono" />
        )}
        <EntryRow title="Listen Port" inputMode="numeric" value={iface.listen_port || ''}
          onChange={(e) => set('listen_port', Number(e.target.value) || 51820)} placeholder="51820" className="font-mono" error={checkPort(String(iface.listen_port || ''))} />
        <EntryRow title="MTU (0 = auto)" inputMode="numeric" value={iface.mtu || ''}
          onChange={(e) => set('mtu', Number(e.target.value) || 0)} placeholder="0" className="font-mono" error={iface.mtu ? checkMTU(String(iface.mtu)) : null} />
        <ComboRow title="Routing Table">
          <Select value={tableMode} onValueChange={(v) => switchTableMode(v as 'off' | 'auto' | 'custom')}>
            <SelectTrigger className="h-9 w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="auto">auto</SelectItem>
              <SelectItem value="off">off</SelectItem>
              <SelectItem value="custom">custom…</SelectItem>
            </SelectContent>
          </Select>
        </ComboRow>
        {tableMode === 'custom' && (
          <EntryRow title="Custom Table Number" inputMode="numeric" value={iface.table ? Number(iface.table) : ''}
            onChange={(e) => set('table', e.target.value ? String(Number(e.target.value)) : '')}
            placeholder="Table #" className="font-mono" />
        )}
        <SwitchRow title="Allow Inbound"
          subtitle="Open this interface's listen port in the firewall (routier input chain)"
          checked={iface.allow_inbound ?? false}
          onCheckedChange={(v) => set('allow_inbound', v)} />
      </PreferencesGroup>

      <PreferencesGroup
        title="Private Key"
        header={(
          <div className="flex items-center gap-2">
            <Segmented
              value={keyMode}
              onChange={switchKeyMode}
              options={[{ value: 'inline', label: 'Inline' }, { value: 'file', label: 'File' }]}
            />
            {keyMode === 'inline' && (
              <Button variant="outline" size="sm" onClick={generateKey} disabled={generatingKey}>
                {generatingKey ? 'Generating…' : 'Generate'}
              </Button>
            )}
          </div>
        )}
      >
        {keyMode === 'inline' ? (
          <>
            <KeyRow title="Private Key" value={iface.private_key} onChange={(v) => set('private_key', v)} placeholder="Base64 private key" />
            {serverPubKey && (
              <Row title="Public Key">
                <span className="truncate font-mono text-sm">{serverPubKey}</span>
                <CopyButton text={serverPubKey} />
              </Row>
            )}
          </>
        ) : (
          <EntryRow title="Key File" value={iface.private_key_file} onChange={(e) => set('private_key_file', e.target.value)} className="font-mono" placeholder="/etc/wireguard/private.key" />
        )}
      </PreferencesGroup>

      <PreferencesGroup>
        <ComboRow title="Addresses">
          <TagInput values={iface.addresses} onChange={(v) => set('addresses', v)} placeholder="10.0.0.1/24" mono validate={checkCIDR} />
        </ComboRow>
      </PreferencesGroup>

      <PreferencesGroup title="Hooks">
        {(
          [['pre_up', 'PreUp'], ['post_up', 'PostUp'], ['pre_down', 'PreDown'], ['post_down', 'PostDown']] as [keyof WgIface, string][]
        ).map(([key, label]) => (
          <ComboRow key={key} title={label}>
            <TagInput values={iface[key] as string[]} onChange={(v) => set(key, v)} placeholder="command or script" mono />
          </ComboRow>
        ))}
      </PreferencesGroup>

      <AccordionList
        items={peerItems}
        getId={(x) => x.uid}
        description="Peers"
        addLabel="Add Peer"
        onAdd={addPeer}
        onRemove={(x) => deletePeer(x.uid)}
        emptyTitle="No Peers"
        renderSummary={(x) => <PeerSummary peer={x.p} />}
        renderActions={(x) => (
          <PeerActions
            peer={x.p}
            serverPubKey={serverPubKey}
            serverPort={iface.listen_port}
            serverAddresses={iface.addresses ?? []}
            ifaceName={name}
            onChange={(u) => updatePeer(x.uid, u)}
          />
        )}
        renderBody={(x) => <PeerBody peer={x.p} onChange={(u) => updatePeer(x.uid, u)} />}
      />

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Close'}</Button>
        {onAdd && <Button onClick={onAdd}>Add Interface</Button>}
      </div>
    </>
  )
}
