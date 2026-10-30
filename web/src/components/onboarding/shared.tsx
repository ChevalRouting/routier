import type { TypesSetupHost as SetupHost, TypesSystemNic as SystemNic } from '@/api'
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

function ipv4Address(nic: SystemNic): string {
  return nic.addrs?.find((addr) => addr.includes('.')) ?? ''
}

export function defaultSetting(nic: SystemNic, host: SetupHost | null): IfaceSetting {
  const cidr = ipv4Address(nic)
  if (host && nic.name === host.default_iface) {
    const dhcp = host.dhcp_ifaces.includes(nic.name) || !cidr
    return {
      role: 'internet',
      cidr,
      mode: dhcp ? 'dhcp' : 'static',
      gateway: host.gateway4 ?? '',
      dns: host.nameservers,
    }
  }
  if (host && cidr && !host.dhcp_ifaces.includes(nic.name)) {
    return { role: 'local', cidr, mode: 'static', gateway: '', dns: [] }
  }
  return { role: 'none', cidr, mode: 'dhcp', gateway: '', dns: [] }
}
