import { ServiceForm } from '@/components/services/ServiceForm'
import { buildPayload, ConfigFileRow, defaultService, NEW_KEY, newConfigFileId, ServiceConfig, ServiceRow, ServicesMap } from '@/components/services/shared'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { Button, EmptyState, PageHeader, Pagination, PreferencesColumns, ReloadButton, SaveButton, Sheet, Spinner, usePagination } from 'cheval-ui'
import { Plus, Server } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

type ServicesShape = { embedded?: boolean }

export default function Services({ embedded = false }: ServicesShape = {}) {
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
        rows[name] = (svc.configs ?? []).map((c) => ({ ...c, id: newConfigFileId() }))
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

  const handleClick = () => { setServices(null); reset(); reload(true) }

  const handleCancel = () => { setServices(null); reset(); reload(true) }

  const actions = (
        <div className="flex items-center gap-2">
          <SaveButton isDirty={isDirty} saving={saving} onClick={() => save(buildPayload(current, configRows))} onCancel={handleCancel} />
          <ReloadButton onClick={handleClick} />
        </div>
  )

  return (
    <div className="space-y-6">
      {embedded ? <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">Manage service enablement and configuration files</p>
        {actions}
      </div> : <PageHeader title="Services" description="Manage service enablement and configuration files" action={actions} />}

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
