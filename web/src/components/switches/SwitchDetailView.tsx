import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { AlertDialog, BackLink, Button, NoticeBanner, PageHeader, PreferencesGroup, Row, Spinner, SwitchFaceplate, ReloadButton } from 'cheval-ui'
import { ListChecks, Settings2, SlidersHorizontal, Trash2 } from 'lucide-react'
import { DraftControls } from '@/components/DraftControls'
import { api } from '@/lib/client'
import { faceplatePorts, pendingPorts, vlanOptions, type InterfacesShape, type SwitchDefinition } from '@/lib/switches'
import { useFetch } from '@/lib/useFetch'
import { useSwitchDefinitions } from '@/lib/useSwitchDefinitions'
import { PlanDialog } from './PlanDialog'
import { PortDialog } from './PortDialog'
import { SwitchSettingsSheet } from './SwitchSettingsSheet'
import { switchHealth } from './switchStatus'

type SwitchDetailViewShape = { name: string }

type Overlay = 'ports' | 'settings' | 'plan' | 'remove' | null

export function SwitchDetailView({ name }: SwitchDetailViewShape) {
  const navigate = useNavigate()
  const draft = useSwitchDefinitions()
  const status = useFetch(() => api.apiSwitchesNameGet({ name }))
  const models = useFetch(() => api.apiSwitchModelsGet())
  const interfaces = useFetch(() => api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<InterfacesShape | null>)
  const [selected, setSelected] = useState<string[]>([])
  const [overlay, setOverlay] = useState<Overlay>(null)
  const [refreshing, setRefreshing] = useState(false)

  const definition = draft.definitions[name]
  const current = status.latest
  const state = current?.state
  const model = current?.model ?? models.latest?.find((candidate) => candidate.id === definition?.model)
  const vlans = useMemo(() => vlanOptions(interfaces.latest, state), [interfaces.latest, state])
  const pending = useMemo(() => pendingPorts(definition, state), [definition, state])
  const health = switchHealth(current ?? undefined)

  const handleSelect = (port: string, additive: boolean) => {
    if (!additive) {
      setSelected([port])
      setOverlay('ports')
      return
    }

    setSelected((current) => current.includes(port) ? current.filter((item) => item !== port) : [...current, port])
  }

  const handlePortClose = () => {
    setSelected([])
    setOverlay(null)
  }

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      const [response] = await Promise.all([api.apiSwitchesNameRefreshPost({ name }), new Promise((resolve) => setTimeout(resolve, 600))])
      await status.reload()
      const error = response.error
      if (error) toast.error(`Could not read ${name}: ${error}`)
      else toast.success(`Read ${name} from the switch`)
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setRefreshing(false)
    }
  }

  const handleChange = (next: SwitchDefinition) => {
    draft.setDefinition(name, next)
    setSelected([])
  }

  const handleRemove = async () => {
    if (!(await draft.removeDefinition(name))) return
    setOverlay(null)
    navigate('/switches')
  }

  if (draft.isLoading || !models.hasData) return <Spinner />

  const actions = (
    <div className="flex flex-wrap items-center gap-1">
      <Button variant="ghost" size="sm" className="gap-1.5" onClick={() => setOverlay('plan')} disabled={!definition || draft.isDirty} title={draft.isDirty ? 'Save the Draft First' : undefined}>
        <ListChecks className="h-3.5 w-3.5" />Sync From Device
      </Button>
      <Button variant="ghost" size="sm" className="gap-1.5" onClick={() => setOverlay('settings')} disabled={!definition}>
        <Settings2 className="h-3.5 w-3.5" />Settings
      </Button>
      <Button variant="ghost" size="sm" className="gap-1.5" onClick={() => setOverlay('remove')} disabled={!definition}>
        <Trash2 className="h-3.5 w-3.5" />Remove…
      </Button>
    </div>
  )

  return (
    <div className="space-y-6">
      <BackLink to="/switches" className="max-md:hidden">Switches</BackLink>
      <PageHeader
        title={name}
        description={[model?.name, definition?.host ?? current?.host].filter(Boolean).join(' · ')}
        action={actions}
      />
      <DraftControls isDirty={draft.isDirty} saving={draft.saving} onClick={draft.saveDraft} onCancel={draft.discard} />
      {!definition && <NoticeBanner intent="warning">This switch is removed in the draft; apply to stop managing it</NoticeBanner>}
      {current?.error && <NoticeBanner intent="danger">{current.error}</NoticeBanner>}
      {!status.hasData && !status.isLoading && definition && <NoticeBanner intent="info">Save the draft and apply to start reading this switch</NoticeBanner>}
      {state && !state.vlan_filtering && <NoticeBanner intent="warning">Bridge VLAN filtering is off on this switch; enable it before managing ports</NoticeBanner>}
      {model && (
        <div className="space-y-3">
          <div className="min-w-0">
            <SwitchFaceplate
              groups={model.faceplate}
              ports={faceplatePorts(state)}
              uplinkPort={definition?.uplink}
              selected={selected}
              pending={pending}
              onSelect={handleSelect}
            />
          </div>
          {selected.length > 0 && overlay === null && (
            <div className="flex flex-wrap items-center gap-2 text-sm">
              <span className="font-mono">{selected.join(', ')}</span>
              <Button size="sm" variant="ghost" onClick={() => setSelected([])}>Clear</Button>
              <Button size="sm" className="gap-1.5" onClick={() => setOverlay('ports')}>
                <SlidersHorizontal className="h-3.5 w-3.5" />Configure…
              </Button>
            </div>
          )}
        </div>
      )}
      {current && (
        <PreferencesGroup
          title="Device"
          header={
            <ReloadButton onClick={handleRefresh} loading={refreshing} label="Read Switch" />
          }
        >
          <Row title="State"><span className="text-sm first-letter:uppercase">{health.label}</span></Row>
          {state?.facts.version && <Row title="Firmware"><span className="font-mono text-sm">{state.facts.version}</span></Row>}
          {state?.facts.serial && <Row title="Serial"><span className="font-mono text-sm">{state.facts.serial}</span></Row>}
          {current.read_at ? <Row title="Last Read"><span className="text-sm tabular-nums">{new Date(current.read_at * 1000).toLocaleString()}</span></Row> : null}
        </PreferencesGroup>
      )}
      {overlay === 'ports' && selected.length > 0 && (
        <PortDialog ports={selected} definition={definition} state={state} vlans={vlans} pending={pending} onChange={handleChange} onClose={handlePortClose} />
      )}
      {overlay === 'settings' && definition && (
        <SwitchSettingsSheet name={name} definition={definition} model={model} onChange={handleChange} onCredentialsSaved={() => void status.reload()} onClose={() => setOverlay(null)} />
      )}
      {overlay === 'plan' && <PlanDialog name={name} onClose={() => setOverlay(null)} />}
      <AlertDialog
        open={overlay === 'remove'}
        onCancel={() => setOverlay(null)}
        onConfirm={handleRemove}
        title={`Stop Managing ${name}?`}
        description="Applying stops managing the switch and leaves its ports and bonds as they are"
        confirmLabel="Remove"
        cancelLabel="Cancel"
        busy={draft.saving}
        destructive
      />
    </div>
  )
}
