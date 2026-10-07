import { Badge, Button, EmptyState, Input, PreferencesColumns, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Sheet } from 'cheval-ui'
import { Plus, Radio, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { InterfaceSheet } from './RadvdTabParts/InterfaceSheet'

type RouteRowShape = {
  route: RADVDRoute
  onChange: (r: RADVDRoute) => void
  onDelete: () => void
}

type RadvdTabShape = {
  radvd: RADVDConfig | null
  onChange: (v: RADVDConfig | null) => void
  onDirty: () => void
  ifaceNames: string[]
}

export interface RADVDPrefix {
  prefix: string
  adv_on_link?: boolean
  adv_autonomous?: boolean
  adv_router_addr?: boolean
  adv_valid_lifetime?: string
  adv_preferred_lifetime?: string
}

export interface RADVDRDNSS {
  servers?: string[]
  lifetime?: number
}

export interface RADVDRoute {
  prefix: string
  lifetime?: number
  preference?: string
}

export interface RADVDInterface {
  adv_send_advert?: boolean
  min_rtr_adv_interval?: number
  max_rtr_adv_interval?: number
  adv_managed_flag?: boolean
  adv_other_config_flag?: boolean
  adv_default_lifetime?: number
  adv_default_preference?: string
  adv_link_mtu?: number
  adv_ra_src_address?: string[]
  prefixes?: RADVDPrefix[]
  rdnss?: RADVDRDNSS
  routes?: RADVDRoute[]
}

export interface RADVDConfig {
  interfaces?: Record<string, RADVDInterface>
}

export const PREFERENCES = ['low', 'medium', 'high']

export function numOrUndef(v: string): number | undefined {
  const n = parseInt(v, 10)
  return isNaN(n) || v === '' ? undefined : n
}

export function RouteRow({
  route,
  onChange,
  onDelete,
}: RouteRowShape) {
  return (
    <div className="flex items-center gap-2">
      <Input
        value={route.prefix}
        onChange={(e) => onChange({ ...route, prefix: e.target.value })}
        placeholder="2001:db8::/48"
        className="font-mono text-xs h-8 flex-1"
      />
      <Select value={route.preference ?? '_none'} onValueChange={(v) => onChange({ ...route, preference: v === '_none' ? undefined : v })}>
        <SelectTrigger className="h-8 text-xs w-28">
          <SelectValue placeholder="preference" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="_none">preference</SelectItem>
          {PREFERENCES.map((p) => <SelectItem key={p} value={p}>{p}</SelectItem>)}
        </SelectContent>
      </Select>
      <Input
        value={route.lifetime ?? ''}
        onChange={(e) => onChange({ ...route, lifetime: numOrUndef(e.target.value) })}
        placeholder="lifetime"
        className="font-mono text-xs h-8 w-24"
      />
      <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0 hover:text-destructive" onClick={onDelete}>
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

export const NEW_KEY = '__new__'

export default function RadvdTab({
  radvd,
  onChange,
  onDirty,
  ifaceNames,
}: RadvdTabShape) {
  const [openIface, setOpenIface] = useState<string | null>(null)
  const [pendingName, setPendingName] = useState('')
  const [draftIface, setDraftIface] = useState<RADVDInterface | null>(null)

  const ifaces = radvd?.interfaces ?? {}

  const updateAll = (next: Record<string, RADVDInterface>) => {
    onChange(Object.keys(next).length ? { interfaces: next } : null)
    onDirty()
  }

  const deleteIface = (name: string) => {
    const next = { ...ifaces }
    delete next[name]
    updateAll(next)
    if (openIface === name) setOpenIface(null)
  }

  const handleAddNew = () => {
    setDraftIface({ adv_send_advert: true })
    setPendingName('')
    setOpenIface(NEW_KEY)
  }

  const handleCommitNew = () => {
    const n = pendingName.trim()
    if (!n) { toast.error('Please enter an interface name'); return }
    if (n in ifaces) { toast.error(`Interface "${n}" already exists`); return }
    updateAll({ ...ifaces, [n]: draftIface ?? { adv_send_advert: true } })
    setDraftIface(null)
    setOpenIface(null)
  }

  const handleCloseSheet = () => { setOpenIface(null); setDraftIface(null) }

  const renameIface = (oldName: string, newName: string) => {
    setPendingName(newName)
    const trimmed = newName.trim()
    if (!trimmed || trimmed === oldName || (trimmed in ifaces)) return

    const next: Record<string, RADVDInterface> = {}
    for (const [k, v] of Object.entries(ifaces)) {
      next[k === oldName ? trimmed : k] = v
    }

    updateAll(next)
    setOpenIface(trimmed)
  }

  const openIfaceData = openIface === NEW_KEY ? draftIface : (openIface ? ifaces[openIface] : null)
  const sheetTitle = openIface === NEW_KEY ? 'Add Interface' : (pendingName || openIface || 'Interface')

  const rows = Object.entries(ifaces)

  return (
    <div className="space-y-4">
      {rows.length === 0 ? (
        <EmptyState
          className="max-w-2xl mx-auto"
          icon={<Radio />}
          title="No RADVD interfaces"
          message="Add an interface to start advertising IPv6 prefixes."
          action={
            <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
              <Plus className="h-3.5 w-3.5" />Add interface
            </Button>
          }
        />
      ) : (
        <PreferencesColumns
          title="Interfaces"
          header={
            <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
              <Plus className="h-4 w-4" />Add
            </Button>
          }
        >
          {rows.map(([name, iface]) => {
            const prefixCount = iface.prefixes?.length ?? 0
            return (
              <div
                key={name}
                onClick={() => { setPendingName(name); setOpenIface(name) }}
                className="flex items-center gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors"
              >
              <span className="font-mono font-semibold text-sm w-32 shrink-0 truncate">{name}</span>
              <div className="flex flex-wrap gap-1 flex-1 min-w-0">
                <Badge variant={iface.adv_send_advert ? 'default' : 'secondary'} className="text-xs">
                  {iface.adv_send_advert ? 'RA on' : 'RA off'}
                </Badge>
                {prefixCount > 0 && (
                  <Badge variant="outline" className="text-xs font-mono">
                    {prefixCount} prefix{prefixCount !== 1 ? 'es' : ''}
                  </Badge>
                )}
                {iface.rdnss?.servers?.length ? (
                  <Badge variant="outline" className="text-xs">RDNSS</Badge>
                ) : null}
                {(iface.routes?.length ?? 0) > 0 && (
                  <Badge variant="outline" className="text-xs">{iface.routes!.length} route{iface.routes!.length !== 1 ? 's' : ''}</Badge>
                )}
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={(e) => { e.stopPropagation(); deleteIface(name) }}
                className="h-7 w-7 p-0 hover:text-destructive shrink-0"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
              </div>
            )
          })}
        </PreferencesColumns>
      )}

      <Sheet open={openIface !== null} onClose={handleCloseSheet} title={sheetTitle}>
        {openIface !== null && openIfaceData !== null && (
          <InterfaceSheet
            name={pendingName}
            iface={openIfaceData!}
            ifaceNames={ifaceNames}
            onChangeName={openIface === NEW_KEY ? setPendingName : (n) => renameIface(openIface!, n)}
            onChange={openIface === NEW_KEY ? setDraftIface : (u) => updateAll({ ...ifaces, [openIface!]: u })}
            onClose={handleCloseSheet}
            onAdd={openIface === NEW_KEY ? handleCommitNew : undefined}
          />
        )}
      </Sheet>
    </div>
  )
}

export { InterfaceSheet } from './RadvdTabParts/InterfaceSheet'
export { PrefixRow } from './RadvdTabParts/PrefixRow'
