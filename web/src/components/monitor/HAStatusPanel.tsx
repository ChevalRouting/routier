export { BreakdownRow } from 'cheval-ui'

export const PROTO_COLORS: Record<string, string> = { tcp: 'bg-blue-500', udp: 'bg-amber-500', icmp: 'bg-green-500', icmpv6: 'bg-emerald-500' }
export const STATE_COLORS: Record<string, string> = { ESTABLISHED: 'bg-green-500', TIME_WAIT: 'bg-amber-400', CLOSE_WAIT: 'bg-orange-400', FIN_WAIT: 'bg-orange-500', SYN_SENT: 'bg-blue-400', SYN_RECV: 'bg-blue-500', LAST_ACK: 'bg-amber-500', CLOSE: 'bg-red-500' }

export { HAStatusPanel } from './HAStatusPanelParts/HAStatusPanel'
export { VRRPInstanceCard } from './HAStatusPanelParts/VRRPInstanceCard'
