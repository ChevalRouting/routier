import { ClauseOp, ClauseValueInput, CUSTOM_OP } from '@/components/routing/ClauseEditor'
import { KVPair, newId } from '@/components/routing/shared'
import { Button, Input, Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue } from 'cheval-ui'
import { Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'

type ClauseEditorShape = {
  pairs: KVPair[]
  onChange: (v: KVPair[]) => void
  catalog: ClauseOp[]
  prefixListNames: string[]
  verb: 'match' | 'set'
}

export function ClauseEditor({
  pairs, onChange, catalog, prefixListNames, verb,
}: ClauseEditorShape) {
  const [customIds, setCustomIds] = useState<Set<number>>(new Set())
  const groups = [...new Set(catalog.map((o) => o.group))]

  const upd = (id: number, patch: Partial<KVPair>) =>
    onChange(pairs.map((p) => (p._id === id ? { ...p, ...patch } : p)))
  const add = () => onChange([...pairs, { _id: newId(), key: '', value: '' }])
  const remove = (id: number) => onChange(pairs.filter((p) => p._id !== id))

  const pickOp = (id: number, opKey: string) => {
    if (opKey === CUSTOM_OP) {
      setCustomIds((s) => new Set(s).add(id))
      upd(id, { key: '' })
      return
    }
    setCustomIds((s) => { const n = new Set(s); n.delete(id); return n })
    upd(id, { key: opKey, value: '' })
  }

  return (
    <div className="space-y-1.5">
      {pairs.map(({ _id, key, value }) => {
        const op = catalog.find((o) => o.key === key)
        const isCustom = customIds.has(_id) || (!op && key !== '')
        const selectValue = isCustom ? CUSTOM_OP : (op ? op.key : undefined)
        return (
          <div key={_id} className="flex items-start gap-1.5">
            <Select value={selectValue} onValueChange={(v) => pickOp(_id, v)}>
              <SelectTrigger className="h-7 text-xs w-40 shrink-0"><SelectValue placeholder={`${verb}…`} /></SelectTrigger>
              <SelectContent>
                {groups.map((g) => (
                  <SelectGroup key={g}>
                    <SelectLabel className="text-[10px] uppercase tracking-wide text-muted-foreground">{g}</SelectLabel>
                    {catalog.filter((o) => o.group === g).map((o) => (
                      <SelectItem key={o.key} value={o.key} className="text-xs">{o.label}</SelectItem>
                    ))}
                  </SelectGroup>
                ))}
                <SelectGroup>
                  <SelectItem value={CUSTOM_OP} className="text-xs italic">Custom…</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>

            <div className="flex-1 min-w-0 space-y-1.5">
              {isCustom ? (
                <>
                  <Input
                    value={key}
                    onChange={(e) => upd(_id, { key: e.target.value })}
                    placeholder={`raw ${verb} token`}
                    className="font-mono h-7 text-xs w-full"
                  />
                  <Input
                    value={value}
                    onChange={(e) => upd(_id, { value: e.target.value })}
                    placeholder="value"
                    className="font-mono h-7 text-xs w-full"
                  />
                </>
              ) : (
                <>
                  <ClauseValueInput op={op} value={value} onChange={(v) => upd(_id, { value: v })} prefixListNames={prefixListNames} />
                  {op && <p className="text-[11px] leading-tight text-muted-foreground px-0.5">{op.desc}</p>}
                </>
              )}
            </div>

            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive"
              onClick={() => remove(_id)}
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
        )
      })}
      <Button
        variant="ghost"
        size="sm"
        className="h-6 text-xs gap-1 text-muted-foreground px-1"
        onClick={add}
      >
        <Plus className="h-3 w-3" />Add {verb}
      </Button>
    </div>
  )
}
