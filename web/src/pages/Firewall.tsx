import { useState, useEffect, useMemo } from 'react'
import { api } from '@/lib/client'
import type { RenderNftVar as NftVar, TypesNftValidateResult as NftValidateResult, ConfigNftablesConfig } from '@/api'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { useDataRefresh } from '@/lib/dataVersion'
import { useTabState } from 'cheval-ui'
import { Switch } from 'cheval-ui'
import { Input } from 'cheval-ui'
import { Label } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { Separator } from 'cheval-ui'
import { SectionNav } from 'cheval-ui'
import CodeMirror from '@uiw/react-codemirror'
import { nftExtensions, nftTheme, nftAutocomplete, nftErrorHighlight } from '@/lib/nftables-lang'
import { SaveButton } from 'cheval-ui'
import { ChevronDown, ChevronRight, EyeOff, CheckCircle2, AlertCircle, Loader2 } from 'lucide-react'
import { AccordionList } from '@/components/ui/AccordionList'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from 'cheval-ui'
import { PageHeader } from 'cheval-ui'
import { Spinner } from 'cheval-ui'
import { VarPickerInput } from '@/components/VarPickerInput'

interface RuleMatch {
  protocol?: string
  iif?: string
  oif?: string
  saddr?: string
  daddr?: string
  sport?: string
  dport?: string
  ct_state?: string
  addr_family?: string
}

interface ManagedRule {
  comment?: string
  tag?: string
  disabled?: boolean
  match?: RuleMatch
  action: string
  action_to?: string
}

interface NftChain {
  policy?: string
  rules?: string
  files?: string[]
  managed?: ManagedRule[]
}

interface NftablesConfig {
  defines?: string
  chains?: Record<string, NftChain>
  include?: string[]
}

const ACTIONS = ['accept', 'drop', 'reject', 'return', 'log', 'masquerade', 'dnat', 'snat', 'redirect']
const PROTOCOLS = ['tcp', 'udp', 'icmp', 'icmpv6', 'esp', 'ah', 'gre']
const CT_STATES = ['established,related', 'new', 'invalid', 'established', 'related', 'untracked']
const ADDR_FAMILIES = ['ip', 'ip6']

const ACTION_COLORS: Record<string, string> = {
  accept:     'bg-success/15 text-success',
  drop:       'bg-danger/15 text-danger',
  reject:     'bg-danger/15 text-danger',
  masquerade: 'bg-info/15 text-info',
  dnat:       'bg-info/15 text-info',
  snat:       'bg-info/15 text-info',
  log:        'bg-warning/15 text-warning',
  return:     'bg-muted text-muted-foreground',
  redirect:   'bg-info/15 text-info',
}

const OWNED_CHAINS: { name: string; kind: 'filter' | 'nat'; defPolicy: string }[] = [
  { name: 'input', kind: 'filter', defPolicy: 'drop' },
  { name: 'forward', kind: 'filter', defPolicy: 'drop' },
  { name: 'output', kind: 'filter', defPolicy: 'accept' },
  { name: 'prerouting', kind: 'nat', defPolicy: 'accept' },
  { name: 'postrouting', kind: 'nat', defPolicy: 'accept' },
]

const EDITOR_HEIGHT = 'calc(100vh - 17rem)'

function emptyRule(): ManagedRule {
  return { action: 'accept', match: {} }
}

function fromApi(raw: unknown): NftablesConfig {
  const r = (raw && typeof raw === 'object') ? raw as NftablesConfig : {}
  return { defines: r.defines ?? '', chains: r.chains ?? {}, include: r.include ?? [] }
}

function trimRuleBlock(s: string): string {
  return s.split('\n').map((l) => l.replace(/[ \t]+$/, '')).join('\n').replace(/\n+$/, '')
}

function parseUserDefines(text: string): NftVar[] {
  const out: NftVar[] = []
  for (const raw of text.split('\n')) {
    const m = raw.trim().match(/^define\s+([A-Za-z0-9_]+)\s*=\s*(.+)$/)
    if (m) out.push({ name: m[1], value: m[2].trim() })
  }
  return out
}

function cleanChain(ch: NftChain): NftChain | null {
  const rules = trimRuleBlock(ch.rules ?? '')
  const files = (ch.files ?? []).map((f) => f.trim()).filter(Boolean)
  const managed = ch.managed ?? []
  const out: NftChain = {}
  if (ch.policy) out.policy = ch.policy
  if (rules) out.rules = rules
  if (files.length) out.files = files
  if (managed.length) out.managed = managed
  return Object.keys(out).length ? out : null
}

function toApi(cfg: NftablesConfig): NftablesConfig {
  const out: NftablesConfig = {}
  if (cfg.defines && cfg.defines.trim()) out.defines = trimRuleBlock(cfg.defines)
  const chains: Record<string, NftChain> = {}
  for (const [name, ch] of Object.entries(cfg.chains ?? {})) {
    const cleaned = cleanChain(ch)
    if (cleaned) chains[name] = cleaned
  }
  if (Object.keys(chains).length) out.chains = chains
  const include = (cfg.include ?? []).map((s) => s.trim()).filter(Boolean)
  if (include.length) out.include = include
  return out
}

function chainRuleCount(ch?: NftChain): number {
  if (!ch) return 0
  const ruleLines = (ch.rules ?? '').split('\n').filter((l) => l.trim() && !l.trim().startsWith('#')).length
  return ruleLines + (ch.managed ?? []).length + (ch.files ?? []).length
}

function ruleSummary(rule: ManagedRule): string {
  const m = rule.match
  const parts: string[] = []
  if (m) {
    if (m.ct_state) parts.push(`ct:${m.ct_state}`)
    if (m.protocol) parts.push(m.protocol)
    if (m.iif)      parts.push(`in:${m.iif}`)
    if (m.oif)      parts.push(`out:${m.oif}`)
    if (m.saddr)    parts.push(`src:${m.saddr}`)
    if (m.daddr)    parts.push(`dst:${m.daddr}`)
    if (m.sport)    parts.push(`sport:${m.sport}`)
    if (m.dport)    parts.push(`dport:${m.dport}`)
  }
  const matchStr = parts.length ? parts.join(' ') : '(match all)'
  const actionStr = rule.action_to ? `${rule.action} → ${rule.action_to}` : rule.action
  return `${matchStr}  →  ${actionStr}`
}

function CodeArea({ value, onChange, placeholder, height = 180, vars, errorLines }: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  height?: number | string
  vars?: NftVar[]
  errorLines?: number[]
}) {
  const h = typeof height === 'number' ? `${height}px` : height
  const errorKey = (errorLines ?? []).join(',')
  const extensions = useMemo(() => {
    const ext = vars && vars.length ? [...nftExtensions, nftAutocomplete(vars)] : [...nftExtensions]
    if (errorKey) {
      ext.push(nftErrorHighlight(errorKey.split(',').map(Number)))
    }

    return ext
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [vars, errorKey])
  return (
    <div className="rounded-md bg-card overflow-hidden shadow-[var(--card-shadow)]">
      <CodeMirror
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        height={h}
        theme={nftTheme}
        extensions={extensions}
        basicSetup={{
          lineNumbers: true,
          foldGutter: false,
          autocompletion: false,
          highlightActiveLine: true,
          highlightSelectionMatches: true,
          searchKeymap: true,
        }}
      />
    </div>
  )
}

function RuleSummary({ rule, index }: { rule: ManagedRule; index: number }) {
  const actionColor = ACTION_COLORS[rule.action] ?? 'bg-muted text-muted-foreground'
  return (
    <div className={`flex min-w-0 flex-1 items-center gap-2 ${rule.disabled ? 'opacity-50' : ''}`}>
      <span className="shrink-0 text-xs text-muted-foreground">#{index + 1}</span>
      <span className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">
        {ruleSummary(rule)}
      </span>
      {rule.tag && <Badge variant="secondary" className="shrink-0 text-[10px]">{rule.tag}</Badge>}
      <span className={`shrink-0 rounded px-1.5 py-0.5 text-[11px] font-medium ${actionColor}`}>
        {rule.action}
      </span>
      {rule.disabled && <EyeOff className="h-3 w-3 shrink-0 text-muted-foreground" />}
    </div>
  )
}

function RuleBody({ rule, onChange, vars }: { rule: ManagedRule; onChange: (r: ManagedRule) => void; vars: string[] }) {
  const m = rule.match ?? {}

  const setMatch = (patch: Partial<RuleMatch>) =>
    onChange({ ...rule, match: { ...m, ...patch } })

  return (
    <>
          <div className="grid grid-cols-3 gap-2">
            <div className="space-y-1.5">
              <Label className="text-[11px]">Protocol</Label>
              <Select value={m.protocol ?? '_any'} onValueChange={(v) => setMatch({ protocol: v === '_any' ? undefined : v })}>
                <SelectTrigger className="h-8 text-xs font-mono">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="_any">any</SelectItem>
                  {PROTOCOLS.map((p) => <SelectItem key={p} value={p} className="font-mono">{p}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Input interface</Label>
              <VarPickerInput value={m.iif ?? ''} onChange={(v) => setMatch({ iif: v || undefined })}
                vars={vars.filter((v) => v.endsWith('_interfaces') || v === 'interfaces' || v === 'tunnels' || v === 'wireguard')}
                placeholder="$lan_interfaces" mono />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Output interface</Label>
              <VarPickerInput value={m.oif ?? ''} onChange={(v) => setMatch({ oif: v || undefined })}
                vars={vars.filter((v) => v.endsWith('_interfaces') || v === 'interfaces' || v === 'tunnels' || v === 'wireguard')}
                placeholder="$wan_interfaces" mono />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Source addr</Label>
              <VarPickerInput value={m.saddr ?? ''} onChange={(v) => setMatch({ saddr: v || undefined })}
                vars={vars.filter((v) => v.includes('_address') || v.includes('_network'))}
                placeholder="$lan_network" mono />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Dest addr</Label>
              <VarPickerInput value={m.daddr ?? ''} onChange={(v) => setMatch({ daddr: v || undefined })}
                vars={vars.filter((v) => v.includes('_address') || v.includes('_network'))}
                placeholder="192.168.0.0/24" mono />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">
                Addr family
                {(m.saddr || m.daddr) && <span className="text-destructive ml-0.5">*</span>}
              </Label>
              <Select value={m.addr_family ?? '_none'} onValueChange={(v) => setMatch({ addr_family: v === '_none' ? undefined : v })}>
                <SelectTrigger className={`h-8 text-xs ${(m.saddr || m.daddr) && !m.addr_family ? 'border-destructive focus:ring-destructive' : ''}`}>
                  <SelectValue placeholder="- required for saddr/daddr -" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="_none">- required for saddr/daddr -</SelectItem>
                  {ADDR_FAMILIES.map((f) => <SelectItem key={f} value={f}>{f}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Source port</Label>
              <Input value={m.sport ?? ''} onChange={(e) => setMatch({ sport: e.target.value || undefined })}
                placeholder="1024-65535" className="font-mono text-xs h-8" />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">Dest port</Label>
              <Input value={m.dport ?? ''} onChange={(e) => setMatch({ dport: e.target.value || undefined })}
                placeholder="80,443" className="font-mono text-xs h-8" />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px]">CT state</Label>
              <Select value={m.ct_state ?? '_any'} onValueChange={(v) => setMatch({ ct_state: v === '_any' ? undefined : v })}>
                <SelectTrigger className="h-8 text-xs font-mono">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="_any">any</SelectItem>
                  {CT_STATES.map((s) => <SelectItem key={s} value={s} className="font-mono">{s}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
          </div>

          <Separator />

          <div className="flex flex-wrap items-end gap-3">
            <div className="space-y-1.5">
              <Label className="text-[11px]">Action</Label>
              <Select value={rule.action} onValueChange={(v) => onChange({ ...rule, action: v })}>
                <SelectTrigger className="h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ACTIONS.map((a) => <SelectItem key={a} value={a}>{a}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            {(rule.action === 'dnat' || rule.action === 'snat') && (
              <div className="space-y-1.5 flex-1 min-w-32">
                <Label className="text-[11px]">Target address</Label>
                <Input value={rule.action_to ?? ''}
                  onChange={(e) => onChange({ ...rule, action_to: e.target.value || undefined })}
                  placeholder="10.0.0.1" className="font-mono text-xs h-8" />
              </div>
            )}
            <div className="space-y-1.5 flex-1 min-w-32">
              <Label className="text-[11px]">Comment</Label>
              <Input value={rule.comment ?? ''}
                onChange={(e) => onChange({ ...rule, comment: e.target.value || undefined })}
                placeholder="optional description" className="text-xs h-8" />
            </div>
            <div className="flex items-center justify-between gap-2 mb-0.5">
              <span className="text-xs text-muted-foreground">Disabled</span>
              <Switch checked={rule.disabled ?? false}
                onCheckedChange={(v) => onChange({ ...rule, disabled: v || undefined })} />
            </div>
          </div>
    </>
  )
}

function ChainPane({ meta, chain, onChange, varNames, acVars, errorLines }: {
  meta: typeof OWNED_CHAINS[number]
  chain: NftChain
  onChange: (c: NftChain) => void
  varNames: string[]
  acVars: NftVar[]
  errorLines?: number[]
}) {
  const [showManaged, setShowManaged] = useState((chain.managed ?? []).length > 0)
  const rulesText = chain.rules ?? ''
  const managed = chain.managed ?? []
  const setManaged = (rules: ManagedRule[]) => onChange({ ...chain, managed: rules })

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <span className="font-mono font-semibold text-sm">{meta.name}</span>
        <Badge variant="secondary" className="text-[10px]">{meta.kind}</Badge>
        <div className="ml-auto flex items-center gap-2">
          <span className="text-xs text-muted-foreground">policy</span>
          <Select value={chain.policy ?? '_default'}
            onValueChange={(v) => onChange({ ...chain, policy: v === '_default' ? undefined : v })}>
            <SelectTrigger className="h-7 text-xs w-36"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="_default">default ({meta.defPolicy})</SelectItem>
              <SelectItem value="accept">accept</SelectItem>
              <SelectItem value="drop">drop</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <CodeArea
        value={rulesText}
        onChange={(v) => onChange({ ...chain, rules: v })}
        placeholder={'ip saddr $lan_network tcp dport 22 accept\nlog prefix "in: " counter'}
        height={EDITOR_HEIGHT}
        vars={acVars}
        errorLines={errorLines} />
      <p className="text-[11px] text-muted-foreground">
        Appended after routier&apos;s auto rules and the structured rules below. Type <span className="font-mono">$</span> to autocomplete variables like $wan_interfaces, $me.
      </p>

      <div>
        <button type="button" onClick={() => setShowManaged(!showManaged)}
          className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
          {showManaged ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
          Structured rules ({managed.length})
        </button>
        {showManaged && (
          <div className="mt-2">
            <AccordionList
              items={managed}
              addLabel="Add structured rule"
              onAdd={() => setManaged([...managed, emptyRule()])}
              onRemove={(r) => setManaged(managed.filter((x) => x !== r))}
              emptyTitle="No structured rules"
              emptyMessage="Add a structured rule, or write nftables above."
              renderSummary={(r) => <RuleSummary rule={r} index={managed.indexOf(r)} />}
              renderBody={(r) => (
                <RuleBody rule={r} vars={varNames} onChange={(nr) => setManaged(managed.map((x) => (x === r ? nr : x)))} />
              )}
            />
          </div>
        )}
      </div>
    </div>
  )
}

const DRAFT_KEY = 'routier:draft:nftables'

export default function Firewall() {
  const { data, isLoading } = useFetch(() => api.apiConfigSectionGet({ section: 'nftables' }))
  const { data: rawVars } = useFetch<NftVar[]>(() => api.apiConfigNftablesVarsGet())
  const [cfg, setCfg] = useState<NftablesConfig>({})
  const [initialized, setInitialized] = useState(false)
  useDataRefresh(() => { try { localStorage.removeItem(DRAFT_KEY) } catch { /* ignore */ } setInitialized(false) })
  const { isDirty, markDirty, save, saving } = usePageSave('nftables')
  const [section, setSection] = useTabState<string>('firewall', 'input')
  const [validation, setValidation] = useState<NftValidateResult | null>(null)
  const [validating, setValidating] = useState(false)

  const vars = useMemo(() => rawVars ?? [], [rawVars])
  const varNames = useMemo(() => vars.map((v) => v.name), [vars])

  const acVars = useMemo<NftVar[]>(() => {
    const userDefines = parseUserDefines(cfg.defines ?? '')
    const seen = new Set(vars.map((v) => v.name))
    return [...vars, ...userDefines.filter((v) => !seen.has(v.name))]
      .sort((a, b) => a.name.localeCompare(b.name))
  }, [vars, cfg.defines])

  useEffect(() => {
    if (data !== null && !initialized) {
      let next = fromApi(data)
      try {
        const raw = localStorage.getItem(DRAFT_KEY)
        if (raw) { next = JSON.parse(raw) as NftablesConfig; markDirty() }
      } catch { /* ignore */ }
      setCfg(next)
      setInitialized(true)
    }
  }, [data, initialized, markDirty])

  const validatePayload = useMemo(() => (initialized ? JSON.stringify(toApi(cfg)) : ''), [cfg, initialized])
  useEffect(() => {
    if (!initialized) return
    const ctrl = new AbortController()
    const id = setTimeout(async () => {
      setValidating(true)
      try {
        setValidation(await api.apiConfigNftablesValidatePost(
          { ConfigNftablesConfig: JSON.parse(validatePayload) as ConfigNftablesConfig },
          { signal: ctrl.signal },
        ))
      } catch { /* aborted or transient; keep last result */ } finally {
        setValidating(false)
      }
    }, 750)
    return () => { clearTimeout(id); ctrl.abort() }
  }, [validatePayload, initialized])

  const update = (next: NftablesConfig) => {
    setCfg(next)
    markDirty()
    try { localStorage.setItem(DRAFT_KEY, JSON.stringify(next)) } catch { /* ignore */ }
  }
  const setChain = (name: string, ch: NftChain) =>
    update({ ...cfg, chains: { ...cfg.chains, [name]: ch } })

  const handleSave = async () => {
    await save(toApi(cfg))
    try { localStorage.removeItem(DRAFT_KEY) } catch { /* ignore */ }
  }

  if (isLoading) return <Spinner />

  const items = [
    { key: 'defines', label: 'Defines' },
    ...OWNED_CHAINS.map((c) => {
      const n = chainRuleCount(cfg.chains?.[c.name])
      return { key: c.name, label: c.name.charAt(0).toUpperCase() + c.name.slice(1), badge: n > 0 ? n : undefined }
    }),
  ]

  const chainMeta = OWNED_CHAINS.find((c) => c.name === section)
  const errorLinesFor = (sec: string) =>
    (validation?.errors ?? []).filter((e) => e.section === sec && e.line > 0).map((e) => e.line)

  return (
    <div className="space-y-6">
      <PageHeader title="Firewall"
        description="Routier owns table inet routier; add your rules into its chains"
        action={<SaveButton isDirty={isDirty} saving={saving} onClick={handleSave} />} />

      <SectionNav items={items} active={section} onChange={setSection}>
        {section === 'defines' ? (
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <span className="font-mono font-semibold text-sm">defines</span>
              <Badge variant="secondary" className="text-[10px]">variables</Badge>
            </div>
            <CodeArea
              value={cfg.defines ?? ''}
              onChange={(v) => update({ ...cfg, defines: v })}
              placeholder={'define my_set = { 10.0.0.0/8, 2001:db8::/48 }'}
              height={EDITOR_HEIGHT}
              vars={acVars}
              errorLines={errorLinesFor('defines')} />
            <p className="text-[11px] text-muted-foreground">
              Raw nft `define` lines, emitted before the ruleset so your rules can reference them.
            </p>
          </div>
        ) : chainMeta ? (
          <ChainPane
            meta={chainMeta}
            chain={cfg.chains?.[section] ?? {}}
            onChange={(ch) => setChain(section, ch)}
            varNames={varNames}
            acVars={acVars}
            errorLines={errorLinesFor(chainMeta.name)} />
        ) : null}
      </SectionNav>

      <ValidationStrip validating={validating} result={validation} />
    </div>
  )
}

function ValidationStrip({ validating, result }: { validating: boolean; result: NftValidateResult | null }) {
  if (validating && !result) {
    return (
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />Checking ruleset…
      </div>
    )
  }

  if (!result) return null

  const errors = result.errors ?? []
  if (result.ok || errors.length === 0) {
    return (
      <div className="flex items-center gap-2 text-xs text-success">
        <CheckCircle2 className="h-3.5 w-3.5" />Ruleset is valid{validating ? ' (rechecking…)' : ''}
      </div>
    )
  }

  return (
    <div className="rounded-md border border-danger/40 bg-danger/10 p-3 space-y-1.5">
      <div className="flex items-center gap-2 text-xs font-medium text-danger">
        <AlertCircle className="h-3.5 w-3.5" />nft validation failed
      </div>
      <ul className="space-y-0.5">
        {errors.map((e, i) => (
          <li key={i} className="font-mono text-[11px] text-danger/90">
            {e.section ? `${e.section} ` : ''}{e.line > 0 ? `line ${e.line}: ` : ''}{e.message}
          </li>
        ))}
      </ul>
    </div>
  )
}
