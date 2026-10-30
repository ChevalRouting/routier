import type { RenderNftVar as NftVar } from '@/api'
import { RuleBody } from '@/components/firewall/RuleBody'
import { CodeArea, EDITOR_HEIGHT, emptyRule, ManagedRule, NftChain, OWNED_CHAINS, RuleSummary } from '@/components/firewall/shared'
import { AccordionList } from '@/components/ui/AccordionList'
import { Badge, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Button } from 'cheval-ui'
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
        <Badge variant="secondary" className="text-xs">{meta.kind}</Badge>
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
      <p className="text-xs text-muted-foreground">
        Appended to <code>table inet routier</code> after the automatic and structured rules
      </p>

      <div>
        <Button variant="ghost" size="sm" aria-expanded={showManaged} onClick={() => setShowManaged(!showManaged)} className="gap-1.5">
          {showManaged ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          Structured Rules ({managed.length})
        </Button>
        {showManaged && (
          <div className="mt-2">
            <AccordionList
              items={managed}
              addLabel="Add Structured Rule"
              onAdd={() => setManaged([...managed, emptyRule()])}
              onRemove={(r) => setManaged(managed.filter((x) => x !== r))}
              emptyTitle="No Structured Rules"
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
