import { api } from '@/lib/client'
import { Button, EntryRow, PageHeader, PreferencesGroup, Row, Segmented, useTheme, type ThemePreference } from 'cheval-ui'
import { APPEARANCE_OPTIONS } from '@/components/shell/MainMenu'
import { useState } from 'react'
import { toast } from 'sonner'

interface ViewOption<T extends string> {
  value: T
  label: string
}

const THEME_OPTIONS: ViewOption<ThemePreference>[] = APPEARANCE_OPTIONS.map(({ value, label }) => ({ value, label }))

export default function Settings() {
  const [current, setCurrent] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [saving, setSaving] = useState(false)
  const { preference, setPreference } = useTheme()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (newPassword !== confirm) {
      toast.error('New password and confirmation do not match')
      return
    }

    setSaving(true)
    try {
      await api.apiAuthPasswordPut({ TypesChangePasswordRequest: { current, new: newPassword } })
      toast.success('Password changed')
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
    <div className="mx-auto max-w-3xl space-y-6">
      <PageHeader title="Preferences" />

      <PreferencesGroup title="Interface">
        <Row title="Appearance">
          <Segmented value={preference} onChange={setPreference} options={THEME_OPTIONS} />
        </Row>
      </PreferencesGroup>

      <form onSubmit={handleSubmit} className="space-y-4">
        <PreferencesGroup
          title="Password"
          header={(
            <Button type="submit" disabled={saving || !current || !newPassword || newPassword !== confirm}>
              {saving ? 'Changing…' : 'Change Password'}
            </Button>
          )}
        >
          <EntryRow
            id="current-password"
            title="Current Password"
            type="password"
            autoComplete="current-password"
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
            required
          />
          <EntryRow
            id="new-password"
            title="New Password"
            type="password"
            autoComplete="new-password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            required
          />
          <EntryRow
            id="confirm-password"
            title="Confirm Password"
            type="password"
            autoComplete="new-password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            required
          />
        </PreferencesGroup>
      </form>

    </div>
  )
}
