import { useState, useEffect } from 'react'
import { toast } from 'sonner'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { useDataRefresh } from '@/lib/dataVersion'
import { usePageSave } from '@/lib/usePageSave'
import { SaveButton } from 'cheval-ui'
import { Button } from 'cheval-ui'
import { Badge } from 'cheval-ui'
import { Sheet } from 'cheval-ui'
import { Plus, Trash2, Server } from 'lucide-react'
import { PageHeader } from 'cheval-ui'
import { PreferencesGroup, PreferencesColumns, EntryRow, SwitchRow } from 'cheval-ui'
import { EmptyState } from 'cheval-ui'
import { Pagination, usePagination } from 'cheval-ui'
import { ReloadButton } from 'cheval-ui'
import { Spinner } from 'cheval-ui'

interface ServiceConfig {
  enable: boolean
  configs: ConfigFile[]
  after: string
}

interface ConfigFile {
  src: string
  dest: string
  mode: number
}

type ServicesMap = Record<string, ServiceConfig>

let nextFileId = 1

interface ConfigFileRow extends ConfigFile {
  id: number
}

function defaultService(): ServiceConfig {
  return { enable: true, configs: [], after: '' }
}

function buildPayload(services: ServicesMap, configRows: Record<string, ConfigFileRow[]>): ServicesMap {
  const payload: ServicesMap = {}
  for (const name of Object.keys(services)) {
    payload[name] = {
      ...services[name],
      configs: (configRows[name] ?? []).map(({ id: _id, ...rest }) => rest),
    }
  }
  return payload
}

interface ServiceFormProps {
  service: ServiceConfig
  rows: ConfigFileRow[]
  onServiceChange: (s: ServiceConfig) => void
  onRowsChange: (r: ConfigFileRow[]) => void
  nameInput?: string
  onNameChange?: (n: string) => void
  onAdd?: () => void
  onDone: () => void
}

function ServiceForm({ service, rows, onServiceChange, onRowsChange, nameInput, onNameChange, onAdd, onDone }: ServiceFormProps) {
  const set = <K extends keyof ServiceConfig>(key: K, val: ServiceConfig[K]) =>
    onServiceChange({ ...service, [key]: val })

  const addRow = () => onRowsChange([...rows, { id: nextFileId++, src: '', dest: '', mode: 644 }])
  const updRow = (id: number, patch: Partial<ConfigFileRow>) =>
    onRowsChange(rows.map((r) => (r.id === id ? { ...r, ...patch } : r)))
  const delRow = (id: number) => onRowsChange(rows.filter((r) => r.id !== id))

  return (
    <div className="space-y-5">
      <PreferencesGroup>
        {onNameChange !== undefined && (
          <EntryRow
            title="Service name"
            autoFocus
            value={nameInput ?? ''}
            onChange={(e) => onNameChange(e.target.value)}
            placeholder="nginx"
            className="font-mono"
          />
        )}
        <SwitchRow
          title="Enabled"
          subtitle="Start on boot and keep running"
          checked={service.enable}
          onCheckedChange={(v) => set('enable', v)}
        />
        <EntryRow
          title="After"
          value={service.after}
          onChange={(e) => set('after', e.target.value)}
          placeholder="network"
          className="font-mono"
        />
      </PreferencesGroup>

      <PreferencesGroup
        title="Config files"
        description="Files deployed to the target when this service is applied."
        header={
          <Button variant="outline" size="sm" onClick={addRow} className="gap-1.5">
            <Plus className="h-4 w-4" />Add
          </Button>
        }
      >
        {rows.length === 0 ? (
          <p className="px-4 py-3 text-sm text-muted-foreground">No config files, this service runs without any file deployments.</p>
        ) : (
          rows.map((row) => (
            <div key={row.id} className="flex items-center gap-2 px-4 py-2">
              <input
                value={row.src}
                onChange={(e) => updRow(row.id, { src: e.target.value })}
                placeholder="/templates/nginx.conf"
                className="min-w-0 flex-1 bg-transparent font-mono text-xs text-foreground outline-none placeholder:text-muted-foreground/50"
              />
              <span className="shrink-0 text-muted-foreground">→</span>
              <input
                value={row.dest}
                onChange={(e) => updRow(row.id, { dest: e.target.value })}
                placeholder="/etc/nginx/nginx.conf"
                className="min-w-0 flex-1 bg-transparent font-mono text-xs text-foreground outline-none placeholder:text-muted-foreground/50"
              />
              <input
                value={row.mode}
                onChange={(e) => updRow(row.id, { mode: Number(e.target.value) })}
                placeholder="644"
                className="w-12 shrink-0 bg-transparent font-mono text-xs text-foreground outline-none placeholder:text-muted-foreground/50"
              />
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive"
                onClick={() => delRow(row.id)}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          ))
        )}
      </PreferencesGroup>

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="outline" onClick={onDone}>{onAdd ? 'Cancel' : 'Done'}</Button>
        {onAdd && <Button onClick={onAdd}>Add service</Button>}
      </div>
    </div>
  )
}

interface ServiceRowProps {
  name: string
  service: ServiceConfig
  configCount: number
  onEdit: () => void
  onDelete: () => void
}

function ServiceRow({ name, service, configCount, onEdit, onDelete }: ServiceRowProps) {
  return (
    <div
      onClick={onEdit}
      className="flex items-start gap-3 px-4 py-3 hover:bg-accent/50 cursor-pointer transition-colors"
    >
      <span className="font-mono font-semibold text-sm w-28 shrink-0 truncate pt-0.5">{name}</span>
      <div className="flex flex-wrap gap-1 flex-1 min-w-0">
        <Badge variant={service.enable ? 'success' : 'neutral'} className="text-xs">
          {service.enable ? 'enabled' : 'disabled'}
        </Badge>
        {service.after && (
          <Badge variant="outline" className="text-xs font-mono">after: {service.after}</Badge>
        )}
        {configCount > 0 && (
          <Badge variant="outline" className="text-xs">{configCount} file{configCount !== 1 ? 's' : ''}</Badge>
        )}
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

export default function Services() {
  const { data, isLoading, reload } = useFetch<ServicesMap>(
    () => api.apiConfigSectionGet({ section: 'services' }) as Promise<ServicesMap>
  )
  const [services, setServices] = useState<ServicesMap | null>(null)
  const [configRows, setConfigRows] = useState<Record<string, ConfigFileRow[]>>({})
  const [newName, setNewName] = useState('')
  const [openSheet, setOpenSheet] = useState<string | null>(null)
  const [formDraft, setFormDraft] = useState<ServiceConfig | null>(null)
  const [formDraftRows, setFormDraftRows] = useState<ConfigFileRow[]>([])
  const { isDirty, markDirty, save, saving, reset } = usePageSave('services')

  useDataRefresh(() => { setServices(null); setConfigRows({}); reset() })

  useEffect(() => {
    if (data && services === null) {
      setServices(data)
      const rows: Record<string, ConfigFileRow[]> = {}
      for (const [name, svc] of Object.entries(data)) {
        rows[name] = (svc.configs ?? []).map((c) => ({ ...c, id: nextFileId++ }))
      }
      setConfigRows(rows)
    }
  }, [data, services])

  const current: ServicesMap = services ?? (data as ServicesMap | null) ?? {}

  const updateService = (name: string, patch: Partial<ServiceConfig>) => {
    setServices({ ...current, [name]: { ...current[name], ...patch } }); markDirty()
  }
  const updateRows = (name: string, rows: ConfigFileRow[]) => {
    setConfigRows({ ...configRows, [name]: rows }); markDirty()
  }

  const deleteService = (name: string) => {
    const nextSvc = { ...current }; delete nextSvc[name]
    const nextRows = { ...configRows }; delete nextRows[name]
    setServices(nextSvc); setConfigRows(nextRows)
    if (openSheet === name) setOpenSheet(null)
    markDirty()
  }

  const handleAddNew = () => {
    setFormDraft(defaultService()); setFormDraftRows([]); setNewName(''); setOpenSheet(NEW_KEY)
  }

  const handleCommitNew = () => {
    const n = newName.trim()
    if (!n) { toast.error('Please enter a service name'); return }
    if (current[n]) { toast.error(`Service "${n}" already exists`); return }
    setServices({ ...current, [n]: formDraft ?? defaultService() })
    setConfigRows({ ...configRows, [n]: formDraftRows })
    markDirty(); setOpenSheet(null); setFormDraft(null)
  }

  const handleCloseSheet = () => { setOpenSheet(null); setFormDraft(null) }

  const names = Object.keys(current)
  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(names, 12)

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      <PageHeader title="Services" description="Manage service enablement and configuration files" action={
        <div className="flex items-center gap-2">
          <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(buildPayload(current, configRows))} onCancel={() => { setServices(null); reset() }} />
          <ReloadButton onClick={() => { setServices(null); reload() }} />
        </div>
      } />

      {names.length === 0 ? (
        <div className="w-full max-w-2xl mx-auto">
          <EmptyState
            icon={<Server />}
            title="No services"
            message="Define a service to manage its enablement and deploy its configuration files."
            action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add service</Button>}
          />
        </div>
      ) : (
        <PreferencesColumns
          title="Services"
          header={
            <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
              <Plus className="h-4 w-4" />Add
            </Button>
          }
        >
          {pageItems.map((name) => (
            <ServiceRow
              key={name}
              name={name}
              service={current[name]}
              configCount={(configRows[name] ?? []).length}
              onEdit={() => setOpenSheet(name)}
              onDelete={() => deleteService(name)}
            />
          ))}
        </PreferencesColumns>
      )}

      <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="services" />

      <Sheet
        open={!!openSheet}
        onClose={handleCloseSheet}
        title={openSheet === NEW_KEY ? 'New Service' : (openSheet ?? '')}
      >
        {openSheet && (openSheet === NEW_KEY ? formDraft : current[openSheet]) && (
          <ServiceForm
            service={openSheet === NEW_KEY ? formDraft! : current[openSheet]}
            rows={openSheet === NEW_KEY ? formDraftRows : (configRows[openSheet] ?? [])}
            onServiceChange={openSheet === NEW_KEY ? setFormDraft : (s) => updateService(openSheet, s)}
            onRowsChange={openSheet === NEW_KEY ? setFormDraftRows : (r) => updateRows(openSheet, r)}
            nameInput={openSheet === NEW_KEY ? newName : undefined}
            onNameChange={openSheet === NEW_KEY ? setNewName : undefined}
            onAdd={openSheet === NEW_KEY ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
          />
        )}
      </Sheet>
    </div>
  )
}
