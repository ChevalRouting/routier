import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import type { TypesSystemNic as SystemNic } from '@/api'
import { useFetch } from '@/lib/useFetch'
import { Button } from '@/components/ui/button'
import { PreferencesGroup, EntryRow, Row } from '@/components/Preferences'
import { checkCIDR, checkIP } from '@/lib/validate'
import { cn } from '@/lib/utils'
import Logo from '@/components/Logo'
import { WatchdogConfirmBar } from '@/components/WatchdogConfirmBar'

type Step = 'password' | 'hostname' | 'interfaces' | 'review' | 'confirm'
const STEPS: { key: Step; label: string }[] = [
  { key: 'password',   label: 'Password' },
  { key: 'hostname',   label: 'Hostname' },
  { key: 'interfaces', label: 'Interfaces' },
  { key: 'review',     label: 'Review' },
  { key: 'confirm',    label: 'Confirm' },
]

type IfaceMode = 'dhcp' | 'static'
interface IfaceSetting { mode: IfaceMode; cidr: string }

export default function Onboarding() {
  const navigate = useNavigate()
  const { data: nics } = useFetch<SystemNic[]>(() => api.apiSystemNicsGet())

  const [step, setStepRaw]      = useState<Step>('password')
  const [busy, setBusy]          = useState(false)
  const [stateLoaded, setStateLoaded] = useState(false)

  const [curPw, setCurPw]         = useState('')
  const [newPw, setNewPw]         = useState('')
  const [confirmPw, setConfirmPw] = useState('')

  const [hostname, setHostname]           = useState('routier')
  const [ifaceSettings, setIfaceSettings] = useState<Record<string, IfaceSetting>>({})
  const [gateway, setGateway]             = useState('')

  const nicList: SystemNic[] = nics ?? []
  const stepIdx = STEPS.findIndex((s) => s.key === step)

  useEffect(() => {
    api.apiSetupOnboardingGet().then((raw) => {
      const saved = raw as Record<string, unknown>
      if (saved.step)          setStepRaw(saved.step as Step)
      if (saved.hostname)      setHostname(saved.hostname as string)
      if (saved.ifaceSettings) setIfaceSettings(saved.ifaceSettings as Record<string, IfaceSetting>)
      if (saved.gateway)       setGateway(saved.gateway as string)
      setStateLoaded(true)
    }).catch(() => setStateLoaded(true))
  }, [])

  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(() => {
    if (!stateLoaded) return
    if (saveTimer.current) clearTimeout(saveTimer.current)
    saveTimer.current = setTimeout(() => {
      api.apiSetupOnboardingPut({ body: { step, hostname, ifaceSettings, gateway } }).catch(() => {})
    }, 400)
    return () => { if (saveTimer.current) clearTimeout(saveTimer.current) }
  }, [step, hostname, ifaceSettings, gateway, stateLoaded])

  const setStep = (s: Step) => setStepRaw(s)

  useEffect(() => {
    if (!nics) return
    setIfaceSettings(prev => {
      const next = { ...prev }
      for (const nic of nics) {
        if (next[nic.name]) continue
        const firstAddr = nic.addrs?.[0]
        next[nic.name] = firstAddr
          ? { mode: 'static', cidr: firstAddr }
          : { mode: 'dhcp', cidr: '' }
      }
      return next
    })
  }, [nics])

  const setSetting = (name: string, patch: Partial<IfaceSetting>) =>
    setIfaceSettings(prev => ({ ...prev, [name]: { ...prev[name], ...patch } }))

  const savePassword = async () => {
    if (newPw.length < 8) return toast.error('Password must be at least 8 characters')
    if (newPw !== confirmPw) return toast.error('Passwords do not match')
    setBusy(true)
    try {
      await api.apiAuthPasswordPut({ TypesChangePasswordRequest: { current: curPw, new: newPw } })
      setStep('hostname')
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const apply = async () => {
    setBusy(true)
    try {
      const ifaces: Record<string, unknown> = {}
      for (const nic of nicList) {
        const s = ifaceSettings[nic.name] ?? { mode: 'dhcp', cidr: '' }
        ifaces[nic.name] = {
          select: nic.name,
          addresses: s.mode === 'dhcp' ? ['dhcp'] : [s.cidr],
        }
      }
      await api.apiConfigSectionPut({ section: 'hostname', body: hostname as unknown as object })
      await api.apiConfigSectionPut({ section: 'interfaces', body: ifaces })
      if (gateway) {
        await api.apiConfigSectionPut({
          section: 'routing',
          body: { static: [{ destination: '0.0.0.0/0', via: gateway }] },
        })
      }
      let applied = false
      try {
        const result = await api.apiConfigApplyPost()
        if (result.warning) toast.warning(result.warning)
        applied = true
      } catch {
        try {
          const p = await api.apiApplyPendingGet()
          if (p.pending) applied = true
        } catch { applied = true }
      }
      if (applied) setStep('confirm')
      else toast.error('Apply failed, check your configuration')
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const onKept = async () => {
    await api.apiSetupOnboardingPut({ body: {} }).catch(() => {})
    try { await api.apiSetupCompletePost() } catch {}
    toast.success('Routier is set up')
    navigate('/')
  }

  const onRolledBack = () => {
    toast.error('Not confirmed in time, configuration rolled back. Adjust and try again.')
    setStep('review')
  }

  const ifacesValid = nicList.every(nic => {
    const s = ifaceSettings[nic.name]
    if (!s || s.mode === 'dhcp') return true
    return !checkCIDR(s.cidr)
  })
  const gatewayValid = !gateway || !checkIP(gateway)

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/30 p-4">
      <div className="w-full max-w-lg space-y-6">

        <div className="flex flex-col items-center gap-3">
          <Logo className="h-14 w-14" />
          <div className="text-center">
            <h1 className="text-2xl font-bold tracking-tight">Welcome to Routier</h1>
            <p className="text-sm text-muted-foreground">Let's get your router set up.</p>
          </div>
        </div>

        <div className="flex items-center justify-center gap-2">
          {STEPS.map((s, i) => (
            <div key={s.key} className="flex items-center gap-2">
              <span className={cn(
                'flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold',
                i < stepIdx
                  ? 'bg-primary text-primary-foreground'
                  : i === stepIdx
                    ? 'bg-primary/15 text-primary ring-1 ring-primary'
                    : 'bg-muted text-muted-foreground',
              )}>{i + 1}</span>
              <span className={cn('text-xs', i === stepIdx ? 'font-medium text-foreground' : 'text-muted-foreground')}>
                {s.label}
              </span>
              {i < STEPS.length - 1 && <span className="mx-1 h-px w-4 bg-border" />}
            </div>
          ))}
        </div>

        {step === 'password' && (
          <>
            <PreferencesGroup title="Set a new admin password" description="Replace the default credentials before going further.">
              <EntryRow title="Current password" type="password" value={curPw} onChange={(e) => setCurPw(e.target.value)} placeholder="routier" autoFocus />
              <EntryRow title="New password" type="password" value={newPw} onChange={(e) => setNewPw(e.target.value)}
                error={newPw && newPw.length < 8 ? 'At least 8 characters' : null} />
              <EntryRow title="Confirm new password" type="password" value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)}
                error={confirmPw && confirmPw !== newPw ? 'Does not match' : null} />
            </PreferencesGroup>
            <div className="flex justify-end">
              <Button onClick={savePassword} disabled={busy || !curPw || !newPw || newPw !== confirmPw}>
                Set password
              </Button>
            </div>
          </>
        )}

        {step === 'hostname' && (
          <>
            <PreferencesGroup title="Hostname">
              <EntryRow title="Hostname" value={hostname} onChange={(e) => setHostname(e.target.value)} placeholder="routier" autoFocus className="font-mono" />
            </PreferencesGroup>
            <StepNav onBack={() => setStep('password')} onNext={() => setStep('interfaces')} nextDisabled={!hostname.trim()} />
          </>
        )}

        {step === 'interfaces' && (
          <>
            <PreferencesGroup title="Network interfaces" description="Configure each interface. Static mode requires a CIDR address (e.g. 192.168.1.1/24).">
              {nicList.length === 0 && (
                <p className="px-4 py-3 text-sm text-muted-foreground">No physical interfaces detected.</p>
              )}
              {nicList.map(nic => {
                const s = ifaceSettings[nic.name] ?? { mode: 'dhcp' as IfaceMode, cidr: '' }
                const cidrErr = s.mode === 'static' ? checkCIDR(s.cidr) : null
                const isOk = s.mode === 'dhcp' ? true : (!cidrErr && !!s.cidr)
                return (
                  <div key={nic.name} className="px-4 py-3 space-y-3 border-b last:border-b-0">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-medium text-sm">{nic.name}</span>
                        {isOk && (
                          <span className="inline-flex items-center rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900/30 dark:text-green-400">
                            OK
                          </span>
                        )}
                        {nic.operstate && (
                          <span className={cn('text-xs', nic.operstate === 'up' ? 'text-green-600' : 'text-muted-foreground')}>
                            {nic.operstate}
                          </span>
                        )}
                      </div>
                      <ModeToggle value={s.mode} onChange={(m) => setSetting(nic.name, { mode: m })} />
                    </div>

                    {s.mode === 'static' ? (
                      <div>
                        <EntryRow
                          title="Address (CIDR)"
                          value={s.cidr}
                          onChange={(e) => setSetting(nic.name, { cidr: e.target.value })}
                          placeholder="192.168.1.1/24"
                          className="font-mono"
                          error={cidrErr}
                        />
                      </div>
                    ) : (
                      <p className="text-xs text-muted-foreground">
                        Address will be assigned automatically via DHCP.
                        {nic.addrs && nic.addrs.length > 0 && (
                          <> Current: <span className="font-mono">{nic.addrs.join('  ')}</span></>
                        )}
                      </p>
                    )}
                  </div>
                )
              })}
            </PreferencesGroup>

            <PreferencesGroup title="Default gateway" description="Required when any interface uses a static address.">
              <EntryRow
                title="Gateway"
                value={gateway}
                onChange={(e) => setGateway(e.target.value)}
                placeholder="192.168.1.254  (leave blank for DHCP)"
                className="font-mono"
                error={gateway ? checkIP(gateway) : null}
              />
            </PreferencesGroup>

            <StepNav
              onBack={() => setStep('hostname')}
              onNext={() => setStep('review')}
              nextDisabled={nicList.length === 0 || !ifacesValid || !gatewayValid}
            />
          </>
        )}

        {step === 'review' && (
          <>
            <PreferencesGroup title="Review">
              <Row title="Hostname">{hostname}</Row>
              {nicList.map(nic => {
                const s = ifaceSettings[nic.name] ?? { mode: 'dhcp' as IfaceMode, cidr: '' }
                return (
                  <Row key={nic.name} title={nic.name}>
                    <span className="font-mono">{s.mode === 'dhcp' ? 'DHCP' : s.cidr}</span>
                  </Row>
                )
              })}
              {gateway && <Row title="Gateway"><span className="font-mono">{gateway}</span></Row>}
            </PreferencesGroup>
            <p className="px-1 text-xs text-muted-foreground">
              Applying writes the config and brings the interfaces up. You'll have 60 s to confirm connectivity before the changes roll back.
            </p>
            <StepNav onBack={() => setStep('interfaces')} onNext={apply} nextLabel={busy ? 'Applying…' : 'Apply'} nextDisabled={busy} />
          </>
        )}

        {step === 'confirm' && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground text-center">
              Configuration applied. Confirm connectivity to keep the changes, the timer below shows your remaining window.
            </p>
            <WatchdogConfirmBar onKept={onKept} onRolledBack={onRolledBack} />
          </div>
        )}
      </div>
    </div>
  )
}

function ModeToggle({ value, onChange }: { value: IfaceMode; onChange: (m: IfaceMode) => void }) {
  return (
    <div className="flex rounded-md border text-xs overflow-hidden">
      {(['dhcp', 'static'] as IfaceMode[]).map(m => (
        <button
          key={m}
          onClick={() => onChange(m)}
          className={cn(
            'px-3 py-1 capitalize transition-colors',
            value === m
              ? 'bg-primary text-primary-foreground font-medium'
              : 'bg-background text-muted-foreground hover:text-foreground',
          )}
        >
          {m}
        </button>
      ))}
    </div>
  )
}

function StepNav({ onBack, onNext, nextLabel = 'Next', nextDisabled }: {
  onBack: () => void
  onNext: () => void
  nextLabel?: string
  nextDisabled?: boolean
}) {
  return (
    <div className="flex justify-between">
      <Button variant="outline" onClick={onBack}>Back</Button>
      <Button onClick={onNext} disabled={nextDisabled}>{nextLabel}</Button>
    </div>
  )
}
