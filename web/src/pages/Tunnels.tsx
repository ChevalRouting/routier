import { useState } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { useDataRefresh } from '@/lib/dataVersion'
import { usePageSave } from '@/lib/usePageSave'
import SaveButton from '@/components/SaveButton'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Sheet } from '@/components/ui/sheet'
import TagInput from '@/components/TagInput'
import { Plus, Trash2, GitBranch } from 'lucide-react'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import { PageHeader } from '@/components/PageHeader'
import { PreferencesGroup, PreferencesColumns, EntryRow, ComboRow } from '@/components/Preferences'
import { EmptyState } from '@/components/EmptyState'
import { Pagination, usePagination } from '@/components/Pagination'
import { ReloadButton } from '@/components/ReloadButton'
import { checkCIDR, checkMTU } from '@/lib/validate'
import { Spinner } from '@/components/Spinner'

type TunnelMode = 'sit' | 'gre' | 'ipip' | 'ip6tnl' | 'ip6ip6' | 'ip6gre'

interface Tunnel {
  mode: TunnelMode
  local: string
  remote: string
  ttl: number
  addresses: string[]
  mtu: number
}

type TunnelMap = Record<string, Tunnel>

const TUNNEL_MODES: TunnelMode[] = ['sit', 'gre', 'ipip', 'ip6tnl', 'ip6ip6', 'ip6gre']

function emptyTunnel(): Tunnel {
  return { mode: 'gre', local: '', remote: '', ttl: 0, addresses: [], mtu: 0 }
}

interface TunnelFormProps {
  tunnel: Tunnel
  onChange: (updated: Tunnel) => void
  name?: string
  onNameChange?: (n: string) => void
  onAdd?: () => void
  onDone: () => void
}

function TunnelForm({ tunnel, onChange, name, onNameChange, onAdd, onDone }: TunnelFormProps) {
  const set = <K extends keyof Tunnel>(key: K, val: Tunnel[K]) =>
    onChange({ ...tunnel, [key]: val })

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        {onNameChange !== undefined && (
          <EntryRow
            title="Name"
            autoFocus
            value={name ?? ''}
            onChange={(e) => onNameChange(e.target.value)}
            placeholder="tun0"
            className="font-mono"
          />
        )}
        <ComboRow title="Mode">
          <Select value={tunnel.mode} onValueChange={(v) => set('mode', v as TunnelMode)}>
            <SelectTrigger className="font-mono">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {TUNNEL_MODES.map((m) => (
                <SelectItem key={m} value={m} className="font-mono">{m}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </ComboRow>
        <EntryRow title="MTU (0 = auto)" value={tunnel.mtu || ''} onChange={(e) => set('mtu', Number(e.target.value) || 0)} placeholder="0" className="font-mono" error={tunnel.mtu ? checkMTU(String(tunnel.mtu)) : null} />
        <EntryRow title="TTL (0 = inherit)" value={tunnel.ttl || ''} onChange={(e) => set('ttl', Number(e.target.value) || 0)} placeholder="0" className="font-mono" />
        <EntryRow title="Local endpoint" value={tunnel.local} onChange={(e) => set('local', e.target.value)} placeholder="203.0.113.1" className="font-mono" />
        <EntryRow title="Remote endpoint" value={tunnel.remote} onChange={(e) => set('remote', e.target.value)} placeholder="203.0.113.2" className="font-mono" />
      </PreferencesGroup>

      <PreferencesGroup title="Addresses">
        <div className="px-4 py-3">
          <TagInput values={tunnel.addresses ?? []} onChange={(v) => set('addresses', v)} placeholder="10.0.0.1/30" mono validate={checkCIDR} />
        </div>
      </PreferencesGroup>

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add tunnel</Button>}
      </div>
    </div>
  )
}

interface TunnelRowProps {
  name: string
  tunnel: Tunnel
  onEdit: () => void
  onDelete: () => void
}

function TunnelRow({ name, tunnel, onEdit, onDelete }: TunnelRowProps) {
  return (
    <div
      onClick={onEdit}
      className="flex items-start gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors"
    >
      <span className="font-mono font-semibold text-sm w-28 shrink-0 truncate pt-0.5">{name}</span>
      <div className="flex flex-wrap gap-1 flex-1 min-w-0">
        <Badge variant="info" className="text-xs">{tunnel.mode}</Badge>
        {(tunnel.local || tunnel.remote) && (
          <Badge variant="outline" className="text-xs font-mono">
            {tunnel.local || '-'} → {tunnel.remote || '-'}
          </Badge>
        )}
        {(tunnel.addresses ?? []).map((addr) => (
          <Badge key={addr} variant="outline" className="text-xs font-mono">{addr}</Badge>
        ))}
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={(e) => { e.stopPropagation(); onDelete() }}
        className="h-7 w-7 p-0 hover:text-destructive shrink-0"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}

const NEW_KEY = '__new__'

export default function Tunnels() {
  const { data, isLoading, reload } = useFetch<TunnelMap>(() => api.apiConfigSectionGet({ section: 'tunnels' }) as Promise<TunnelMap>)
  const [entries, setEntries] = useState<TunnelMap | null>(null)
  const [newName, setNewName] = useState('')
  const [openSheet, setOpenSheet] = useState<string | null>(null)
  const [formDraft, setFormDraft] = useState<Tunnel | null>(null)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('tunnels')

  useDataRefresh(() => { setEntries(null); reset() })

  const current: TunnelMap = entries ?? (data as TunnelMap | null) ?? {}

  const update = (name: string, t: Tunnel) => { setEntries({ ...current, [name]: t }); markDirty() }

  const deleteEntry = (name: string) => {
    const next = { ...current }; delete next[name]; setEntries(next)
    if (openSheet === name) setOpenSheet(null)
    markDirty()
  }

  const handleAddNew = () => { setFormDraft(emptyTunnel()); setNewName(''); setOpenSheet(NEW_KEY) }

  const handleCommitNew = () => {
    const n = newName.trim()
    if (!n) { toast.error('Please enter a tunnel name'); return }
    if (current[n]) { toast.error(`Tunnel "${n}" already exists`); return }
    setEntries({ ...current, [n]: formDraft ?? emptyTunnel() }); markDirty()
    setOpenSheet(null); setFormDraft(null)
  }

  const handleCloseSheet = () => { setOpenSheet(null); setFormDraft(null) }

  const names = Object.keys(current)
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(names, 12)

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      <PageHeader title="Tunnels" description="Manage tunnel interfaces (SIT, GRE, IPIP, IP6)" action={
        <div className="flex items-center gap-2">
          <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(current)} onCancel={() => { setEntries(null); reset() }} />
          <ReloadButton onClick={() => { setEntries(null); reload() }} />
        </div>
      } />

      {names.length === 0 ? (
        <div className="w-full max-w-2xl mx-auto">
          <EmptyState
            icon={<GitBranch />}
            title="No tunnels"
            message="Define a tunnel interface to encapsulate traffic between two endpoints."
            action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add tunnel</Button>}
          />
        </div>
      ) : (
        <PreferencesColumns
          title="Tunnels"
          header={
            <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
              <Plus className="h-4 w-4" />Add
            </Button>
          }
        >
          {pageItems.map((name) => (
            <TunnelRow
              key={name}
              name={name}
              tunnel={current[name]}
              onEdit={() => setOpenSheet(name)}
              onDelete={() => deleteEntry(name)}
            />
          ))}
        </PreferencesColumns>
      )}

      <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="tunnels" />

      <Sheet
        open={!!openSheet}
        onClose={handleCloseSheet}
        title={openSheet === NEW_KEY ? 'New Tunnel' : (openSheet ?? '')}
      >
        {openSheet && (openSheet === NEW_KEY ? formDraft : current[openSheet]) && (
          <TunnelForm
            tunnel={openSheet === NEW_KEY ? formDraft! : current[openSheet]}
            onChange={openSheet === NEW_KEY ? setFormDraft : (u) => update(openSheet, u)}
            name={openSheet === NEW_KEY ? newName : undefined}
            onNameChange={openSheet === NEW_KEY ? setNewName : undefined}
            onAdd={openSheet === NEW_KEY ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
          />
        )}
      </Sheet>
    </div>
  )
}
