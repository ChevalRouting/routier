import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import type { TypesApplyLogRecord as ApplyLogRecord, TypesSnapshotInfo as SnapshotInfo, TypesAPIKey } from '@/api'
import type { ConfigMonitoringConfig as MonitoringConfig, ConfigLLDPConfig as LLDPConfig } from '@/api'
import { WatchdogConfirmBar } from '@/components/WatchdogConfirmBar'
import UpdatesPanel from '@/components/system/UpdatesPanel'
import { useTabState } from 'cheval-ui'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { useDataRefresh } from '@/lib/dataVersion'
import { SectionNav } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { Input } from 'cheval-ui'
import { CopyButton } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { Separator } from 'cheval-ui'
import { SectionLabel } from 'cheval-ui'
import { PreferencesGroup, PreferencesGroups, PreferencesColumns, Row, EntryRow, ComboRow } from 'cheval-ui'
import { SwitchRow } from 'cheval-ui'
import { TagInput } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { Dialog, AlertDialog } from 'cheval-ui'
import { Pagination, usePagination } from 'cheval-ui'
import { checkPort, isIP } from '@/lib/validate'
import { SaveButton } from 'cheval-ui'
import { PageHeader } from 'cheval-ui'
import { Spinner } from 'cheval-ui'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from 'cheval-ui'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from 'cheval-ui'
import { Plus, Trash2, X, Download, Upload, RotateCcw, RefreshCw, SlidersHorizontal, ExternalLink, FileJson, History, KeyRound } from 'lucide-react'

interface TabSaveState {
  isDirty: boolean
  saving: boolean
  save: () => void
}

type OnStateChange = (s: TabSaveState) => void


function HostnameTab({ onStateChange }: { onStateChange: OnStateChange }) {
  const { data, isLoading } = useFetch<string>(() => api.apiConfigSectionGet({ section: 'hostname' }) as unknown as Promise<string>)
  const [hostname, setHostname] = useState('')
  const [initialized, setInitialized] = useState(false)
  const { isDirty, markDirty, save, saving } = usePageSave('hostname')
  useDataRefresh(() => setInitialized(false))

  useEffect(() => {
    if (data !== null && data !== undefined && !initialized) {
      setHostname(data); setInitialized(true)
    }
  }, [data, initialized])

  const handleSave = useCallback(() => save(hostname), [save, hostname])
  useEffect(() => { onStateChange({ isDirty, saving, save: handleSave }) }, [isDirty, saving, handleSave, onStateChange])

  if (isLoading) return <Spinner />

  return (
    <div className="max-w-xl">
      <PreferencesGroup description="Written to /etc/hostname and applied immediately on config apply.">
        <EntryRow
          id="hostname"
          title="Hostname"
          value={hostname}
          onChange={(e) => { setHostname(e.target.value); markDirty() }}
          placeholder="router.example.com"
          className="font-mono"
        />
      </PreferencesGroup>
    </div>
  )
}


interface DNSConfig { nameservers?: string[]; search?: string[] }

function DNSTab({ onStateChange }: { onStateChange: OnStateChange }) {
  const { data, isLoading } = useFetch<DNSConfig>(() => api.apiConfigSectionGet({ section: 'dns' }) as Promise<DNSConfig>)
  const [nameservers, setNameservers] = useState<string[]>([])
  const [search, setSearch] = useState<string[]>([])
  const [newNs, setNewNs] = useState('')
  const [newSearch, setNewSearch] = useState('')
  const [initialized, setInitialized] = useState(false)
  const { isDirty, markDirty, save, saving } = usePageSave('dns')
  useDataRefresh(() => setInitialized(false))

  useEffect(() => {
    if (data && !initialized) { setNameservers(data.nameservers ?? []); setSearch(data.search ?? []); setInitialized(true) }
  }, [data, initialized])

  const handleSave = useCallback(() => save({ nameservers, search }), [save, nameservers, search])
  useEffect(() => { onStateChange({ isDirty, saving, save: handleSave }) }, [isDirty, saving, handleSave, onStateChange])

  const addNs = () => {
    const t = newNs.trim()
    if (!t || !isIP(t) || nameservers.includes(t)) return
    setNameservers([...nameservers, t]); setNewNs(''); markDirty()
  }
  const addSearch = () => {
    const t = newSearch.trim()
    if (!t || search.includes(t)) return
    setSearch([...search, t]); setNewSearch(''); markDirty()
  }

  if (isLoading) return <Spinner />

  return (
    <PreferencesGroups className="max-w-4xl">
      <PreferencesGroup title="Nameservers" description="DNS resolver IP addresses">
        {nameservers.map((ns) => (
          <Row key={ns} title={<span className="font-mono">{ns}</span>}>
            <button onClick={() => { setNameservers(nameservers.filter((n) => n !== ns)); markDirty() }} className="text-muted-foreground hover:text-destructive" aria-label={`Remove ${ns}`}>
              <X className="h-4 w-4" />
            </button>
          </Row>
        ))}
        <div className="flex items-center gap-2 px-4 py-2">
          <input value={newNs} onChange={(e) => setNewNs(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addNs()} placeholder="8.8.8.8" className={`min-w-0 flex-1 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50 ${newNs.trim() && !isIP(newNs.trim()) ? 'text-destructive' : ''}`} />
          <Button variant="ghost" size="sm" onClick={addNs} disabled={!!newNs.trim() && !isIP(newNs.trim())} className="shrink-0 gap-1"><Plus className="h-4 w-4" />Add</Button>
        </div>
      </PreferencesGroup>

      <PreferencesGroup title="Search domains" description="DNS search domain suffixes">
        {search.map((s) => (
          <Row key={s} title={<span className="font-mono">{s}</span>}>
            <button onClick={() => { setSearch(search.filter((n) => n !== s)); markDirty() }} className="text-muted-foreground hover:text-destructive" aria-label={`Remove ${s}`}>
              <X className="h-4 w-4" />
            </button>
          </Row>
        ))}
        <div className="flex items-center gap-2 px-4 py-2">
          <input value={newSearch} onChange={(e) => setNewSearch(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addSearch()} placeholder="example.com" className="min-w-0 flex-1 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50" />
          <Button variant="ghost" size="sm" onClick={addSearch} className="shrink-0 gap-1"><Plus className="h-4 w-4" />Add</Button>
        </div>
      </PreferencesGroup>
    </PreferencesGroups>
  )
}


interface SSHConfig {
  port?: number; permit_root_login?: string; password_auth?: string; pubkey_auth?: string
  allow_tcp_forwarding?: string; x11_forwarding?: string; max_auth_tries?: number
  login_grace_time?: number; client_alive_interval?: number; client_alive_count_max?: number
  allow_users?: string[]; allow_groups?: string[]; banner?: string
}

const YES_NO = ['yes', 'no']
const ROOT_LOGIN_OPTS = ['yes', 'no', 'prohibit-password', 'forced-commands-only']
const TCP_FWD_OPTS = ['yes', 'no', 'local', 'remote']

function SSHTab({ onStateChange }: { onStateChange: OnStateChange }) {
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
        <div className="flex items-center gap-2 px-4 py-2">
          <input value={newUser} onChange={(e) => setNewUser(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addToList(allowUsers, setAllowUsers, newUser, setNewUser)} placeholder="username" className="min-w-0 flex-1 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50" />
          <Button variant="ghost" size="sm" onClick={() => addToList(allowUsers, setAllowUsers, newUser, setNewUser)} className="shrink-0 gap-1"><Plus className="h-4 w-4" />Add</Button>
        </div>
      </PreferencesGroup>

      <PreferencesGroup title="Allow groups" description="When empty, all groups may log in.">
        {allowGroups.map((g) => (
          <Row key={g} title={<span className="font-mono">{g}</span>}>
            <button onClick={() => { setAllowGroups(allowGroups.filter((x) => x !== g)); markDirty() }} className="text-muted-foreground hover:text-destructive" aria-label={`Remove ${g}`}>
              <X className="h-4 w-4" />
            </button>
          </Row>
        ))}
        <div className="flex items-center gap-2 px-4 py-2">
          <input value={newGroup} onChange={(e) => setNewGroup(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && addToList(allowGroups, setAllowGroups, newGroup, setNewGroup)} placeholder="groupname" className="min-w-0 flex-1 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50" />
          <Button variant="ghost" size="sm" onClick={() => addToList(allowGroups, setAllowGroups, newGroup, setNewGroup)} className="shrink-0 gap-1"><Plus className="h-4 w-4" />Add</Button>
        </div>
      </PreferencesGroup>

      <PreferencesGroup title="Banner" description="Path to a file whose contents are sent to the client before authentication.">
        <EntryRow title="Banner file" value={banner} onChange={(e) => { setBanner(e.target.value); markDirty() }} placeholder="/etc/issue.net" className="font-mono" />
      </PreferencesGroup>
    </PreferencesGroups>
  )
}


interface SysctlEntry { key: string; value: string }

function parseEntries(data: unknown): SysctlEntry[] {
  if (!data || typeof data !== 'object' || Array.isArray(data)) return []
  return Object.entries(data as Record<string, string>).map(([key, value]) => ({ key, value }))
}

function toRecord(entries: SysctlEntry[]): Record<string, string> {
  const r: Record<string, string> = {}
  for (const { key, value } of entries) { if (key.trim()) r[key.trim()] = value }
  return r
}

function SysctlTab({ onStateChange }: { onStateChange: OnStateChange }) {
  const { data, isLoading } = useFetch(() => api.apiConfigSectionGet({ section: 'sysctl' }))
  const [entries, setEntries] = useState<SysctlEntry[] | null>(null)
  const { isDirty, markDirty, save, saving } = usePageSave('sysctl')

  const parsed = useMemo(() => parseEntries(data), [data])
  const current = entries ?? parsed

  const handleSave = useCallback(() => save(toRecord(current)), [save, current])
  useEffect(() => { onStateChange({ isDirty, saving, save: handleSave }) }, [isDirty, saving, handleSave, onStateChange])

  const handleChange = (index: number, field: 'key' | 'value', val: string) => {
    setEntries(current.map((e, i) => (i === index ? { ...e, [field]: val } : e))); markDirty()
  }
  const handleAdd = () => { setEntries([...current, { key: '', value: '' }]); markDirty() }
  const handleRemove = (index: number) => { setEntries(current.filter((_, i) => i !== index)); markDirty() }

  if (isLoading) return <Spinner />

  return (
    <>
      {current.length === 0 ? (
        <EmptyState
          className="max-w-2xl mx-auto"
          icon={<SlidersHorizontal />}
          title="No parameters"
          message="Written to /etc/sysctl.d/99-routier.conf and applied on each config apply."
          action={<Button variant="outline" onClick={handleAdd} className="gap-2"><Plus className="h-4 w-4" />Add parameter</Button>}
        />
      ) : (
        <PreferencesColumns
          title="Parameters"
          description="Written to /etc/sysctl.d/99-routier.conf and applied on each config apply."
          header={
            <Button variant="outline" size="sm" onClick={handleAdd} className="gap-1.5">
              <Plus className="h-4 w-4" />Add
            </Button>
          }
        >
          {current.map((entry, index) => (
            <div key={index} className="flex items-center gap-2 px-4 py-2">
              <input
                value={entry.key}
                onChange={(e) => handleChange(index, 'key', e.target.value)}
                placeholder="net.ipv4.ip_forward"
                className="min-w-0 flex-1 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50"
              />
              <span className="shrink-0 text-muted-foreground">=</span>
              <input
                value={entry.value}
                onChange={(e) => handleChange(index, 'value', e.target.value)}
                placeholder="1"
                className="w-32 shrink-0 bg-transparent font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/50"
              />
              <Button
                variant="ghost"
                size="icon"
                onClick={() => handleRemove(index)}
                className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))}
        </PreferencesColumns>
      )}
    </>
  )
}


function resultBadge(r: ApplyLogRecord) {
  if (r.confirmed_at) return <Badge className="bg-success/15 text-success border-success/25">confirmed</Badge>
  if (r.rolledback_at || r.result === 'rolledback') return <Badge className="bg-danger/15 text-danger border-danger/25">rolled back</Badge>
  if (r.result === 'failed') return <Badge className="bg-danger/15 text-danger border-danger/25">failed</Badge>
  if (r.result === 'applied') return <Badge className="bg-warning/15 text-warning border-warning/25">applied</Badge>
  return <Badge variant="outline">{r.result || 'running'}</Badge>
}

function ApplyHistoryTab() {
  const { data, isLoading, reload } = useFetch(() => api.apiApplyLogsGet())
  const [openId, setOpenId] = useState<string | null>(null)
  const [logText, setLogText] = useState('')

  const view = (id: string) => {
    setOpenId(id); setLogText('Loading…')
    api.apiApplyLogsIdGet({ id }).then(setLogText).catch(() => setLogText('Failed to load log.'))
  }

  const records = data?.records ?? []
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(records, 15)
  if (isLoading) return <Spinner />

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">Recent configuration applies, restores and rollbacks.</p>
        <Button variant="outline" size="sm" onClick={reload} className="gap-2"><RefreshCw className="h-3.5 w-3.5" />Refresh</Button>
      </div>
      {records.length === 0 ? (
        <EmptyState
          icon={<History />}
          title="No applies yet"
          message="Configuration applies, restores and rollbacks will show up here."
        />
      ) : (
        <>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Started</TableHead><TableHead>Source</TableHead>
              <TableHead>Status</TableHead><TableHead>Snapshot</TableHead><TableHead></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {pageItems.map((r) => (
              <TableRow key={r.id}>
                <TableCell className="font-mono text-xs">{new Date(r.started_at).toLocaleString()}</TableCell>
                <TableCell><Badge variant="outline">{r.source}</Badge></TableCell>
                <TableCell>{resultBadge(r)}</TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">{r.snap_id || '-'}</TableCell>
                <TableCell className="text-right">
                  <Button variant="ghost" size="sm" onClick={() => view(r.id)}>View log</Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="applies" />
        </>
      )}

      <Dialog open={!!openId} onClose={() => setOpenId(null)} title={`Apply log · ${openId ?? ''}`} className="max-w-3xl">
        <pre className="m-0 overflow-auto whitespace-pre-wrap rounded-md bg-[#09090b] p-4 font-mono text-xs text-zinc-200">{logText}</pre>
      </Dialog>
    </div>
  )
}


function SnapshotsTab({ onChanged }: { onChanged: () => void }) {
  const { data, isLoading, reload } = useFetch(() => api.apiSnapshotsGet())
  const [busy, setBusy] = useState<string | null>(null)
  const [confirm, setConfirm] = useState<SnapshotInfo | null>(null)

  const restore = (s: SnapshotInfo) => setConfirm(s)

  const doRestore = () => {
    const s = confirm
    if (!s) return
    setConfirm(null)
    setBusy(s.id)
    api.apiSnapshotsIdRestorePost({ id: s.id })
      .then(() => { toast.success('Snapshot restored'); reload(); onChanged() })
      .catch((e) => toast.error(`Restore failed: ${e.message}`))
      .finally(() => setBusy(null))
  }

  const snapshots = data?.snapshots ?? []
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(snapshots, 15)
  if (isLoading) return <Spinner />

  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">Each apply records a snapshot of the files it changed. Restore reverts to a previous state.</p>
      {snapshots.length === 0 ? (
        <EmptyState
          icon={<RotateCcw />}
          title="No snapshots"
          message="Every apply snapshots the previous state so it can be restored."
        />
      ) : (
        <>
        <Table>
          <TableHeader>
            <TableRow><TableHead>Time</TableHead><TableHead>ID</TableHead><TableHead>Files</TableHead><TableHead></TableHead></TableRow>
          </TableHeader>
          <TableBody>
            {pageItems.map((s) => (
              <TableRow key={s.id}>
                <TableCell className="font-mono text-xs">{new Date(s.time).toLocaleString()}</TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">{s.id}</TableCell>
                <TableCell>{(s.files ?? []).length}</TableCell>
                <TableCell className="text-right">
                  <Button variant="outline" size="sm" className="gap-1.5" disabled={busy === s.id} onClick={() => restore(s)}>
                    <RotateCcw className="h-3.5 w-3.5" />Restore
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="snapshots" />
        </>
      )}

      <AlertDialog
        open={!!confirm}
        onCancel={() => setConfirm(null)}
        onConfirm={doRestore}
        title="Restore this snapshot?"
        description={confirm ? `Reverts the managed files to their state before apply ${confirm.id}. The current files will be overwritten.` : undefined}
        confirmLabel="Restore"
        cancelLabel="Cancel"
        destructive
      />
    </div>
  )
}


function BackupTab({ onChanged }: { onChanged: () => void }) {
  const fileRef = useRef<HTMLInputElement>(null)
  const [exporting, setExporting] = useState(false)
  const [importing, setImporting] = useState(false)

  const doExport = async () => {
    setExporting(true)
    try {
      const res = await api.apiBackupExportGetRaw()
      const blob = await res.raw.blob()
      const disposition = res.raw.headers.get('Content-Disposition') ?? ''
      const match = /filename=([^;]+)/.exec(disposition)
      const filename = match ? match[1].trim() : `routier-backup-${Date.now()}.bin`
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
    } catch (e) {
      toast.error(`Export failed: ${(e as Error).message}`)
    } finally {
      setExporting(false)
    }
  }

  const doImport = async (file: File) => {
    setImporting(true)
    try {
      const form = new FormData()
      form.append('file', file)
      const res = await api.apiBackupImportPostRaw({ method: 'POST', body: form })
      const parsed = await res.value()
      toast.success('Backup imported and applied')
      if (parsed.result?.warning) toast.warning(parsed.result.warning)
      onChanged()
    } catch (e) {
      toast.error(`Import failed: ${(e as Error).message}`)
    } finally {
      setImporting(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  return (
    <div className="space-y-5 max-w-xl">
      <div className="space-y-2">
        <SectionLabel>Export</SectionLabel>
        <p className="text-sm text-muted-foreground">
          Download a <code className="text-xs bg-muted px-1 py-0.5 rounded">.bin</code> archive (zstd) of the config and referenced files
          (nftables, WireGuard keys, template overrides). Keep it safe, it contains private keys.
        </p>
        <Button onClick={doExport} disabled={exporting} className="gap-2">
          {exporting ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}Export backup
        </Button>
      </div>
      <Separator />
      <div className="space-y-2">
        <SectionLabel>Import</SectionLabel>
        <p className="text-sm text-muted-foreground">Restore a <code className="text-xs bg-muted px-1 py-0.5 rounded">.bin</code> archive and apply it. The watchdog is armed, confirm above to keep the changes.</p>
        <input ref={fileRef} type="file" accept=".bin" className="hidden"
          onChange={(e) => { const f = e.target.files?.[0]; if (f) doImport(f) }} />
        <Button variant="outline" onClick={() => fileRef.current?.click()} disabled={importing} className="gap-2">
          {importing ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}Import backup
        </Button>
      </div>
    </div>
  )
}


function ApiKeysTab() {
  const { data, isLoading, reload } = useFetch(() => api.apiAuthApiKeysGet())
  const [name, setName] = useState('')
  const [creating, setCreating] = useState(false)
  const [created, setCreated] = useState<{ name: string; token: string } | null>(null)
  const [revoke, setRevoke] = useState<TypesAPIKey | null>(null)
  const [busy, setBusy] = useState<string | null>(null)

  const keys = (data ?? []).filter((k) => !k.revoked_at)

  const create = () => {
    const trimmed = name.trim()
    if (!trimmed) return
    setCreating(true)
    api.apiAuthApiKeysPost({ TypesCreateAPIKeyRequest: { name: trimmed } })
      .then((res) => {
        const token = res.token ?? ''
        setCreated({ name: trimmed, token })
        setName('')
        reload()
      })
      .catch((e) => toast.error(`Create failed: ${e.message}`))
      .finally(() => setCreating(false))
  }

  const doRevoke = () => {
    const k = revoke
    if (!k) return
    setRevoke(null)
    setBusy(k.id)
    api.apiAuthApiKeysIdDelete({ id: k.id })
      .then(() => { toast.success('API key revoked'); reload() })
      .catch((e) => toast.error(`Revoke failed: ${e.message}`))
      .finally(() => setBusy(null))
  }

  return (
    <div className="space-y-4">
      <SectionLabel>API Keys</SectionLabel>
      <p className="text-sm text-muted-foreground">
        API keys authenticate programmatic clients such as <code className="text-xs bg-muted px-1 py-0.5 rounded">routier-mcp</code> using a
        <code className="text-xs bg-muted px-1 py-0.5 rounded">Bearer</code> token. They cannot manage authentication credentials. The full token is
        shown only once at creation.
      </p>

      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') create() }}
          placeholder="Key name (e.g. edge-a-mcp)"
          className="max-w-xs"
          disabled={creating}
        />
        <Button onClick={create} disabled={creating || !name.trim()} className="gap-2">
          {creating ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}Generate key
        </Button>
      </div>

      {isLoading ? <Spinner /> : keys.length === 0 ? (
        <EmptyState
          icon={<KeyRound />}
          title="No API keys"
          message="Generate a key to authenticate programmatic clients."
        />
      ) : (
        <Table>
          <TableHeader>
            <TableRow><TableHead>Name</TableHead><TableHead>Prefix</TableHead><TableHead>Created</TableHead><TableHead>Created by</TableHead><TableHead></TableHead></TableRow>
          </TableHeader>
          <TableBody>
            {keys.map((k) => (
              <TableRow key={k.id}>
                <TableCell>{k.name}</TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">{k.prefix}…</TableCell>
                <TableCell className="font-mono text-xs">{new Date(k.created_at * 1000).toLocaleString()}</TableCell>
                <TableCell className="text-muted-foreground">{k.created_by}</TableCell>
                <TableCell className="text-right">
                  <Button variant="outline" size="sm" className="gap-1.5" disabled={busy === k.id} onClick={() => setRevoke(k)}>
                    <Trash2 className="h-3.5 w-3.5" />Revoke
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <Dialog open={!!created} onClose={() => setCreated(null)} title="API key created">
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Copy the token for <span className="font-medium text-foreground">{created?.name}</span> now. It will not be shown again.
          </p>
          <div className="flex items-center gap-2 rounded-md bg-[#09090b] p-3">
            <code className="flex-1 overflow-auto whitespace-nowrap font-mono text-xs text-zinc-200">{created?.token}</code>
            <CopyButton text={created?.token ?? ''} />
          </div>
        </div>
      </Dialog>

      <AlertDialog
        open={!!revoke}
        onCancel={() => setRevoke(null)}
        onConfirm={doRevoke}
        title="Revoke this API key?"
        description={revoke ? `Clients using "${revoke.name}" will immediately lose access. This cannot be undone.` : undefined}
        confirmLabel="Revoke"
        cancelLabel="Cancel"
        destructive
      />
    </div>
  )
}


function ApiDocsTab() {
  return (
    <div className="space-y-4">
      <SectionLabel>API Documentation</SectionLabel>
      <p className="text-sm text-muted-foreground">
        Interactive OpenAPI documentation for the Routier API, generated from the server.
      </p>
      <div className="flex flex-wrap gap-2">
        <Button asChild variant="outline" className="gap-2">
          <a href="/api/v1/docs/" target="_blank" rel="noopener noreferrer">
            <ExternalLink className="h-4 w-4" />Open API docs
          </a>
        </Button>
        <Button asChild variant="outline" className="gap-2">
          <a href="/api/v1/docs/openapi.json" target="_blank" rel="noopener noreferrer">
            <FileJson className="h-4 w-4" />OpenAPI spec
          </a>
        </Button>
      </div>
    </div>
  )
}

const INTERVAL_KEYS: Array<{ key: keyof NonNullable<MonitoringConfig['collection']>; label: string }> = [
  { key: 'iface',     label: 'Interfaces' },
  { key: 'system',    label: 'CPU / RAM' },
  { key: 'bgp',       label: 'BGP' },
  { key: 'proto',     label: 'Protocols' },
  { key: 'neighbors', label: 'Neighbors' },
  { key: 'lldp',      label: 'LLDP / CDP' },
  { key: 'routes',    label: 'Route cache TTL' },
]

function MonitoringTab({ onStateChange }: { onStateChange: OnStateChange }) {
  const { data, isLoading } = useFetch<MonitoringConfig>(() => api.apiConfigSectionGet({ section: 'monitoring' }) as Promise<MonitoringConfig>)
  const [monitoring, setMonitoring] = useState<MonitoringConfig>({})
  const [intervalRaw, setIntervalRaw] = useState<Record<string, string>>({})
  const [initialized, setInitialized] = useState(false)
  const { isDirty, markDirty, save, saving } = usePageSave('monitoring')
  useDataRefresh(() => setInitialized(false))

  useEffect(() => {
    if (data && !initialized) {
      setMonitoring(data)
      const raw: Record<string, string> = {}
      for (const { key } of INTERVAL_KEYS) raw[key] = String(data.collection?.[key] ?? 60)
      setIntervalRaw(raw)
      setInitialized(true)
    }
  }, [data, initialized])

  const handleSave = useCallback(() => save(monitoring), [save, monitoring])
  useEffect(() => { onStateChange({ isDirty, saving, save: handleSave }) }, [isDirty, saving, handleSave, onStateChange])

  const updateInterval = (key: string, raw: string) => {
    setIntervalRaw((p) => ({ ...p, [key]: raw }))
    const n = parseInt(raw, 10)
    if (!isNaN(n) && n >= 10) setMonitoring((prev) => ({ ...prev, collection: { ...prev.collection, [key]: n } }))
    markDirty()
  }

  const lldp = monitoring.lldp ?? {}
  const patchLldp = (next: Partial<LLDPConfig>) => { setMonitoring((prev) => ({ ...prev, lldp: { ...prev.lldp, ...next } })); markDirty() }

  if (isLoading) return <Spinner />

  return (
    <div className="max-w-2xl space-y-6">
      <PreferencesGroup title="Collection intervals" description="How often each metric is sampled, in seconds (minimum 10).">
        <div className="grid gap-3 grid-cols-2 px-4 py-3">
          {INTERVAL_KEYS.map(({ key, label }) => (
            <div key={key} className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">{label}</Label>
              <div className="flex items-center gap-1.5">
                <Input value={intervalRaw[key] ?? ''} onChange={(e) => updateInterval(key, e.target.value)} className="font-mono text-xs h-8" placeholder="60" />
                <span className="text-xs text-muted-foreground shrink-0">s</span>
              </div>
            </div>
          ))}
        </div>
      </PreferencesGroup>

      <PreferencesGroup title="Link-layer discovery (LLDP / CDP)" description="Discover directly connected switches and routers. Discovered devices appear in Monitor > Neighbors.">
        <SwitchRow
          title="Enable LLDP / CDP"
          subtitle="Run lldpd to discover directly connected devices"
          checked={!!lldp.enabled}
          onCheckedChange={(v) => patchLldp({ enabled: v })}
        />
      </PreferencesGroup>

      {lldp.enabled && (
        <>
          <PreferencesGroup>
            <SwitchRow
              title="Receive CDP"
              subtitle="Also discover neighbors advertising Cisco Discovery Protocol"
              checked={!!lldp.cdp}
              onCheckedChange={(v) => patchLldp({ cdp: v })}
            />
            <SwitchRow
              title="Advertise this router"
              subtitle="Transmit LLDP so upstream devices can discover this router. When off, lldpd only listens."
              checked={!!lldp.transmit}
              onCheckedChange={(v) => patchLldp({ transmit: v })}
            />
          </PreferencesGroup>

          <PreferencesGroup title="Interfaces" description="Restrict discovery to these interfaces. Leave empty for all.">
            <div className="px-4 py-3">
              <TagInput values={lldp.interfaces ?? []} onChange={(v) => patchLldp({ interfaces: v })} placeholder="eth1" mono />
            </div>
          </PreferencesGroup>
        </>
      )}
    </div>
  )
}

type SystemTab = 'hostname' | 'dns' | 'ssh' | 'sysctl' | 'monitoring' | 'updates' | 'apply' | 'snapshots' | 'backup' | 'apikeys' | 'apidocs'

const SYSTEM_TABS: { key: SystemTab; label: string }[] = [
  { key: 'hostname',  label: 'Hostname' },
  { key: 'dns',       label: 'DNS' },
  { key: 'ssh',       label: 'SSH' },
  { key: 'sysctl',    label: 'Sysctl' },
  { key: 'monitoring', label: 'Monitoring' },
  { key: 'updates',   label: 'Updates' },
  { key: 'apply',     label: 'Apply History' },
  { key: 'snapshots', label: 'Snapshots' },
  { key: 'backup',    label: 'Backup' },
  { key: 'apikeys',   label: 'API Keys' },
  { key: 'apidocs',   label: 'API Docs' },
]

const CONFIG_TABS: SystemTab[] = ['hostname', 'dns', 'ssh', 'sysctl', 'monitoring']

const EMPTY_SAVE: TabSaveState = { isDirty: false, saving: false, save: () => {} }

export default function SystemConfig() {
  const [activeTab, setActiveTab] = useTabState<SystemTab>('system-config', 'hostname')
  const [saveState, setSaveState] = useState<TabSaveState>(EMPTY_SAVE)
  const [pendingPoll, setPendingPoll] = useState(0)

  const handleTabChange = (tab: SystemTab) => {
    setActiveTab(tab)
    setSaveState(EMPTY_SAVE)
  }

  const isConfigTab = CONFIG_TABS.includes(activeTab)

  return (
    <div className="space-y-6">
      <PageHeader
        title="General"
        description="OS hostname, DNS, SSH daemon, kernel parameters, monitoring, apply history and backups"
        action={isConfigTab ? <SaveButton isDirty={saveState.isDirty} saving={saveState.saving} onClick={saveState.save} /> : undefined}
      />
      <WatchdogConfirmBar pollKey={pendingPoll} />
      <SectionNav items={SYSTEM_TABS} active={activeTab} onChange={handleTabChange}>
        {activeTab === 'hostname'  && <HostnameTab onStateChange={setSaveState} />}
        {activeTab === 'dns'       && <DNSTab onStateChange={setSaveState} />}
        {activeTab === 'ssh'       && <SSHTab onStateChange={setSaveState} />}
        {activeTab === 'sysctl'    && <SysctlTab onStateChange={setSaveState} />}
        {activeTab === 'monitoring' && <MonitoringTab onStateChange={setSaveState} />}
        {activeTab === 'updates'   && <UpdatesPanel />}
        {activeTab === 'apply'     && <ApplyHistoryTab />}
        {activeTab === 'snapshots' && <SnapshotsTab onChanged={() => setPendingPoll((k) => k + 1)} />}
        {activeTab === 'backup'    && <BackupTab onChanged={() => setPendingPoll((k) => k + 1)} />}
        {activeTab === 'apikeys'   && <ApiKeysTab />}
        {activeTab === 'apidocs'   && <ApiDocsTab />}
      </SectionNav>
    </div>
  )
}
