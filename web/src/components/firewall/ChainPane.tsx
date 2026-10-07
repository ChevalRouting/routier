import type { RenderNftVar as NftVar } from '@/api'
import { RuleBody } from '@/components/firewall/RuleBody'
import { CodeArea, EDITOR_HEIGHT, emptyRule, ManagedRule, NftChain, OWNED_CHAINS, RuleSummary } from '@/components/firewall/shared'
import { AccordionList } from '@/components/ui/AccordionList'
import { Badge, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'cheval-ui'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useState } from 'react'

type ChainPaneShape = {
  meta: typeof OWNED_CHAINS[number]
  chain: NftChain
  onChange: (c: NftChain) => void
  varNames: string[]
  acVars: NftVar[]
  errorLines?: number[]
}

export function ChainPane({ meta, chain, onChange, varNames, acVars, errorLines }: ChainPaneShape) {
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
