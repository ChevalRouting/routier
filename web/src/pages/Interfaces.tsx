import type { TypesSystemNic as SystemNic } from '@/api'
import { IfaceCard } from '@/components/interfaces/IfaceCard'
import { IfaceForm } from '@/components/interfaces/IfaceForm'
import { emptyIface, Iface, IfaceMap, IfaceRow, ifaceToTunnel, matchNic, NEW_KEY, TunnelMap, tunnelToIface } from '@/components/interfaces/shared'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { Button, EmptyState, PageHeader, Pagination, ReloadButton, SaveButton, Segmented, Sheet, Spinner, Table, TableBody, TableHead, TableHeader, TableRow, usePagination } from 'cheval-ui'
import {
  Network,
  Plus
} from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

type InterfacesShape4 = { table: number }

type InterfacesShape3 = { table: number }

type InterfacesShape2 = { vrrp?: VrrpShape2[] }

type VrrpShape2 = { interface: string }

type InterfacesShape = { vrrp?: VrrpShape[] }

type VrrpShape = { interface: string }

export default function Interfaces() {
  const { data, isLoading, reload } = useFetch<IfaceMap>(() => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<IfaceMap>)
  const { data: tunData, reload: reloadTun } = useFetch<TunnelMap>(() => api.apiConfigSectionGet({ section: 'tunnels' }) as Promise<TunnelMap>)
  const { data: vrfsData } = useFetch<Record<string, InterfacesShape4>>(
    () => api.apiConfigSectionGet({ section: 'vrfs' }) as Promise<Record<string, InterfacesShape3>>
  )
  const { data: nics } = useFetch<SystemNic[]>(() => api.apiSystemNicsGet())
  const { data: haData } = useFetch<InterfacesShape2>(
    () => api.apiConfigSectionGet({ section: 'ha' }) as Promise<InterfacesShape>
  )
  const vrrpIfaces = new Set((haData?.vrrp ?? []).map((v) => v.interface))
  const nicFor = (iface: Iface, name: string) => matchNic(nics ?? [], iface, name)
  const vrfNames = Object.keys(vrfsData ?? {})
  const [entries, setEntries] = useState<IfaceMap | null>(null)
  const [newName, setNewName] = useState('')
  const [view, setView] = useState<'cards' | 'list'>(() => (localStorage.getItem('interfaces-view') === 'list' ? 'list' : 'cards'))
  const [openSheet, setOpenSheet] = useState<string | null>(null)
  const [formDraft, setFormDraft] = useState<Iface | null>(null)
  const [pendingName, setPendingName] = useState<string | null>(null)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('interfaces')
  const tun = usePageSave('tunnels')

  useDataRefresh(() => { setEntries(null); reset(); tun.reset() })

  const merged: IfaceMap = {
    ...((data as IfaceMap | null) ?? {}),
    ...Object.fromEntries(Object.entries(tunData ?? {}).map(([n, t]) => [n, tunnelToIface(t)])),
  }
  const current: IfaceMap = entries ?? merged

  const saveAll = async () => {
    const ifaceMap: IfaceMap = {}
    const tunMap: TunnelMap = {}
    for (const [name, iface] of Object.entries(current)) {
      if (iface.type === 'tunnel') {
        tunMap[name] = ifaceToTunnel(iface)
      } else {
        ifaceMap[name] = iface
      }
    }
    if (!await tun.save(tunMap)) return
    await save(ifaceMap)
  }

  const resetAll = () => { setEntries(null); reset(); tun.reset() }
  const reloadAll = () => { resetAll(); reload(true); reloadTun(true) }

  const update = (name: string, iface: Iface) => {
    setEntries({ ...current, [name]: iface })
    markDirty()
  }

  const deleteEntry = (name: string) => {
    const next = { ...current }
    delete next[name]
    setEntries(next)
    if (openSheet === name) setOpenSheet(null)
    markDirty()
  }

  const handleAddNew = () => { setFormDraft(emptyIface()); setNewName(''); setOpenSheet(NEW_KEY) }

  const handleCommitNew = () => {
    const n = newName.trim()
    if (!n) { toast.error('Please enter an interface name'); return }
    if (current[n]) { toast.error(`Interface "${n}" already exists`); return }
    setEntries({ ...current, [n]: formDraft ?? emptyIface() })
    markDirty()
    setOpenSheet(null)
    setFormDraft(null)
  }

  const handleCloseSheet = () => {
    if (openSheet && openSheet !== NEW_KEY && pendingName !== null && pendingName.trim() !== openSheet) {
      const n = pendingName.trim()
      if (n && !current[n]) {
        const next: IfaceMap = {}
        for (const [k, v] of Object.entries(current)) {
          next[k === openSheet ? n : k] = v
        }
        setEntries(next)
        markDirty()
      }
    }
    setOpenSheet(null)
    setFormDraft(null)
    setPendingName(null)
  }

  const names = Object.keys(current)
  const parentNames = names.filter((n) => current[n].type !== 'tunnel' && current[n].type !== 'vlan')
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(names, 12)

  const setViewMode = (v: 'cards' | 'list') => { localStorage.setItem('interfaces-view', v); setView(v) }

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      <PageHeader title="Interfaces" description="Manage network interface configuration" action={
        <div className="flex items-center gap-2">
          <SaveButton isDirty={isDirty} saving={saving || tun.saving} onClick={saveAll} onCancel={reloadAll} />
          <ReloadButton onClick={reloadAll} />
        </div>
      } />

      {names.length === 0 ? (
        <EmptyState
          className="max-w-2xl mx-auto"
          icon={<Network />}
          title="No interfaces"
          message="Define an interface to assign addresses, VLANs and VRRP."
          action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add interface</Button>}
        />
      ) : (
        <div className="space-y-3">
          <div className="flex items-center justify-between px-1">
            <div className="text-sm font-medium">Interfaces</div>
            <div className="flex items-center gap-2">
              <Segmented
                value={view}
                onChange={setViewMode}
                className="text-xs"
                options={[{ value: 'cards', label: 'Cards' }, { value: 'list', label: 'List' }]}
              />
              <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
                <Plus className="h-4 w-4" />Add
              </Button>
            </div>
          </div>
          {view === 'cards' ? (
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              {pageItems.map((name) => (
                <IfaceCard
                  key={name}
                  name={name}
                  iface={current[name]}
                  nic={nicFor(current[name], name)}
                  hasVrrp={vrrpIfaces.has(name)}
                  onEdit={() => { setOpenSheet(name); setPendingName(name) }}
                  onDelete={() => deleteEntry(name)}
                />
              ))}
            </div>
          ) : (
            <div className="rounded-xl bg-card shadow-[var(--card-shadow)] overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Device</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Addresses</TableHead>
                    <TableHead>VRF</TableHead>
                    <TableHead className="w-[1%]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pageItems.map((name) => (
                    <IfaceRow
                      key={name}
                      name={name}
                      iface={current[name]}
                      nic={nicFor(current[name], name)}
                      onEdit={() => { setOpenSheet(name); setPendingName(name) }}
                      onDelete={() => deleteEntry(name)}
                    />
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      )}

      <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="interfaces" />

      <Sheet
        open={!!openSheet}
        onClose={handleCloseSheet}
        title={openSheet === NEW_KEY ? 'New Interface' : (pendingName ?? openSheet ?? '')}
        className="max-w-2xl"
      >
        {openSheet && (openSheet === NEW_KEY ? formDraft : current[openSheet]) && (
          <IfaceForm
            name={openSheet === NEW_KEY ? newName : (pendingName ?? openSheet ?? '')}
            iface={openSheet === NEW_KEY ? formDraft! : current[openSheet]}
            vrfNames={vrfNames}
            parents={parentNames}
            nics={nics ?? []}
            onChange={openSheet === NEW_KEY ? setFormDraft : (u) => update(openSheet, u)}
            onNameChange={openSheet === NEW_KEY ? setNewName : setPendingName}
            onAdd={openSheet === NEW_KEY ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
          />
        )}
      </Sheet>
    </div>
  )
}
