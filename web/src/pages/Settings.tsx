import { useState } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import { Button } from '@/components/ui/button'
import { PageHeader } from '@/components/PageHeader'
import { PreferencesGroup, EntryRow } from '@/components/Preferences'

export default function Settings() {
  const [current, setCurrent] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [saving, setSaving] = useState(false)

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
    </div>
  )
}
