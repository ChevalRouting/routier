import { useEffect, useState } from 'react'
import { Dialog, NoticeBanner, Spinner } from 'cheval-ui'
import { api } from '@/lib/client'

type PlanDialogShape = { name: string; onClose: () => void }

export function PlanDialog({ name, onClose }: PlanDialogShape) {
  const [steps, setSteps] = useState<string[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let active = true
    api.apiSwitchesNamePlanGet({ name })
      .then((plan) => { if (active) setSteps(plan.steps) })
      .catch((err: Error) => { if (active) setError(err.message) })
    return () => { active = false }
  }, [name])

  return (
    <Dialog open onClose={onClose} title={`Device Changes for ${name}`} description="What applying the saved draft would send to the switch" className="max-w-3xl">
      {error && <NoticeBanner intent="danger">{error}</NoticeBanner>}
      {!steps && !error && <Spinner />}
      {steps && (
        <pre className="max-h-96 overflow-auto rounded-md bg-muted p-3 font-mono text-xs leading-relaxed">
          {steps.join('\n') || 'The switch already matches the draft.'}
        </pre>
      )}
    </Dialog>
  )
}
