import { PreferencesGroup, EntryRow, SwitchRow } from 'cheval-ui'
import { TagInput } from 'cheval-ui'
import { checkIP } from '@/lib/validate'
import { Conntrackd } from '@/components/ha/types'

function emptyConntrackd(): Conntrackd {
  return { interface: '', address: '', peer_ips: [] }
}

export function ConntrackdConfig({ cfg, onChange }: {
  cfg: Conntrackd | null
  onChange: (next: Conntrackd | null) => void
}) {
  const enabled = cfg !== null

  const set = <K extends keyof Conntrackd>(key: K, val: Conntrackd[K]) =>
    onChange({ ...(cfg ?? emptyConntrackd()), [key]: val })

  return (
    <div className="space-y-6">
      <p className="max-w-2xl text-sm text-muted-foreground">
        Configure this node&apos;s conntrackd daemon. To set up both sides of an HA pair in one step,
        derive it from a friend instead: that configures the friend&apos;s conntrackd and this node&apos;s together.
      </p>

      <PreferencesGroup>
        <SwitchRow
          title="Enable conntrackd"
          subtitle="Run the conntrackd state-sync daemon on this node"
          checked={enabled}
          onCheckedChange={(v) => onChange(v ? emptyConntrackd() : null)}
        />
      </PreferencesGroup>

      {enabled && cfg && (
        <>
          <PreferencesGroup title="Sync link">
            <EntryRow
              title="Sync interface"
              value={cfg.interface}
              onChange={(e) => set('interface', e.target.value)}
              placeholder="eth1"
              className="font-mono"
            />
            <EntryRow
              title="Local address"
              value={cfg.address}
              onChange={(e) => set('address', e.target.value)}
              placeholder="10.255.0.1"
              className="font-mono"
            />
            <EntryRow
              title="Port (0 = default 3780)"
              value={cfg.port || ''}
              onChange={(e) => set('port', Number(e.target.value) || 0)}
              placeholder="3780"
              className="font-mono"
            />
          </PreferencesGroup>

          <PreferencesGroup title="Peer addresses">
            <div className="px-4 py-3">
              <TagInput values={cfg.peer_ips ?? []} onChange={(v) => set('peer_ips', v)} placeholder="10.255.0.2" mono validate={checkIP} />
            </div>
          </PreferencesGroup>

          <PreferencesGroup>
            <SwitchRow
              title="Allow inbound"
              subtitle="Open the conntrackd sync port on the sync interface in the firewall"
              checked={!!cfg.allow_inbound}
              onCheckedChange={(v) => set('allow_inbound', v)}
            />
          </PreferencesGroup>
        </>
      )}
    </div>
  )
}
