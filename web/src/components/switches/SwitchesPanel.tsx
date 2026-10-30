import { useEffect, useState } from 'react'
import { Button, EmptyState, PageHeader, ReloadButton, SectionLabel, Spinner } from 'cheval-ui'
import { Network, Plus } from 'lucide-react'
import type { TypesSwitchCandidate as SwitchCandidate } from '@/api'
import { DraftControls } from '@/components/DraftControls'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { useSwitchDefinitions } from '@/lib/useSwitchDefinitions'
import { CandidatesTable } from './CandidatesTable'
import { EnrollSheet } from './EnrollSheet'
import { SwitchesTable } from './SwitchesTable'

const POLL_INTERVAL_MS = 30_000

export function SwitchesPanel() {
  const draft = useSwitchDefinitions()
  const statuses = useFetch(() => api.apiSwitchesGet())
  const candidates = useFetch(() => api.apiSwitchesCandidatesGet())
  const models = useFetch(() => api.apiSwitchModelsGet())
  const interfaces = useFetch(() => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<Record<string, unknown> | null>)
  const [enrolling, setEnrolling] = useState<SwitchCandidate | 'manual' | null>(null)

  const reloadStatuses = statuses.reload
  useEffect(() => {
    const timer = window.setInterval(() => void reloadStatuses(), POLL_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [reloadStatuses])

  const reloadAll = () => {
    void statuses.reload()
    void candidates.reload()
  }

  const pendingCandidates = (candidates.data ?? []).filter((candidate) => !Object.values(draft.definitions).some((definition) => definition.host === candidate.mgmt_ip))
  const hasSwitches = Object.keys(draft.definitions).length > 0 || (statuses.latest ?? []).length > 0

  return (
    <div className="space-y-6">
      <PageHeader
        title="Switches"
        action={
          <div className="flex items-center gap-2">
            <ReloadButton onClick={reloadAll} />
            <Button className="gap-1.5" onClick={() => setEnrolling('manual')} disabled={!models.data}>
              <Plus className="h-4 w-4" />Add Switch…
            </Button>
          </div>
        }
      />
      <DraftControls isDirty={draft.isDirty} saving={draft.saving} onClick={draft.saveDraft} onCancel={draft.discard} />
      {(draft.isLoading || (!statuses.hasData && statuses.isLoading)) && <Spinner />}
      {draft.error && <p className="text-sm text-danger">{draft.error}</p>}
      {!draft.isLoading && !hasSwitches && (
        <EmptyState icon={<Network />} title="No Managed Switch" message="Add one, or enable LLDP under General › Monitoring to discover it" />
      )}
      {hasSwitches && <SwitchesTable definitions={draft.definitions} statuses={statuses.latest ?? []} />}
      {pendingCandidates.length > 0 && (
        <section className="space-y-2">
          <SectionLabel>Discovered</SectionLabel>
          <CandidatesTable candidates={pendingCandidates} onManage={setEnrolling} />
        </section>
      )}
      {enrolling && models.data && (
        <EnrollSheet
          candidate={enrolling === 'manual' ? undefined : enrolling}
          models={models.data}
          interfaces={Object.keys(interfaces.data ?? {})}
          taken={Object.keys(draft.definitions)}
          onEnroll={draft.setDefinition}
          onClose={() => setEnrolling(null)}
        />
      )}
    </div>
  )
}
