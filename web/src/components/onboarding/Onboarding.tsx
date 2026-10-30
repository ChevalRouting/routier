import { SetupUplinkModeEnum, type SetupNetwork, type TypesSetupHost as SetupHost, type TypesSystemNic as SystemNic } from '@/api'
import Logo from '@/components/Logo'
import { WatchdogConfirmBar } from '@/components/WatchdogConfirmBar'
import { AddrMode, defaultSetting, IfaceRole, IfaceSetting, selectorFor, Step, StepNav, STEPS } from '@/components/onboarding/shared'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { checkCIDR, checkIP } from '@/lib/validate'
import { Badge, Button, cn, EntryRow, LoginScreen, PreferencesGroup, Row, Segmented, TagInput, StateChip } from 'cheval-ui'
import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'

export default function Onboarding() {
  const navigate = useNavigate()
  const { data: nics } = useFetch<SystemNic[]>(() => api.apiSystemNicsGet())
  const { data: host, isLoading: hostLoading } = useFetch<SetupHost>(() => api.apiSetupHostGet())

  const [step, setStepRaw]      = useState<Step>('password')
  const [busy, setBusy]          = useState(false)
  const [stateLoaded, setStateLoaded] = useState(false)

  const [curPw, setCurPw]         = useState('')
  const [newPw, setNewPw]         = useState('')
  const [confirmPw, setConfirmPw] = useState('')

  const [hostname, setHostname]           = useState('routier')
  const [ifaceSettings, setIfaceSettings] = useState<Record<string, IfaceSetting>>({})

  const nicList: SystemNic[] = nics ?? []
  const stepIdx = STEPS.findIndex((s) => s.key === step)

  useEffect(() => {
    api.apiSetupOnboardingGet().then((raw) => {
      const saved = raw as Record<string, unknown>
      if (saved.step)          setStepRaw(saved.step as Step)
      if (saved.hostname)      setHostname(saved.hostname as string)
      if (saved.ifaceSettings) setIfaceSettings(saved.ifaceSettings as Record<string, IfaceSetting>)
      setStateLoaded(true)
    }).catch(() => setStateLoaded(true))
  }, [])

  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(() => {
    if (!stateLoaded) return
    if (saveTimer.current) clearTimeout(saveTimer.current)
    saveTimer.current = setTimeout(() => {
      api.apiSetupOnboardingPut({ body: { step, hostname, ifaceSettings } }).catch(() => {})
    }, 400)
    return () => { if (saveTimer.current) clearTimeout(saveTimer.current) }
  }, [step, hostname, ifaceSettings, stateLoaded])

  const setStep = (s: Step) => setStepRaw(s)

  useEffect(() => {
    if (!nics || !stateLoaded || hostLoading) return
    setIfaceSettings(prev => {
      const next = { ...prev }
      for (const nic of nics) {
        if (next[nic.name]?.role) continue
        next[nic.name] = defaultSetting(nic, host ?? null)
      }
      return next
    })
  }, [nics, host, hostLoading, stateLoaded])

  const setSetting = (name: string, patch: Partial<IfaceSetting>) =>
    setIfaceSettings(prev => ({ ...prev, [name]: { ...prev[name], ...patch } }))

  const setRole = (name: string, role: IfaceRole) => {
    setIfaceSettings((previous) => {
      const next = { ...previous }
      if (role === 'internet') {
        for (const nicName of Object.keys(next)) {
          if (nicName !== name && next[nicName]?.role === 'internet') {
            next[nicName] = { ...next[nicName], role: 'none' }
          }
        }
      }
      next[name] = { ...next[name], role }
      return next
    })
  }

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
      const uplinkNIC = nicList.find((nic) => ifaceSettings[nic.name]?.role === 'internet')
      const uplink = uplinkNIC ? ifaceSettings[uplinkNIC.name] : undefined
      const network: SetupNetwork = {
        hostname,
        uplink: uplinkNIC ? (uplink?.mode === 'static'
          ? { select: selectorFor(uplinkNIC), mode: SetupUplinkModeEnum.Static, address: uplink.cidr, gateway: uplink.gateway, dns: uplink.dns }
          : { select: selectorFor(uplinkNIC), mode: SetupUplinkModeEnum.Dhcp }) : undefined,
        lans: nicList.flatMap((nic) => {
          const setting = ifaceSettings[nic.name]
          if (setting?.role !== 'local') return []
          return [{ name: nic.name, select: selectorFor(nic), address: setting.cidr }]
        }),
      }

      await api.apiSetupNetworkPut({ SetupNetwork: network })
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
      else toast.error('Apply failed')
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
    if (!s) return true
    if (s.role === 'local') return !!s.cidr && !checkCIDR(s.cidr)
    if (s.role === 'internet' && s.mode === 'static') {
      return !!s.cidr && !checkCIDR(s.cidr) && !!s.gateway && !checkIP(s.gateway) && (s.dns ?? []).every((entry) => !checkIP(entry))
    }
    return true
  })

  return (
    <LoginScreen title="Welcome to Routier" logo={Logo}>
      <div className="space-y-6">
        <p className="text-center text-sm text-muted-foreground sm:hidden">
          Step {stepIdx + 1} of {STEPS.length} · <span className="font-medium text-foreground">{STEPS[stepIdx]?.label}</span>
        </p>
        <div className="hidden items-center justify-center gap-2 sm:flex">
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
            <PreferencesGroup title="Set a New Admin Password" description="The current password is the one shown on the console or printed when Routier was installed">
              <EntryRow title="Current Password" type="password" value={curPw} onChange={(e) => setCurPw(e.target.value)} autoFocus />
              <EntryRow title="New Password" type="password" value={newPw} onChange={(e) => setNewPw(e.target.value)}
                error={newPw && newPw.length < 8 ? 'At least 8 characters' : null} />
              <EntryRow title="Confirm New Password" type="password" value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)}
                error={confirmPw && confirmPw !== newPw ? 'Does not match' : null} />
            </PreferencesGroup>
            <div className="flex justify-end">
              <Button onClick={savePassword} disabled={busy || !curPw || !newPw || newPw !== confirmPw}>
                Set Password
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
            <PreferencesGroup title="Network Interfaces">
              {nicList.length === 0 && (
                <p className="px-4 py-3 text-sm text-muted-foreground">No physical interfaces detected</p>
              )}
              {nicList.map(nic => {
                const s = ifaceSettings[nic.name] ?? { role: 'none' as IfaceRole, cidr: '', mode: 'dhcp' as AddrMode, gateway: '', dns: [] }
                const staticUplink = s.role === 'internet' && s.mode === 'static'
                const cidrErr = (s.role === 'local' || staticUplink) ? checkCIDR(s.cidr) : null
                const gatewayErr = staticUplink ? checkIP(s.gateway) : null
                const isOk = (s.role === 'internet' && s.mode === 'dhcp')
                  || (staticUplink && !cidrErr && !!s.cidr && !gatewayErr && !!s.gateway)
                  || (s.role === 'local' && !cidrErr && !!s.cidr)
                return (
                  <div key={nic.name} className="space-y-2">
                    <Row
                      title={<span className="font-mono">{nic.name}</span>}
                      subtitle={s.role === 'internet'
                        ? 'Connects to the modem or upstream network'
                        : s.role === 'local'
                          ? 'Connect computers and other devices to this local network.'
                          : 'Do not configure this port yet.'}
                    >
                      {isOk && <Badge variant="success">Ready</Badge>}
                      {nic.operstate && <StateChip state={nic.operstate} />}
                      <Segmented
                        value={s.role}
                        onChange={(role) => setRole(nic.name, role)}
                        options={[
                          { value: 'internet', label: 'Internet' },
                          { value: 'local', label: 'Local' },
                          { value: 'none', label: 'Unused' },
                        ]}
                        className="w-full sm:w-auto"
                      />
                    </Row>

                    {s.role === 'local' ? (
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
                    ) : s.role === 'internet' ? (
                      <div className="space-y-2">
                        <Row title="Addressing">
                          <Segmented
                            value={s.mode}
                            onChange={(mode) => setSetting(nic.name, { mode: mode as AddrMode })}
                            options={[
                              { value: 'dhcp', label: 'Automatic (DHCP)' },
                              { value: 'static', label: 'Static' },
                            ]}
                            className="w-full sm:w-auto"
                          />
                        </Row>
                        {s.mode === 'static' ? (
                          <>
                            <EntryRow
                              title="Address (CIDR)"
                              value={s.cidr}
                              onChange={(e) => setSetting(nic.name, { cidr: e.target.value })}
                              placeholder="203.0.113.2/24"
                              className="font-mono"
                              error={cidrErr}
                            />
                            <EntryRow
                              title="Gateway"
                              value={s.gateway}
                              onChange={(e) => setSetting(nic.name, { gateway: e.target.value })}
                              placeholder="203.0.113.1"
                              className="font-mono"
                              error={gatewayErr}
                            />
                            <div className="space-y-1.5 px-1">
                              <p className="text-sm font-medium">DNS servers</p>
                              <TagInput
                                values={s.dns}
                                onChange={(dns) => setSetting(nic.name, { dns })}
                                placeholder="1.1.1.1"
                                mono
                                validate={checkIP}
                              />
                            </div>
                          </>
                        ) : (
                          <p className="px-4 pb-3 text-xs text-muted-foreground">
                            The uplink uses automatic IPv4 (DHCP) and IPv6 (SLAAC).
                            {nic.addrs && nic.addrs.length > 0 && (
                              <> Current: <span className="font-mono">{nic.addrs.join('  ')}</span></>
                            )}
                          </p>
                        )}
                      </div>
                    ) : null}
                  </div>
                )
              })}
            </PreferencesGroup>

            <StepNav
              onBack={() => setStep('hostname')}
              onNext={() => setStep('review')}
              nextDisabled={nicList.length === 0 || !ifacesValid}
            />
          </>
        )}

        {step === 'review' && (
          <>
            <PreferencesGroup title="Review">
              <Row title="Hostname">{hostname}</Row>
              {nicList.map(nic => {
                const s = ifaceSettings[nic.name] ?? { role: 'none' as IfaceRole, cidr: '', mode: 'dhcp' as AddrMode, gateway: '', dns: [] }
                const summary = s.role === 'internet'
                  ? (s.mode === 'static' ? `Internet uplink · static ${s.cidr}` : 'Internet uplink · DHCP')
                  : s.role === 'local' ? `Local · ${s.cidr}` : 'Unused'
                return (
                  <Row key={nic.name} title={nic.name}>
                    <span className="font-mono">{summary}</span>
                  </Row>
                )
              })}
            </PreferencesGroup>
            <p className="px-1 text-xs text-muted-foreground">
              Applying brings the interfaces up; confirm connectivity within 60 s or the changes roll back.
              {host && !host.taken_over && ' Until you confirm, the current network setup stays in charge; after that, Routier manages networking and keeps a backup of the previous setup.'}
            </p>
            <StepNav onBack={() => setStep('interfaces')} onNext={apply} nextLabel={busy ? 'Applying…' : 'Apply'} nextDisabled={busy} />
          </>
        )}

        {step === 'confirm' && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground text-center">
              Applied. Confirm connectivity before the timer runs out.
            </p>
            <WatchdogConfirmBar onKept={onKept} onRolledBack={onRolledBack} />
          </div>
        )}
      </div>
    </LoginScreen>
  )
}
