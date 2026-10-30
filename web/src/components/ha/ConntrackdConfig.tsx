import { Conntrackd } from '@/components/ha/types'
import { checkIP } from '@/lib/validate'
import { Choice, ChoiceRow, EntryRow, PreferencesGroup, SwitchRow, TagInput } from 'cheval-ui'

type ConntrackdConfigShape = {
  cfg: Conntrackd | null
  ifaceNames: string[]
  onChange: (next: Conntrackd | null) => void
}

function emptyConntrackd(): Conntrackd {
  return { interface: '', address: '', peer_ips: [] }
}

export function ConntrackdConfig({ cfg, ifaceNames, onChange }: ConntrackdConfigShape) {
  const enabled = cfg !== null

  const set = <K extends keyof Conntrackd>(key: K, val: Conntrackd[K]) =>
    onChange({ ...(cfg ?? emptyConntrackd()), [key]: val })

  const ifaceChoices: Choice[] = ifaceNames.map((n) => ({ value: n, label: n }))

  if (cfg?.interface && !ifaceNames.includes(cfg.interface))
    ifaceChoices.unshift({ value: cfg.interface, label: cfg.interface })

  return (
    <div className="space-y-6">

      <PreferencesGroup>
        <SwitchRow
          title="conntrackd"
          subtitle="Run the conntrackd state-sync daemon on this node"
          checked={enabled}
          onCheckedChange={(v) => onChange(v ? emptyConntrackd() : null)}
        />
      </PreferencesGroup>

      {enabled && cfg && (
        <>
          <PreferencesGroup title="Sync Link">
            <ChoiceRow
              title="Sync Interface"
              value={cfg.interface}
              choices={ifaceChoices}
              onChange={(v) => set('interface', v)}
              help="Interface carrying conntrackd state-sync traffic between HA peers"
            />
            <EntryRow
              title="Local Address"
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

          <PreferencesGroup title="Peer Addresses">
            <div className="px-4 py-3">
              <TagInput values={cfg.peer_ips ?? []} onChange={(v) => set('peer_ips', v)} placeholder="10.255.0.2" mono validate={checkIP} />
            </div>
          </PreferencesGroup>

          <PreferencesGroup>
            <SwitchRow
              title="Allow Inbound"
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
