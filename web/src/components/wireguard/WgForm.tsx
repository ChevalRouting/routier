import { useState, useEffect } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import TagInput from '@/components/TagInput'
import { Plus, ChevronDown, ChevronRight, Shuffle } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { CopyButton } from '@/components/CopyButton'
import { checkCIDR, checkPort, checkMTU } from '@/lib/validate'
import { PreferencesGroup, EntryRow, ComboRow, SwitchRow } from '@/components/Preferences'
import { Segmented } from '@/components/ui/segmented'
import { WgPeer, WgIface, emptyPeer, KeyField } from './shared'
import { PeerRow } from './PeerRow'

export interface WgFormProps {
  name: string
  iface: WgIface
  onChange: (updated: WgIface) => void
  onNameChange?: (n: string) => void
  onAdd?: () => void
  onDone: () => void
}

export function WgForm({ name, iface, onChange, onNameChange, onAdd, onDone }: WgFormProps) {
  const [hooksOpen, setHooksOpen] = useState(false)
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

  const peers = iface.peers ?? []
  const updatePeer = (idx: number, peer: WgPeer) => {
    const next = [...peers]; next[idx] = peer; onChange({ ...iface, peers: next })
  }
  const deletePeer = (idx: number) => onChange({ ...iface, peers: peers.filter((_, i) => i !== idx) })
  const addPeer = () => onChange({ ...iface, peers: [...peers, emptyPeer()] })

  return (
    <>
      <PreferencesGroup>
        {onNameChange !== undefined && (
          <EntryRow title="Name" autoFocus value={name} onChange={(e) => onNameChange(e.target.value)}
            placeholder="wg0" className="font-mono" />
        )}
        <EntryRow title="Listen port" inputMode="numeric" value={iface.listen_port || ''}
          onChange={(e) => set('listen_port', Number(e.target.value) || 51820)} placeholder="51820" className="font-mono" error={checkPort(String(iface.listen_port || ''))} />
        <EntryRow title="MTU (0 = auto)" inputMode="numeric" value={iface.mtu || ''}
          onChange={(e) => set('mtu', Number(e.target.value) || 0)} placeholder="0" className="font-mono" error={iface.mtu ? checkMTU(String(iface.mtu)) : null} />
        <ComboRow title="Routing table">
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
          <EntryRow title="Custom table number" inputMode="numeric" value={iface.table ? Number(iface.table) : ''}
            onChange={(e) => set('table', e.target.value ? String(Number(e.target.value)) : '')}
            placeholder="Table #" className="font-mono" />
        )}
        <SwitchRow title="Allow inbound"
          subtitle="Open this interface's listen port in the firewall (routier input chain)"
          checked={iface.allow_inbound ?? false}
          onCheckedChange={(v) => set('allow_inbound', v)} />
      </PreferencesGroup>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label>Private key</Label>
          <div className="flex items-center gap-3">
            <Segmented
              value={keyMode}
              onChange={switchKeyMode}
              options={[{ value: 'inline', label: 'Inline' }, { value: 'file', label: 'File' }]}
            />
            {keyMode === 'inline' && (
              <Button variant="outline" size="sm" onClick={generateKey} disabled={generatingKey} className="h-7 gap-1.5 text-xs">
                <Shuffle className="h-3.5 w-3.5" />
                {generatingKey ? 'Generating…' : 'Generate'}
              </Button>
            )}
          </div>
        </div>

        {keyMode === 'inline' ? (
          <div className="space-y-1.5">
            <KeyField
              value={iface.private_key}
              onChange={(v) => set('private_key', v)}
              placeholder="Base64 private key, click Generate to create one"
            />
            {serverPubKey && (
              <div className="flex items-center gap-2 px-1 py-1 rounded bg-muted/40 text-xs">
                <span className="text-muted-foreground font-medium shrink-0">Public key:</span>
                <span className="font-mono text-foreground truncate flex-1">{serverPubKey}</span>
                <CopyButton text={serverPubKey} />
              </div>
            )}
          </div>
        ) : (
          <Input value={iface.private_key_file}
            onChange={(e) => set('private_key_file', e.target.value)}
            className="font-mono text-sm" placeholder="/etc/wireguard/private.key" />
        )}
      </div>

      <div className="space-y-1.5">
        <Label>Addresses</Label>
        <TagInput values={iface.addresses} onChange={(v) => set('addresses', v)} placeholder="10.0.0.1/24" mono validate={checkCIDR} />
      </div>

      <div className="border rounded-md px-3 py-2">
        <button type="button" onClick={() => setHooksOpen(!hooksOpen)}
          className="flex items-center gap-1 text-sm font-semibold text-foreground hover:text-primary w-full py-1">
          {hooksOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          Hooks
        </button>
        {hooksOpen && (
          <div className="mt-3 grid grid-cols-1 gap-4 sm:grid-cols-2">
            {(
              [['pre_up', 'PreUp'], ['post_up', 'PostUp'], ['pre_down', 'PreDown'], ['post_down', 'PostDown']] as [keyof WgIface, string][]
            ).map(([key, label]) => (
              <div key={key} className="space-y-1.5">
                <Label className="text-xs">{label}</Label>
                <TagInput values={iface[key] as string[]} onChange={(v) => set(key, v)}
                  placeholder="command or script" mono />
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Label className="text-sm font-semibold">Peers</Label>
            <Badge variant="secondary" className="text-xs">{peers.length}</Badge>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={addPeer} className="gap-1 h-7 text-xs">
            <Plus className="h-3 w-3" />Add peer
          </Button>
        </div>

        {peers.length === 0 && (
          <p className="text-sm text-muted-foreground italic">No peers configured.</p>
        )}

        {peers.map((peer, idx) => (
          <PeerRow
            key={idx}
            peer={peer}
            serverPubKey={serverPubKey}
            serverPort={iface.listen_port}
            serverAddresses={iface.addresses ?? []}
            ifaceName={name}
            onChange={(updated) => updatePeer(idx, updated)}
            onDelete={() => deletePeer(idx)}
          />
        ))}
      </div>

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add interface</Button>}
      </div>
    </>
  )
}

