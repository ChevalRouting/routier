import type { LiveDisk, TypesInstallStatus as InstallStatus } from '@/api'
import Logo from '@/components/Logo'
import { DiskList } from '@/components/install/DiskList'
import { InstallProgress } from '@/components/install/InstallProgress'
import { formatSize, skipLive } from '@/components/install/shared'
import { StepNav } from '@/components/onboarding/shared'
import { AutocompleteInput, type AutocompleteOption } from '@/components/ui/AutocompleteInput'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { AlertDialog, Button, EntryRow, LoginScreen, PreferencesGroup, Row, Spinner } from 'cheval-ui'
import { HardDriveDownload, MonitorPlay, Terminal } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'

type Step = 'choose' | 'disk' | 'system' | 'progress'

const hostnamePattern = /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/

export default function Install() {
  const navigate = useNavigate()
  const { data: initial, isLoading } = useFetch<InstallStatus>(() => api.apiInstallStatusGet())
  const { data: disks } = useFetch<LiveDisk[]>(() => api.apiInstallDisksGet())
  const { data: zones } = useFetch<string[]>(() => api.apiInstallTimezonesGet())

  const [step, setStep] = useState<Step | null>(null)
  const [status, setStatus] = useState<InstallStatus | null>(null)
  const [disk, setDisk] = useState('')
  const [hostname, setHostname] = useState('routier')
  const [timezone, setTimezone] = useState('UTC')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)
  const [rebooting, setRebooting] = useState(false)

  useEffect(() => {
    if (!initial || step !== null) return
    setStatus(initial)
    if (initial.state === 'running' || initial.state === 'done') setStep('progress')
    else setStep(initial.installer_only ? 'disk' : 'choose')
  }, [initial, step])

  useEffect(() => {
    if (!disks || disk) return
    const candidate = disks.find((d) => !d.boot_media && !d.removable) ?? disks.find((d) => !d.boot_media)
    if (candidate) setDisk(candidate.path)
  }, [disks, disk])

  const poll = useCallback(async () => {
    try {
      setStatus(await api.apiInstallStatusGet())
    } catch {
      return
    }
  }, [])

  useEffect(() => {
    if (step !== 'progress' || status?.state !== 'running') return
    const timer = setInterval(poll, 2000)
    return () => clearInterval(timer)
  }, [step, status?.state, poll])

  const zoneOptions = useMemo<AutocompleteOption[]>(
    () => (zones ?? []).map((zone) => ({ value: zone, label: zone })),
    [zones],
  )

  const selectedDisk = disks?.find((d) => d.path === disk)
  const hostnameError = hostname && !hostnamePattern.test(hostname) ? 'Letters, digits and hyphens only' : null
  const timezoneError = zones && timezone && !zones.some((zone) => zone.toLowerCase().includes(timezone.toLowerCase())) ? 'Unknown timezone' : null
  const passwordError = password && password.length < 8 ? 'At least 8 characters' : null
  const confirmError = confirmPassword && confirmPassword !== password ? 'Does not match' : null
  const systemValid = !!hostname && !hostnameError && !!zones?.includes(timezone)
    && password.length >= 8 && password === confirmPassword

  const continueLive = () => {
    skipLive()
    navigate('/', { replace: true })
  }

  const start = async () => {
    setBusy(true)
    try {
      const next = await api.apiInstallPost({ TypesInstallRequest: { disk, hostname, timezone, password } })
      setStatus(next)
      setConfirming(false)
      setStep('progress')
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const reboot = async () => {
    setRebooting(true)
    try {
      await api.apiInstallRebootPost()
    } catch (e) {
      setRebooting(false)
      toast.error((e as Error).message)
    }
  }

  const retry = () => setStep('disk')

  if (initial && !initial.live) return <Navigate to="/" replace />
  if (isLoading || step === null) return <Spinner />

  return (
    <LoginScreen title="Install Routier" logo={Logo}>
      <div className="space-y-6">
        {step === 'choose' && (
          <>
            <PreferencesGroup title="This System Runs From the Installation Media" description="Changes are lost on restart until Routier is installed to a disk">
              <Row
                prefix={<HardDriveDownload className="h-5 w-5" />}
                title="Install to Disk"
                subtitle="Erase a disk and install Routier on it"
                onClick={() => setStep('disk')}
              />
              <Row
                prefix={<MonitorPlay className="h-5 w-5" />}
                title="Continue Without Installing"
                subtitle="Configure and try Routier from memory"
                onClick={continueLive}
              />
            </PreferencesGroup>
            <ConsoleHint />
          </>
        )}

        {step === 'disk' && (
          <>
            {disks ? <DiskList disks={disks} selected={disk} onSelect={setDisk} /> : <Spinner />}
            {status?.installer_only ? (
              <>
                <ConsoleHint />
                <div className="flex justify-end">
                  <Button onClick={() => setStep('system')} disabled={!selectedDisk}>Next</Button>
                </div>
              </>
            ) : (
              <StepNav onBack={() => setStep('choose')} onNext={() => setStep('system')} nextDisabled={!selectedDisk} />
            )}
          </>
        )}

        {step === 'system' && (
          <>
            <PreferencesGroup title="System">
              <EntryRow title="Hostname" value={hostname} onChange={(e) => setHostname(e.target.value)} className="font-mono" error={hostnameError} />
              <div className="space-y-1.5 px-4 py-2.5">
                <p className="text-sm font-medium">Timezone</p>
                <AutocompleteInput
                  value={timezone}
                  options={zoneOptions}
                  onChange={setTimezone}
                  placeholder="UTC"
                  emptyMessage="No matching timezone"
                  inputProps={{ 'aria-label': 'Timezone' }}
                />
                {timezoneError && <p className="text-xs text-destructive">{timezoneError}</p>}
              </div>
            </PreferencesGroup>
            <PreferencesGroup title="Admin Password" description="Used to sign in as routier on the installed system">
              <EntryRow title="Password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} error={passwordError} />
              <EntryRow title="Confirm Password" type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} error={confirmError} />
            </PreferencesGroup>
            <div className="flex justify-between">
              <Button variant="outline" onClick={() => setStep('disk')}>Back</Button>
              <Button variant="destructive" onClick={() => setConfirming(true)} disabled={!systemValid}>Erase and Install</Button>
            </div>
          </>
        )}

        {step === 'progress' && status && (
          <InstallProgress status={status} rebooting={rebooting} onReboot={reboot} onRetry={retry} />
        )}
      </div>

      <AlertDialog
        open={confirming}
        onCancel={() => setConfirming(false)}
        onConfirm={start}
        title={`Erase ${disk}?`}
        description={selectedDisk ? `All data on ${disk} (${formatSize(selectedDisk.size)}${selectedDisk.model ? `, ${selectedDisk.model}` : ''}) is permanently erased.` : undefined}
        confirmLabel="Erase and Install"
        destructive
        busy={busy}
      />
    </LoginScreen>
  )
}

function ConsoleHint() {
  return (
    <p className="flex items-center justify-center gap-2 px-1 text-center text-xs text-muted-foreground">
      <Terminal className="h-3.5 w-3.5 shrink-0" />
      <span>Prefer the console? Run <span className="font-mono">setup-routier</span> there.</span>
    </p>
  )
}
