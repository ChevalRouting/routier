import { ServiceForm } from '@/components/anycast/ServiceForm'
import { AnycastService, emptyService, RoutingSection, ServiceRow } from '@/components/anycast/shared'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { Button, EmptyState, PageHeader, Pagination, PreferencesColumns, ReloadButton, SaveButton, Sheet, Spinner, usePagination } from 'cheval-ui'
import {
  Plus,
  Radio
} from 'lucide-react'
import { useEffect, useState } from 'react'

export default function Anycast() {
  const { data, isLoading, reload } = useFetch<RoutingSection>(
    () => api.apiConfigSectionGet({ section: 'routing' }) as Promise<RoutingSection>
  )
  const [routing, setRouting] = useState<RoutingSection | null>(null)
  const [initialized, setInitialized] = useState(false)
  useDataRefresh(() => setInitialized(false))
  const [openIdx, setOpenIdx] = useState<number | null>(null)
  const [formDraft, setFormDraft] = useState<AnycastService | null>(null)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('routing')

  useEffect(() => {
    if (data && !initialized) {
      setRouting(data)
      setInitialized(true)
    }
  }, [data, initialized])

  const services = routing?.anycast?.services ?? []

  const updateServices = (next: AnycastService[]) => {
    setRouting({ ...routing, anycast: { services: next } })
    markDirty()
  }

  const deleteService = (idx: number) => {
    updateServices(services.filter((_, i) => i !== idx))
    if (openIdx === idx) setOpenIdx(null)
  }

  const handleAddNew = () => { setFormDraft(emptyService()); setOpenIdx(-1) }

  const handleCommitNew = () => {
    if (!formDraft) return
    updateServices([...services, formDraft])
    setOpenIdx(null); setFormDraft(null)
  }

  const handleCloseSheet = () => { setOpenIdx(null); setFormDraft(null) }

  const { page, setPage, totalPages, pageItems, total, pageSize } = usePagination(services, 12)

  if (isLoading) return <Spinner />

  const isAdding = openIdx === -1
  const openSvc = openIdx !== null && openIdx >= 0 ? services[openIdx] : null
  const sheetTitle = isAdding ? 'Add Service' : (openSvc?.name || 'Service')

  const handleClick = () => { setRouting(null); setInitialized(false); reset(); reload(true) }

  const handleCancel = () => { setRouting(null); setInitialized(false); reset(); reload(true) }

  return (
    <div className="space-y-6">
      <PageHeader title="Anycast" description="Manage anycast services and health-checked endpoints" action={
        <div className="flex items-center gap-2">
          <SaveButton isDirty={isDirty} saving={saving} onClick={() => routing && save(routing)} onCancel={handleCancel} />
          <ReloadButton onClick={handleClick} />
        </div>
      } />

      {services.length === 0 ? (
        <EmptyState
          className="max-w-2xl mx-auto"
          icon={<Radio />}
          title="No anycast services"
          message="Define an anycast service to advertise a shared address across healthy endpoints."
          action={<Button variant="outline" size="sm" onClick={handleAddNew} className="gap-2"><Plus className="h-4 w-4" />Add service</Button>}
        />
      ) : (
        <PreferencesColumns
          title="Services"
          header={
            <Button variant="outline" size="sm" onClick={handleAddNew} className="gap-1.5">
              <Plus className="h-4 w-4" />Add
            </Button>
          }
        >
          {pageItems.map((svc, localIdx) => {
            const idx = page * pageSize + localIdx
            return (
              <ServiceRow
                key={idx}
                service={svc}
                onEdit={() => setOpenIdx(idx)}
                onDelete={() => deleteService(idx)}
              />
            )
          })}
        </PreferencesColumns>
      )}

      <Pagination page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} unit="services" />

      <Sheet
        open={openIdx !== null}
        onClose={handleCloseSheet}
        title={sheetTitle}
        className="max-w-2xl"
      >
        {openIdx !== null && (isAdding ? formDraft : openSvc) && (
          <ServiceForm
            service={isAdding ? formDraft! : openSvc!}
            onChange={isAdding ? setFormDraft : (u) => updateServices(services.map((s, i) => i === openIdx ? u : s))}
            onAdd={isAdding ? handleCommitNew : undefined}
            onDone={handleCloseSheet}
          />
        )}
      </Sheet>
    </div>
  )
}
