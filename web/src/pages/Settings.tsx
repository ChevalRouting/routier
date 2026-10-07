import type { SimpleProjection } from '@/components/simple/types'
import { addStagingListener, api, configLayerRequest, getStagingState } from '@/lib/client'
import {
  getConfigLayer,
  setConfigLayer,
  type ConfigLayer,
} from '@/lib/configLayer'
import { AlertDialog, Button, EntryRow, PageHeader, PreferencesGroup, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

interface LayerSwitchConfirmation {
  next: ConfigLayer
  ownsStaging: boolean
  losses: NonNullable<SimpleProjection['losses']>
}

export default function Settings() {
  const [current, setCurrent] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [saving, setSaving] = useState(false)
  const [layer, setLayer] = useState<ConfigLayer>(getConfigLayer())
  const [staging, setStaging] = useState(getStagingState())
  const [switchingLayer, setSwitchingLayer] = useState(false)
  const [layerSwitchConfirmation, setLayerSwitchConfirmation] = useState<LayerSwitchConfirmation | null>(null)

  useEffect(() => addStagingListener(setStaging), [])

  const switchLayer = async (next: ConfigLayer) => {
    await api.apiConfigStagingDelete().catch(() => undefined)
    setConfigLayer(next)
    setLayer(next)
    await api.apiConfigStagingDelete().catch(() => undefined)
    toast.success(`Switched to ${next} configuration`)
  }

  const handleLayerChange = async (next: ConfigLayer) => {
    if (next === layer) return

    setSwitchingLayer(true)
    try {
      const pending = await api.apiApplyPendingGet()
      if (pending.pending) {
        toast.error('Confirm or roll back the current configuration before switching views')
        return
      }

      const ownsStaging = staging.pending && (!staging.layer || staging.layer === layer)
      let losses: NonNullable<SimpleProjection['losses']> = []
      if (next === 'simple') {
        const projection = await configLayerRequest<SimpleProjection>('/api/config', undefined, 'simple')
        losses = projection.losses ?? []
      }

      if (ownsStaging || losses.length > 0) {
        setLayerSwitchConfirmation({ next, ownsStaging, losses })
        return
      }

      await switchLayer(next)
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to switch configuration views')
    } finally {
      setSwitchingLayer(false)
    }
  }

  const confirmLayerChange = async () => {
    if (!layerSwitchConfirmation) return
    const { next } = layerSwitchConfirmation
    setSwitchingLayer(true)
    try {
      await switchLayer(next)
      setLayerSwitchConfirmation(null)
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to switch configuration views')
    } finally {
      setSwitchingLayer(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (newPassword !== confirm) {
      toast.error('New password and confirmation do not match')
      return
    }

    setSaving(true)
    try {
      await api.apiAuthPasswordPut({ TypesChangePasswordRequest: { current, new: newPassword } })
      toast.success('Password updated successfully')
      setCurrent('')
      setNewPassword('')
      setConfirm('')
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to update password')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Settings" description="Account settings" />

      <form onSubmit={handleSubmit} className="max-w-xl space-y-4">
        <PreferencesGroup title="Configuration view" description="Choose how much networking detail Routier shows.">
          <div className="flex items-center justify-between gap-4 px-4 py-3">
            <div>
              <p className="text-sm font-medium">Interface</p>
              <p className="text-xs text-muted-foreground">
                Simple groups related settings. Advanced exposes every configuration section.
              </p>
            </div>
            <Select
              value={layer}
              disabled={switchingLayer}
              onValueChange={(value) => handleLayerChange(value as ConfigLayer)}
            >
              <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="simple">Simple</SelectItem>
                <SelectItem value="advanced">Advanced</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </PreferencesGroup>

        <PreferencesGroup title="Change password">
          <EntryRow
            id="current-password"
            title="Current password"
            type="password"
            autoComplete="current-password"
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
            required
          />
          <EntryRow
            id="new-password"
            title="New password"
            type="password"
            autoComplete="new-password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            required
          />
          <EntryRow
            id="confirm-password"
            title="Confirm password"
            type="password"
            autoComplete="new-password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            required
          />
        </PreferencesGroup>
        <Button type="submit" disabled={saving}>
          {saving ? 'Updating…' : 'Update password'}
        </Button>
      </form>

      <AlertDialog
        open={layerSwitchConfirmation !== null}
        onCancel={() => setLayerSwitchConfirmation(null)}
        onConfirm={confirmLayerChange}
        title={`Switch to ${layerSwitchConfirmation?.next ?? ''} configuration?`}
        description={layerSwitchConfirmation && (
          <div className="space-y-3 text-left">
            {layerSwitchConfirmation.ownsStaging && (
              <p>Your unapplied changes will be discarded.</p>
            )}
            {layerSwitchConfirmation.losses.length > 0 && (
              <p>
                Simple mode is a lossy view of the router configuration. If you save from Simple mode,
                advanced settings that this view cannot represent may be replaced.
              </p>
            )}
          </div>
        )}
        confirmLabel="Switch view"
        cancelLabel="Keep current view"
        destructive={layerSwitchConfirmation?.ownsStaging || !!layerSwitchConfirmation?.losses.length}
        busy={switchingLayer}
      />
    </div>
  )
}
