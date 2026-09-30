import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { api, configLayerRequest } from '@/lib/client'
import type { TypesSystemNic as SystemNic } from '@/api'
import { useFetch } from '@/lib/useFetch'
import { Badge, Button, EntryRow, LoginScreen, PreferencesGroup, Row, Segmented, TagInput } from 'cheval-ui'
import { checkCIDR, checkIP } from '@/lib/validate'
import type { Internet, Network } from '@/components/simple/types'
import { cn } from 'cheval-ui'
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

type IfaceRole = 'internet' | 'local' | 'none'
type AddrMode = 'dhcp' | 'static'
interface IfaceSetting { role: IfaceRole; cidr: string; mode: AddrMode; gateway: string; dns: string[] }

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
    if (!nics) return
    setIfaceSettings(prev => {
      const next = { ...prev }
      for (const nic of nics) {
        if (next[nic.name]?.role) continue
        const firstAddr = nic.addrs?.[0]
        next[nic.name] = { role: 'none', cidr: firstAddr ?? '', mode: 'dhcp', gateway: '', dns: [] }
      }
      return next
    })
  }, [nics])

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
      const internet: Internet | null = uplinkNIC ? (uplink?.mode === 'static' ? {
        interface: 'wan',
        select: selectorFor(uplinkNIC),
        mode: 'static',
        ipv6: 'slaac',
        addresses: [uplink.cidr],
        gateway: uplink.gateway,
        dns: uplink.dns,
      } : {
        interface: 'wan',
        select: selectorFor(uplinkNIC),
        mode: 'dhcp',
        ipv6: 'slaac',
        addresses: [],
      }) : null
      const networks: Network[] = nicList.flatMap((nic) => {
        const setting = ifaceSettings[nic.name]
        if (setting?.role !== 'local') return []
        return [{
          id: nic.name,
          name: nic.name,
          select: selectorFor(nic),
          addresses: [setting.cidr],
          manage_dhcp: false,
          masquerade: internet !== null,
          editable: true,
        }]
      })

      await configLayerRequest('/api/config/hostname', { method: 'PUT', body: JSON.stringify(hostname) }, 'simple')
      await configLayerRequest('/api/config/internet', { method: 'PUT', body: JSON.stringify(internet) }, 'simple')
      await configLayerRequest('/api/config/networks', { method: 'PUT', body: JSON.stringify(networks) }, 'simple')
      let applied = false
      try {
        const result = await configLayerRequest<{ warning?: string }>('/api/config/apply', { method: 'POST' }, 'simple')
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
    if (!s) return true
    if (s.role === 'local') return !!s.cidr && !checkCIDR(s.cidr)
    if (s.role === 'internet' && s.mode === 'static') {
      return !!s.cidr && !checkCIDR(s.cidr) && !!s.gateway && !checkIP(s.gateway) && (s.dns ?? []).every((entry) => !checkIP(entry))
    }
    return true
  })

  return (
    <LoginScreen title="Welcome to Routier" subtitle="Let's get your router set up." logo={Logo}>
      <div className="space-y-6">
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
            <PreferencesGroup title="Network interfaces" description="Choose what each physical port connects to. You can leave ports unused.">
              {nicList.length === 0 && (
                <p className="px-4 py-3 text-sm text-muted-foreground">No physical interfaces detected.</p>
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
                        ? 'Connect this port to your modem or upstream network.'
                        : s.role === 'local'
                          ? 'Connect computers and other devices to this local network.'
                          : 'Do not configure this port yet.'}
                    >
                      {isOk && <Badge variant="success">Ready</Badge>}
                      {nic.operstate && <Badge variant={nic.operstate === 'up' ? 'success' : 'neutral'}>{nic.operstate}</Badge>}
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
                        <Row title="Addressing" subtitle="How this port gets its IP address from the upstream network.">
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
    </LoginScreen>
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

function selectorFor(nic: SystemNic): string {
  return nic.mac ? `mac(${nic.mac})` : `name=${nic.name}`
}
