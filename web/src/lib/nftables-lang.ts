import { StreamLanguage, HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { tags as t } from '@lezer/highlight'
import { EditorView, Decoration, type DecorationSet } from '@codemirror/view'
import { StateField, RangeSetBuilder } from '@codemirror/state'
import { autocompletion, type CompletionContext, type CompletionResult } from '@codemirror/autocomplete'
import { search } from '@codemirror/search'

type DocShape = { lines: number; line: (n: number) => LineShape }

type LineShape = { from: number }

const KEYWORDS = new Set([
  'table', 'chain', 'type', 'hook', 'priority', 'policy', 'define', 'include',
  'flush', 'ruleset', 'add', 'rule', 'set', 'map', 'element', 'flowtable',
  'ip', 'ip6', 'inet', 'arp', 'bridge', 'netdev',
  'filter', 'nat', 'route', 'mangle', 'raw',
  'ingress', 'prerouting', 'input', 'forward', 'output', 'postrouting',
  'iifname', 'oifname', 'iif', 'oif', 'meta', 'mark', 'l4proto', 'nfproto',
  'ct', 'state', 'status', 'ether', 'vlan', 'tcp', 'udp', 'sctp', 'dccp',
  'icmp', 'icmpv6', 'igmp', 'esp', 'ah', 'gre', 'comment',
  'saddr', 'daddr', 'sport', 'dport', 'protocol', 'nexthdr', 'th', 'flags',
  'limit', 'rate', 'over', 'burst', 'numgen', 'jhash', 'ct', 'dnat', 'snat',
  'masquerade', 'redirect', 'to', 'log', 'prefix', 'counter', 'level', 'group',
  'established', 'related', 'invalid', 'new', 'untracked', 'devices',
])

const VERDICTS = new Set([
  'accept', 'drop', 'reject', 'return', 'queue', 'continue', 'jump', 'goto', 'notrack',
])

interface Stream {
  eatSpace(): boolean
  match(p: string | RegExp, consume?: boolean): boolean | RegExpMatchArray | null
  current(): string
  skipToEnd(): void
  next(): string | void
}

const nftLanguage = StreamLanguage.define<unknown>({
  languageData: { commentTokens: { line: '#' } },
  token(stream: Stream) {
    if (stream.eatSpace()) return null
    if (stream.match('#')) { stream.skipToEnd(); return 'comment' }
    if (stream.match(/^"(?:[^"\\]|\\.)*"/)) return 'string'
    if (stream.match(/^\$[A-Za-z0-9_]+/)) return 'variableName'
    if (stream.match(/^[A-Za-z][A-Za-z0-9_-]*/)) {
      const w = stream.current()
      if (VERDICTS.has(w)) return 'atom'
      if (KEYWORDS.has(w)) return 'keyword'
      return null
    }
    if (stream.match(/^[0-9a-fA-F]+[0-9a-fA-F:./]*/)) return 'number'
    stream.next()
    return null
  },
})

const nftHighlightStyle = HighlightStyle.define([
  { tag: t.comment, color: 'hsl(var(--muted-foreground))', fontStyle: 'italic' },
  { tag: t.keyword, color: 'hsl(var(--primary))', fontWeight: '600' },
  { tag: t.atom, color: 'hsl(var(--info, var(--primary)))', fontWeight: '600' },
  { tag: t.string, color: 'hsl(var(--success, var(--primary)))' },
  { tag: t.variableName, color: 'hsl(var(--warning, var(--primary)))' },
  { tag: t.number, color: 'hsl(var(--foreground))' },
])

export const nftTheme = EditorView.theme({
  '&': { backgroundColor: 'hsl(var(--card))', color: 'hsl(var(--foreground))' },
  '.cm-content': { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace', fontSize: '12px' },
  '.cm-gutters': { backgroundColor: 'hsl(var(--card))', color: 'hsl(var(--muted-foreground))', border: 'none' },
  '.cm-activeLine': { backgroundColor: 'hsl(var(--muted) / 0.4)' },
  '.cm-activeLineGutter': { backgroundColor: 'transparent', color: 'hsl(var(--foreground))' },
  '&.cm-focused': { outline: 'none' },
  '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'hsl(var(--foreground))' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: 'hsl(var(--primary) / 0.25)',
  },
  '.cm-placeholder': { color: 'hsl(var(--muted-foreground))' },
  '.cm-matchingBracket': { backgroundColor: 'hsl(var(--primary) / 0.2)', outline: 'none' },
  '.cm-tooltip.cm-tooltip-autocomplete': {
    backgroundColor: 'hsl(var(--popover))',
    border: '1px solid hsl(var(--border))',
    borderRadius: '6px',
    boxShadow: '0 4px 12px rgb(0 0 0 / 0.15)',
  },
  '.cm-tooltip-autocomplete > ul': { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace', fontSize: '12px' },
  '.cm-tooltip-autocomplete > ul > li': { color: 'hsl(var(--popover-foreground))', padding: '2px 6px' },
  '.cm-tooltip-autocomplete > ul > li[aria-selected]': {
    backgroundColor: 'hsl(var(--primary) / 0.15)',
    color: 'hsl(var(--foreground))',
  },
  '.cm-completionDetail': { color: 'hsl(var(--muted-foreground))', fontStyle: 'normal', marginLeft: '0.75rem' },
  '.cm-completionIcon': { display: 'none' },
})

export interface NftVar {
  name: string
  value: string
}

export function nftAutocomplete(vars: NftVar[]) {
  return autocompletion({
    icons: false,
    override: [(ctx: CompletionContext): CompletionResult | null => {
      const word = ctx.matchBefore(/\$[A-Za-z0-9_]*/)
      if (!word || (word.from === word.to && !ctx.explicit)) return null
      return {
        from: word.from,
        options: vars.map((v) => ({ label: '$' + v.name, detail: v.value, type: 'variable' })),
        validFor: /^\$[A-Za-z0-9_]*$/,
      }
    }],
  })
}

export const nftExtensions = [nftLanguage, syntaxHighlighting(nftHighlightStyle), EditorView.lineWrapping, search({ top: true })]

const errorLineDeco = Decoration.line({ class: 'cm-nft-error-line' })

const errorLineTheme = EditorView.theme({
  '.cm-nft-error-line': { backgroundColor: 'hsl(var(--danger) / 0.18)' },
})

function buildErrorDeco(doc: DocShape, lines: number[]): DecorationSet {
  const builder = new RangeSetBuilder<Decoration>()
  for (const ln of [...lines].sort((a, b) => a - b)) {
    if (ln >= 1 && ln <= doc.lines) {
      builder.add(doc.line(ln).from, doc.line(ln).from, errorLineDeco)
    }
  }

  return builder.finish()
}

export function nftErrorHighlight(lines: number[]) {
  const field = StateField.define<DecorationSet>({
    create(state) {
      return buildErrorDeco(state.doc, lines)
    },
    update(deco, tr) {
      if (tr.docChanged) {
        return buildErrorDeco(tr.state.doc, lines)
      }

      return deco.map(tr.changes)
    },
    provide: (f) => EditorView.decorations.from(f),
  })

  return [field, errorLineTheme]
}
