import { useEffect, useState } from 'react'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { useDataRefresh } from '@/lib/dataVersion'
import { usePageSave } from '@/lib/usePageSave'
import SaveButton from '@/components/SaveButton'
import { PageHeader } from '@/components/PageHeader'
import { Spinner } from '@/components/Spinner'
import { PreferencesGroup, EntryRow, SwitchRow } from '@/components/Preferences'
import TagInput from '@/components/TagInput'
import { checkIP } from '@/lib/validate'

interface Conntrackd {
  interface: string
  address: string
  peer_ips: string[]
  port?: number
  allow_inbound?: boolean
}

function emptyConntrackd(): Conntrackd {
  return { interface: '', address: '', peer_ips: [] }
}

export default function ConntrackdPage() {
  const { data, isLoading } = useFetch<Conntrackd | null>(() => api.apiConfigSectionGet({ section: 'conntrackd' }) as Promise<Conntrackd | null>)
  const [cfg, setCfg] = useState<Conntrackd | null>(null)
  const [initialized, setInitialized] = useState(false)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('conntrackd')

  useDataRefresh(() => { setInitialized(false); reset() })

  useEffect(() => {
    if (!isLoading && !initialized) {
      setCfg(data ?? null)
      setInitialized(true)
    }
  }, [isLoading, initialized, data])

  const enabled = cfg !== null

  const update = (next: Conntrackd | null) => {
    setCfg(next)
    markDirty()
  }

  const set = <K extends keyof Conntrackd>(key: K, val: Conntrackd[K]) =>
    update({ ...(cfg ?? emptyConntrackd()), [key]: val })

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      <PageHeader
        title="Conntrackd"
        description="Synchronize connection tracking state with an HA peer"
        action={<SaveButton isDirty={isDirty} saving={saving} onClick={() => save(cfg)} onCancel={() => { setCfg(data ?? null); reset() }} />}
      />

      <p className="max-w-2xl text-sm text-muted-foreground">
        Configure this node&apos;s conntrackd daemon. To set up both sides of an HA pair in one step,
        derive it from a friend instead &mdash; that configures the friend&apos;s conntrackd and this node&apos;s together.
      </p>

      <PreferencesGroup>
        <SwitchRow
          title="Enable conntrackd"
          subtitle="Run the conntrackd state-sync daemon on this node"
          checked={enabled}
          onCheckedChange={(v) => update(v ? emptyConntrackd() : null)}
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
