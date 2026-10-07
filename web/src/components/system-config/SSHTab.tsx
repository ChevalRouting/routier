import { OnStateChange, ROOT_LOGIN_OPTS, SSHConfig, TCP_FWD_OPTS, YES_NO } from '@/components/system-config/shared'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { checkPort } from '@/lib/validate'
import { Button, ComboRow, EntryRow, Input, PreferencesGroup, PreferencesGroups, Row, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Spinner } from 'cheval-ui'
import { Plus, X } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'

type SSHTabShape = { onStateChange: OnStateChange }

export function SSHTab({ onStateChange }: SSHTabShape) {
  const { data, isLoading } = useFetch<SSHConfig>(() => api.apiConfigSectionGet({ section: 'ssh' }) as Promise<SSHConfig>)
  const [port, setPort] = useState('')
  const [permitRootLogin, setPermitRootLogin] = useState('')
  const [passwordAuth, setPasswordAuth] = useState('')
  const [pubkeyAuth, setPubkeyAuth] = useState('')
  const [allowTcpForwarding, setAllowTcpForwarding] = useState('')
  const [x11Forwarding, setX11Forwarding] = useState('')
  const [maxAuthTries, setMaxAuthTries] = useState('')
  const [loginGraceTime, setLoginGraceTime] = useState('')
  const [clientAliveInterval, setClientAliveInterval] = useState('')
  const [clientAliveCountMax, setClientAliveCountMax] = useState('')
  const [allowUsers, setAllowUsers] = useState<string[]>([])
  const [allowGroups, setAllowGroups] = useState<string[]>([])
  const [banner, setBanner] = useState('')
  const [newUser, setNewUser] = useState('')
  const [newGroup, setNewGroup] = useState('')
  const [initialized, setInitialized] = useState(false)
  const { isDirty, markDirty, save, saving } = usePageSave('ssh')
  useDataRefresh(() => setInitialized(false))

  useEffect(() => {
    if (data !== null && !initialized) {
      setPort(data?.port ? String(data.port) : '')
      setPermitRootLogin(data?.permit_root_login ?? ''); setPasswordAuth(data?.password_auth ?? '')
      setPubkeyAuth(data?.pubkey_auth ?? ''); setAllowTcpForwarding(data?.allow_tcp_forwarding ?? '')
      setX11Forwarding(data?.x11_forwarding ?? ''); setMaxAuthTries(data?.max_auth_tries ? String(data.max_auth_tries) : '')
      setLoginGraceTime(data?.login_grace_time ? String(data.login_grace_time) : '')
      setClientAliveInterval(data?.client_alive_interval ? String(data.client_alive_interval) : '')
      setClientAliveCountMax(data?.client_alive_count_max ? String(data.client_alive_count_max) : '')
      setAllowUsers(data?.allow_users ?? []); setAllowGroups(data?.allow_groups ?? [])
      setBanner(data?.banner ?? ''); setInitialized(true)
    }
  }, [data, initialized])

  const buildPayload = useCallback((): SSHConfig | null => {
    const cfg: SSHConfig = {}
    const p = parseInt(port); if (!isNaN(p) && p > 0) cfg.port = p
    if (permitRootLogin) cfg.permit_root_login = permitRootLogin
    if (passwordAuth) cfg.password_auth = passwordAuth
    if (pubkeyAuth) cfg.pubkey_auth = pubkeyAuth
    if (allowTcpForwarding) cfg.allow_tcp_forwarding = allowTcpForwarding
    if (x11Forwarding) cfg.x11_forwarding = x11Forwarding
    const mat = parseInt(maxAuthTries); if (!isNaN(mat) && mat > 0) cfg.max_auth_tries = mat
    const lgt = parseInt(loginGraceTime); if (!isNaN(lgt) && lgt > 0) cfg.login_grace_time = lgt
    const cai = parseInt(clientAliveInterval); if (!isNaN(cai) && cai >= 0) cfg.client_alive_interval = cai
    const cac = parseInt(clientAliveCountMax); if (!isNaN(cac) && cac >= 0) cfg.client_alive_count_max = cac
    if (allowUsers.length > 0) cfg.allow_users = allowUsers
    if (allowGroups.length > 0) cfg.allow_groups = allowGroups
    if (banner) cfg.banner = banner
    return Object.keys(cfg).length === 0 ? null : cfg
  }, [port, permitRootLogin, passwordAuth, pubkeyAuth, allowTcpForwarding, x11Forwarding, maxAuthTries, loginGraceTime, clientAliveInterval, clientAliveCountMax, allowUsers, allowGroups, banner])

  const handleSave = useCallback(() => save(buildPayload()), [save, buildPayload])
  useEffect(() => { onStateChange({ isDirty, saving, save: handleSave }) }, [isDirty, saving, handleSave, onStateChange])

  const sel = (value: string, onChange: (v: string) => void) => ({
    value: value || '_none',
    onValueChange: (v: string) => { onChange(v === '_none' ? '' : v); markDirty() },
  })

  const optionSelect = (value: string, onChange: (v: string) => void, opts: string[]) => (
    <Select {...sel(value, onChange)}>
      <SelectTrigger><SelectValue /></SelectTrigger>
      <SelectContent>
        <SelectItem value="_none">(default)</SelectItem>
        {opts.map((v) => <SelectItem key={v} value={v}>{v}</SelectItem>)}
      </SelectContent>
    </Select>
  )

  const addToList = (list: string[], set: React.Dispatch<React.SetStateAction<string[]>>, val: string, setVal: React.Dispatch<React.SetStateAction<string>>) => {
    const t = val.trim()
    if (!t || list.includes(t)) return
    set([...list, t]); setVal(''); markDirty()
  }

  if (isLoading) return <Spinner />

  return (
    <PreferencesGroups className="max-w-4xl">
      <PreferencesGroup title="Connection">
        <EntryRow title="Port" value={port} onChange={(e) => { setPort(e.target.value); markDirty() }} placeholder="22 (default)" className="font-mono" error={checkPort(port)} />
        <EntryRow title="Max auth tries" value={maxAuthTries} onChange={(e) => { setMaxAuthTries(e.target.value); markDirty() }} placeholder="6 (default)" className="font-mono" />
        <EntryRow title="Login grace time (s)" value={loginGraceTime} onChange={(e) => { setLoginGraceTime(e.target.value); markDirty() }} placeholder="120 (default)" className="font-mono" />
        <EntryRow title="Client alive interval (s)" value={clientAliveInterval} onChange={(e) => { setClientAliveInterval(e.target.value); markDirty() }} placeholder="0 (disabled)" className="font-mono" />
        <EntryRow title="Client alive count max" value={clientAliveCountMax} onChange={(e) => { setClientAliveCountMax(e.target.value); markDirty() }} placeholder="3 (default)" className="font-mono" />
      </PreferencesGroup>

      <PreferencesGroup title="Authentication">
        <ComboRow title="Permit root login">{optionSelect(permitRootLogin, setPermitRootLogin, ROOT_LOGIN_OPTS)}</ComboRow>
        <ComboRow title="Password authentication">{optionSelect(passwordAuth, setPasswordAuth, YES_NO)}</ComboRow>
        <ComboRow title="Pubkey authentication">{optionSelect(pubkeyAuth, setPubkeyAuth, YES_NO)}</ComboRow>
      </PreferencesGroup>

      <PreferencesGroup title="Forwarding">
        <ComboRow title="Allow TCP forwarding">{optionSelect(allowTcpForwarding, setAllowTcpForwarding, TCP_FWD_OPTS)}</ComboRow>
        <ComboRow title="X11 forwarding">{optionSelect(x11Forwarding, setX11Forwarding, YES_NO)}</ComboRow>
      </PreferencesGroup>

      <PreferencesGroup title="Allow users" description="When empty, all users may log in.">
        {allowUsers.map((u) => (
          <Row key={u} title={<span className="font-mono">{u}</span>}>
            <button onClick={() => { setAllowUsers(allowUsers.filter((x) => x !== u)); markDirty() }} className="text-muted-foreground hover:text-destructive" aria-label={`Remove ${u}`}>
              <X className="h-4 w-4" />
            </button>
          </Row>
        ))}
        <ComboRow title="Add user">
<div className="flex w-full items-center gap-2">
          <Input value={newUser} onChange={(e) => setNewUser(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addToList(allowUsers, setAllowUsers, newUser, setNewUser)} placeholder="username" className="min-w-0 flex-1 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50" />
          <Button variant="ghost" size="sm" onClick={() => addToList(allowUsers, setAllowUsers, newUser, setNewUser)} className="shrink-0 gap-1"><Plus className="h-4 w-4" />Add</Button>
        </div>
</ComboRow>
      </PreferencesGroup>

      <PreferencesGroup title="Allow groups" description="When empty, all groups may log in.">
        {allowGroups.map((g) => (
          <Row key={g} title={<span className="font-mono">{g}</span>}>
            <button onClick={() => { setAllowGroups(allowGroups.filter((x) => x !== g)); markDirty() }} className="text-muted-foreground hover:text-destructive" aria-label={`Remove ${g}`}>
              <X className="h-4 w-4" />
            </button>
          </Row>
        ))}
        <ComboRow title="Add group">
<div className="flex w-full items-center gap-2">
          <Input value={newGroup} onChange={(e) => setNewGroup(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addToList(allowGroups, setAllowGroups, newGroup, setNewGroup)} placeholder="groupname" className="min-w-0 flex-1 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50" />
          <Button variant="ghost" size="sm" onClick={() => addToList(allowGroups, setAllowGroups, newGroup, setNewGroup)} className="shrink-0 gap-1"><Plus className="h-4 w-4" />Add</Button>
        </div>
</ComboRow>
      </PreferencesGroup>

      <PreferencesGroup title="Banner" description="Path to a file whose contents are sent to the client before authentication.">
        <EntryRow title="Banner file" value={banner} onChange={(e) => { setBanner(e.target.value); markDirty() }} placeholder="/etc/issue.net" className="font-mono" />
      </PreferencesGroup>
    </PreferencesGroups>
  )
}
