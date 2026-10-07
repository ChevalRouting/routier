import type { TypesNftValidateResult as NftValidateResult, RenderNftVar as NftVar } from '@/api'
import { nftAutocomplete, nftErrorHighlight, nftExtensions, nftTheme } from '@/lib/nftables-lang'
import CodeMirror from '@uiw/react-codemirror'
import { Badge } from 'cheval-ui'
import { AlertCircle, CheckCircle2, EyeOff, Loader2 } from 'lucide-react'
import { useMemo } from 'react'

type OWNEDCHAINSShape = { name: string; kind: 'filter' | 'nat'; defPolicy: string }

type CodeAreaShape = {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  height?: number | string
  vars?: NftVar[]
  errorLines?: number[]
}

type RuleSummaryShape = { rule: ManagedRule; index: number }

type ValidationStripShape = { validating: boolean; result: NftValidateResult | null }

export interface RuleMatch {
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

export interface ManagedRule {
  comment?: string
  tag?: string
  disabled?: boolean
  match?: RuleMatch
  action: string
  action_to?: string
}

export interface NftChain {
  policy?: string
  rules?: string
  files?: string[]
  managed?: ManagedRule[]
}

export interface NftablesConfig {
  defines?: string
  chains?: Record<string, NftChain>
  include?: string[]
}

export const ACTIONS = ['accept', 'drop', 'reject', 'return', 'log', 'masquerade', 'dnat', 'snat', 'redirect']

export const PROTOCOLS = ['tcp', 'udp', 'icmp', 'icmpv6', 'esp', 'ah', 'gre']

export const CT_STATES = ['established,related', 'new', 'invalid', 'established', 'related', 'untracked']

export const ADDR_FAMILIES = ['ip', 'ip6']

export const ACTION_COLORS: Record<string, string> = {
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

export const OWNED_CHAINS: OWNEDCHAINSShape[] = [
  { name: 'input', kind: 'filter', defPolicy: 'drop' },
  { name: 'forward', kind: 'filter', defPolicy: 'drop' },
  { name: 'output', kind: 'filter', defPolicy: 'accept' },
  { name: 'prerouting', kind: 'nat', defPolicy: 'accept' },
  { name: 'postrouting', kind: 'nat', defPolicy: 'accept' },
]

export const EDITOR_HEIGHT = 'calc(100vh - 17rem)'

export function emptyRule(): ManagedRule {
  return { action: 'accept', match: {} }
}

export function fromApi(raw: unknown): NftablesConfig {
  const r = (raw && typeof raw === 'object') ? raw as NftablesConfig : {}
  return { defines: r.defines ?? '', chains: r.chains ?? {}, include: r.include ?? [] }
}

export function trimRuleBlock(s: string): string {
  return s.split('\n').map((l) => l.replace(/[ \t]+$/, '')).join('\n').replace(/\n+$/, '')
}

export function parseUserDefines(text: string): NftVar[] {
  const out: NftVar[] = []
  for (const raw of text.split('\n')) {
    const m = raw.trim().match(/^define\s+([A-Za-z0-9_]+)\s*=\s*(.+)$/)
    if (m) out.push({ name: m[1], value: m[2].trim() })
  }
  return out
}

export function cleanChain(ch: NftChain): NftChain | null {
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

export function toApi(cfg: NftablesConfig): NftablesConfig {
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

export function chainRuleCount(ch?: NftChain): number {
  if (!ch) return 0
  const ruleLines = (ch.rules ?? '').split('\n').filter((l) => l.trim() && !l.trim().startsWith('#')).length
  return ruleLines + (ch.managed ?? []).length + (ch.files ?? []).length
}

export function ruleSummary(rule: ManagedRule): string {
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

export function CodeArea({ value, onChange, placeholder, height = 180, vars, errorLines }: CodeAreaShape) {
  const h = typeof height === 'number' ? `${height}px` : height
  const errorKey = (errorLines ?? []).join(',')
  const extensions = useMemo(() => {
    const ext = vars && vars.length ? [...nftExtensions, nftAutocomplete(vars)] : [...nftExtensions]
    if (errorKey) {
      ext.push(nftErrorHighlight(errorKey.split(',').map(Number)))
    }

    return ext

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

export function RuleSummary({ rule, index }: RuleSummaryShape) {
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

export const DRAFT_KEY = 'routier:draft:nftables'

export function ValidationStrip({ validating, result }: ValidationStripShape) {
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
