import { PrefixEntry } from '../types'

type PrefixListRowShape = { _id: number }

type PREFIXPRESETSShape = { label: string; desc: string; entries: Omit<PrefixEntry, 'seq'>[] }

export type PrefixListRow = PrefixEntry & PrefixListRowShape

export const PREFIX_PRESETS: PREFIXPRESETSShape[] = [
  { label: 'Any', desc: 'permit 0.0.0.0/0 le 32', entries: [{ action: 'permit', prefix: '0.0.0.0/0', le: 32 }] },
  { label: 'Default only', desc: 'permit 0.0.0.0/0', entries: [{ action: 'permit', prefix: '0.0.0.0/0' }] },
  { label: 'Default only (v6)', desc: 'permit ::/0', entries: [{ action: 'permit', prefix: '::/0' }] },
  { label: 'Deny default', desc: 'deny 0.0.0.0/0', entries: [{ action: 'deny', prefix: '0.0.0.0/0' }] },
  {
    label: 'RFC1918', desc: 'permit the three private ranges',
    entries: [
      { action: 'permit', prefix: '10.0.0.0/8', le: 32 },
      { action: 'permit', prefix: '172.16.0.0/12', le: 32 },
      { action: 'permit', prefix: '192.168.0.0/16', le: 32 },
    ],
  },
]

export { BGPPrefixListsPanel } from './PrefixListsParts/BGPPrefixListsPanel'
export { PrefixListForm } from './PrefixListsParts/PrefixListForm'
