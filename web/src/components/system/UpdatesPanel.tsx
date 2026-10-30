import { apiRequest } from '@/lib/client'
import { Badge, Button, PreferencesGroup, Row } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { UpgradeModal } from './UpgradeModal'

interface SystemVersion {
  version: string
  kernel: string
  flavor: string
  reboot_pending: boolean
  live: boolean
}

interface UpdatePackage {
  name: string
  old: string
  new: string
}

interface SystemUpdates {
  version: string
  packages: UpdatePackage[]
  reboot_required: boolean
  self_update: boolean
  live: boolean
  checked_at: string
}

export default function UpdatesPanel() {
  const [version, setVersion] = useState<SystemVersion | null>(null)
  const [updates, setUpdates] = useState<SystemUpdates | null>(null)
  const [checking, setChecking] = useState(false)
  const [upgradeOpen, setUpgradeOpen] = useState(false)
  const [rebooting, setRebooting] = useState(false)

  const loadVersion = () =>
    apiRequest<SystemVersion>('/api/system/version').then(setVersion).catch(() => {})

  useEffect(() => { loadVersion() }, [])

  const check = async () => {
    setChecking(true)
    try {
      const result = await apiRequest<SystemUpdates>('/api/system/updates?refresh=1')
      result.packages = result.packages ?? []
      setUpdates(result)
      if (result.packages.length === 0) toast.success('System is up to date')
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setChecking(false)
    }
  }

  const reboot = async () => {
    setRebooting(true)
    try {
      await apiRequest('/api/system/reboot', { method: 'POST' })
      toast.message('Rebooting, the appliance will be unreachable for a moment')
    } catch (e) {
      toast.error((e as Error).message)
      setRebooting(false)
    }
  }

  const live = version?.live ?? false
  const pkgs = updates?.packages ?? []

  return (
    <PreferencesGroup>
      <Row title="Current Version" subtitle={version ? `kernel ${version.kernel}` : undefined}>
        <span className="font-mono text-sm">{version?.version || '…'}</span>
      </Row>

      {version?.reboot_pending && (
        <Row title="Reboot Pending" subtitle="A new kernel is installed but not running">
          <Button size="sm" variant="destructive" onClick={reboot} disabled={rebooting}>
            {rebooting ? 'Rebooting…' : 'Reboot Now'}
          </Button>
        </Row>
      )}

      {live && (
        <div className="px-4 py-3 text-sm text-muted-foreground">
          Running from the live image. Updates require a disk install (run setup-routier).
        </div>
      )}

      {updates && !live && (
        <div className="px-4 py-3 space-y-2">
          {pkgs.length === 0 ? (
            <p className="text-sm text-muted-foreground">System is up to date</p>
          ) : (
            <>
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm font-medium">
                  {pkgs.length} update{pkgs.length === 1 ? '' : 's'} available
                </span>
                {updates.reboot_required && <Badge variant="warning">kernel, reboot needed</Badge>}
                {updates.self_update && <Badge variant="neutral">web UI will restart</Badge>}
              </div>
              <ul className="max-h-48 overflow-auto rounded-md bg-muted/40 p-2 text-xs font-mono space-y-0.5">
                {pkgs.map((p) => (
                  <li key={p.name}>{p.name}  {p.old} → {p.new}</li>
                ))}
              </ul>
            </>
          )}
        </div>
      )}

      <div className="flex items-center gap-2 px-4 py-3">
        <Button variant="outline" onClick={check} disabled={checking || live}>
          {checking ? 'Checking…' : 'Check for Updates'}
        </Button>
        <Button onClick={() => setUpgradeOpen(true)} disabled={live || pkgs.length === 0}>
          Update…
        </Button>
      </div>

      <UpgradeModal
        open={upgradeOpen}
        selfUpdate={updates?.self_update ?? false}
        onClose={() => { setUpgradeOpen(false); loadVersion() }}
      />
    </PreferencesGroup>
  )
}
