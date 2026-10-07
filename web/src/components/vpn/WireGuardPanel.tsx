import { WgIface, WgMap, emptyWg } from '@/components/wireguard/shared'
import { WgForm } from '@/components/wireguard/WgForm'
import { WgRow } from '@/components/wireguard/WgRow'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { Button, EmptyState, Pagination, PreferencesColumns, ReloadButton, SaveButton, Sheet, Spinner, usePagination } from 'cheval-ui'
import { Lock, Plus } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

type WireGuardPanelShape = { onActionChange?: (a: React.ReactNode) => void }

const NEW_KEY = '__new__'

export function WireGuardPanel({ onActionChange }: WireGuardPanelShape) {
  const { data, isLoading, reload } = useFetch<WgMap>(() => api.apiConfigSectionGet({ section: 'wireguard' }) as Promise<WgMap>)
  const [entries, setEntries] = useState<WgMap | null>(null)
  const [newName, setNewName] = useState('')
  const [openSheet, setOpenSheet] = useState<string | null>(null)
  const [formDraft, setFormDraft] = useState<WgIface | null>(null)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('wireguard')

  useDataRefresh(() => { setEntries(null); reset() })

  const current: WgMap = entries ?? (data as WgMap | null) ?? {}

  useEffect(() => {
    const handleClick = () => { setEntries(null); reset(); reload(true) }

    const handleCancel = () => { setEntries(null); reset(); reload(true) }

    onActionChange?.(
      <div className="flex items-center gap-2">
        <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(current)} onCancel={handleCancel} />
        <ReloadButton onClick={handleClick} />
      </div>
    )
    return () => onActionChange?.(null)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isDirty, saving, entries, data, onActionChange, save, reset, reload])

  const update = (name: string, iface: WgIface) => { setEntries({ ...current, [name]: iface }); markDirty() }

  const deleteEntry = (name: string) => {
    const next = { ...current }; delete next[name]; setEntries(next)
    if (openSheet === name) setOpenSheet(null)
    markDirty()
  }

  const handleAddNew = () => { setFormDraft(emptyWg()); setNewName(''); setOpenSheet(NEW_KEY) }

  const handleCommitNew = () => {
    const n = newName.trim()
    if (!n) { toast.error('Please enter an interface name'); return }
    if (current[n]) { toast.error(`Interface "${n}" already exists`); return }
    setEntries({ ...current, [n]: formDraft ?? emptyWg() }); markDirty()
    setOpenSheet(null); setFormDraft(null)
  }

  const handleCloseSheet = () => { setOpenSheet(null); setFormDraft(null) }

  const names = Object.keys(current)
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(names, 12)

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      {names.length === 0 ? (
        <EmptyState
          className="max-w-2xl mx-auto"
          icon={<Lock />}
          title="No WireGuard interfaces"
          message="Add a WireGuard interface to define peers and encrypted tunnels."
          action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add interface</Button>}
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
          {pageItems.map((name) => (
            <WgRow
              key={name}
              name={name}
              iface={current[name]}
              onEdit={() => setOpenSheet(name)}
              onDelete={() => deleteEntry(name)}
            />
          ))}
        </PreferencesColumns>
      )}

      <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="interfaces" />

      <Sheet
        open={!!openSheet}
        onClose={handleCloseSheet}
        title={openSheet === NEW_KEY ? 'New WireGuard Interface' : (openSheet ?? '')}
        className="max-w-2xl"
      >
        {openSheet && (openSheet === NEW_KEY ? formDraft : current[openSheet]) && (
          <WgForm
            key={openSheet}
            name={openSheet === NEW_KEY ? newName : openSheet}
            iface={openSheet === NEW_KEY ? formDraft! : current[openSheet]}
            onChange={openSheet === NEW_KEY ? setFormDraft : (u) => update(openSheet, u)}
            onNameChange={openSheet === NEW_KEY ? setNewName : undefined}
            onAdd={openSheet === NEW_KEY ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
          />
        )}
      </Sheet>
    </div>
  )
}
