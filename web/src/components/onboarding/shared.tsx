import type { TypesSystemNic as SystemNic } from '@/api'
import { Button } from 'cheval-ui'

type STEPSShape = { key: Step; label: string }

type StepNavShape = {
  onBack: () => void
  onNext: () => void
  nextLabel?: string
  nextDisabled?: boolean
}

export type Step = 'password' | 'hostname' | 'interfaces' | 'review' | 'confirm'

export const STEPS: STEPSShape[] = [
  { key: 'password',   label: 'Password' },
  { key: 'hostname',   label: 'Hostname' },
  { key: 'interfaces', label: 'Interfaces' },
  { key: 'review',     label: 'Review' },
  { key: 'confirm',    label: 'Confirm' },
]

export type IfaceRole = 'internet' | 'local' | 'none'

export type AddrMode = 'dhcp' | 'static'

export interface IfaceSetting { role: IfaceRole; cidr: string; mode: AddrMode; gateway: string; dns: string[] }

export function StepNav({ onBack, onNext, nextLabel = 'Next', nextDisabled }: StepNavShape) {
  return (
    <div className="flex justify-between">
      <Button variant="outline" onClick={onBack}>Back</Button>
      <Button onClick={onNext} disabled={nextDisabled}>{nextLabel}</Button>
    </div>
  )
}

export function selectorFor(nic: SystemNic): string {
  return nic.mac ? `mac(${nic.mac})` : `name=${nic.name}`
}
