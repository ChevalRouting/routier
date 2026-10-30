import { DhcpConfigData, DhcpHA, DhcpHAMode, DhcpHAPeer } from '@/components/dhcp-config/shared'
import { Button, ChoiceRow, ComboRow, EntryRow, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, SwitchRow } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'

type HaTabShape = {
  cfg: DhcpConfigData
  ifaceNames: string[]
  onChange: (next: DhcpConfigData) => void
}

type PeerGroupShape = {
  index: number
  peer: DhcpHAPeer
  roles: string[]
  onChange: (next: DhcpHAPeer) => void
  onRemove: () => void
}

const MODES = [
  { value: 'hot-standby', label: 'Hot Standby' },
  { value: 'load-balancing', label: 'Load Balancing' },
]

const ROLES: Record<DhcpHAMode, string[]> = {
  'hot-standby': ['primary', 'standby', 'backup'],
  'load-balancing': ['primary', 'secondary', 'backup'],
}

const EMPTY_HA: DhcpHA = {
  interface: '',
  peers: [
    { name: '', address: '', role: 'primary' },
    { name: '', address: '', role: 'standby' },
  ],
}

function portValue(value: string): number | undefined {
  return value ? Number(value) : undefined
}

function PeerGroup({ index, peer, roles, onChange, onRemove }: PeerGroupShape) {
  const remove = (
    <Button variant="ghost" size="sm" className="h-7 gap-1 text-xs text-muted-foreground hover:text-destructive" onClick={onRemove}>
      <Trash2 className="h-3.5 w-3.5" />Remove
    </Button>
  )

  return (
    <PreferencesGroup title={`Peer ${index + 1}`} header={remove}>
      <EntryRow title="Name" value={peer.name} onChange={(e) => onChange({ ...peer, name: e.target.value })} placeholder="rt1" className="font-mono" />
      <EntryRow title="Address" help="This peer's address on the HA interface" value={peer.address} onChange={(e) => onChange({ ...peer, address: e.target.value })} placeholder="10.0.0.1" className="font-mono" />
      <ComboRow title="Role">
        <Select value={peer.role} onValueChange={(role) => onChange({ ...peer, role })}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            {roles.map((role) => <SelectItem key={role} value={role}>{role}</SelectItem>)}
          </SelectContent>
        </Select>
      </ComboRow>
      <SwitchRow title="Automatic Failover" subtitle="Take over the peer's clients when it stops answering" checked={peer.auto_failover !== false} onCheckedChange={(v) => onChange({ ...peer, auto_failover: v ? undefined : false })} />
    </PreferencesGroup>
  )
}

export function HaTab({ cfg, ifaceNames, onChange }: HaTabShape) {
  const ha = cfg.ha ?? EMPTY_HA
  const mode = ha.mode ?? 'hot-standby'
  const roles = ROLES[mode]
  const write = (next: DhcpHA) => onChange({ ...cfg, ha: next })
  const set = <K extends keyof DhcpHA>(k: K, v: DhcpHA[K]) => write({ ...ha, [k]: v })
  const setPeers = (peers: DhcpHAPeer[]) => set('peers', peers)
  const extraIface = ha.interface && !ifaceNames.includes(ha.interface) ? [ha.interface] : []

  const handleMode = (value: string) => {
    const next = value as DhcpHAMode
    const partner = next === 'hot-standby' ? 'standby' : 'secondary'
    const peers = ha.peers.map((peer) => (ROLES[next].includes(peer.role) ? peer : { ...peer, role: partner }))
    write({ ...ha, mode: next === 'hot-standby' ? undefined : next, peers })
  }

  const addPeer = (
    <Button variant="ghost" size="sm" className="h-7 gap-1 text-xs" onClick={() => setPeers([...ha.peers, { name: '', address: '', role: 'backup' }])}>
      <Plus className="h-3.5 w-3.5" />Add Peer
    </Button>
  )

  const modeHelp = mode === 'hot-standby'
    ? 'The primary serves every client, the standby takes over on failure'
    : 'The primary and secondary split clients, each takes over the whole pool on failure'

  return (
    <div className="space-y-5">
      <PreferencesGroup description="Use the same settings on every router">
        <SwitchRow title="Pair Kea With Its Peers" checked={!!ha.enabled} onCheckedChange={(v) => set('enabled', v || undefined)} />
      </PreferencesGroup>

      {ha.enabled && (
        <>
          <PreferencesGroup title="Relationship">
            <ChoiceRow title="Mode" value={mode} choices={MODES} onChange={handleMode} help={modeHelp} />
            <ComboRow title="Interface" subtitle="The link the peers reach each other on">
              <Select value={ha.interface} onValueChange={(v) => set('interface', v)}>
                <SelectTrigger><SelectValue placeholder="Pick the HA link" /></SelectTrigger>
                <SelectContent>
                  {[...ifaceNames, ...extraIface].map((name) => <SelectItem key={name} value={name} className="font-mono">{name}</SelectItem>)}
                </SelectContent>
              </Select>
            </ComboRow>
            <EntryRow title="This Server" help="The peer name of this router, defaults to the hostname" value={ha.this_server ?? ''} onChange={(e) => set('this_server', e.target.value || undefined)} placeholder="hostname" className="font-mono" />
            <EntryRow title="DHCPv4 Port" type="number" value={ha.port ?? ''} onChange={(e) => set('port', portValue(e.target.value))} placeholder="8004" />
            <EntryRow title="DHCPv6 Port" type="number" value={ha.port6 ?? ''} onChange={(e) => set('port6', portValue(e.target.value))} placeholder="8006" />
            <SwitchRow title="Open the HA Ports" subtitle="Accept the peers on the interface in the firewall" checked={!!ha.allow_inbound} onCheckedChange={(v) => set('allow_inbound', v || undefined)} />
          </PreferencesGroup>

          <p className="px-1 text-xs text-muted-foreground">List every peer, this router included; backup peers only receive lease updates</p>

          {ha.peers.map((peer, i) => (
            <PeerGroup
              key={i}
              index={i}
              peer={peer}
              roles={roles}
              onChange={(next) => setPeers(ha.peers.map((p, j) => (j === i ? next : p)))}
              onRemove={() => setPeers(ha.peers.filter((_, j) => j !== i))}
            />
          ))}

          {addPeer}
        </>
      )}
    </div>
  )
}
