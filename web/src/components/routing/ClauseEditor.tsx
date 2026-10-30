
export type ClauseKind = 'text' | 'number' | 'none' | 'prefix-list' | 'select'
export interface ClauseOp {
  key: string
  label: string
  desc: string
  group: string
  kind?: ClauseKind
  options?: string[]
  placeholder?: string
}

export const CUSTOM_OP = '__custom__'

export const MATCH_OPS: ClauseOp[] = [
  { key: 'ip address prefix-list', label: 'IP Prefix-List', desc: 'Match IPv4 prefixes against a prefix-list', group: 'Prefix', kind: 'prefix-list' },
  { key: 'ipv6 address prefix-list', label: 'IPv6 Prefix-List', desc: 'Match IPv6 prefixes against a prefix-list', group: 'Prefix', kind: 'prefix-list' },
  { key: 'ip next-hop prefix-list', label: 'Next-Hop Prefix-List', desc: 'Match next-hop against a prefix-list', group: 'Prefix', kind: 'prefix-list' },
  { key: 'as-path', label: 'AS-path', desc: 'Match against an as-path access-list', group: 'BGP', placeholder: 'AS-PATH-ACL' },
  { key: 'community', label: 'Community', desc: 'Match a community-list', group: 'BGP', placeholder: 'COMM-LIST' },
  { key: 'large-community', label: 'Large Community', desc: 'Match a large-community-list', group: 'BGP', placeholder: 'LARGE-LIST' },
  { key: 'extcommunity', label: 'Ext Community', desc: 'Match an extended-community-list', group: 'BGP', placeholder: 'EXT-LIST' },
  { key: 'local-preference', label: 'Local Preference', desc: 'Match on local preference value', group: 'BGP', kind: 'number', placeholder: '100' },
  { key: 'metric', label: 'Metric (MED)', desc: 'Match on the metric / MED', group: 'BGP', kind: 'number', placeholder: '100' },
  { key: 'origin', label: 'Origin', desc: 'Match BGP origin code', group: 'BGP', kind: 'select', options: ['igp', 'egp', 'incomplete'] },
  { key: 'peer', label: 'Peer', desc: 'Match the peer the update came from', group: 'BGP', placeholder: '10.0.0.1' },
  { key: 'source-protocol', label: 'Source Protocol', desc: 'Match the routing protocol that sourced the route', group: 'Advanced', kind: 'select', options: ['bgp', 'ospf', 'ospf6', 'rip', 'static', 'connected', 'kernel', 'isis'] },
  { key: 'source-vrf', label: 'Source VRF', desc: 'Match the VRF the route was learned in', group: 'Advanced', placeholder: 'default' },
  { key: 'tag', label: 'Tag', desc: 'Match the route tag', group: 'Advanced', kind: 'number', placeholder: '100' },
]

export const SET_OPS: ClauseOp[] = [
  { key: 'local-preference', label: 'Local Preference', desc: 'BGP local preference (higher is preferred)', group: 'Common', kind: 'number', placeholder: '200' },
  { key: 'metric', label: 'Metric (MED)', desc: 'MED metric (absolute), or +N / -N to adjust', group: 'Common', placeholder: '100 or +10' },
  { key: 'weight', label: 'Weight', desc: 'Cisco-style weight (higher is preferred, local only)', group: 'Common', kind: 'number', placeholder: '100' },
  { key: 'origin', label: 'Origin', desc: 'BGP origin code', group: 'Common', kind: 'select', options: ['igp', 'egp', 'incomplete'] },
  { key: 'ip next-hop', label: 'IP Next-Hop', desc: 'Rewrite the IPv4 next-hop', group: 'Common', placeholder: '10.0.0.1' },
  { key: 'ipv6 next-hop global', label: 'IPv6 Next-Hop', desc: 'Rewrite the IPv6 global next-hop', group: 'Common', placeholder: '2001:db8::1' },
  { key: 'as-path prepend', label: 'AS-path Prepend', desc: 'Prepend ASNs to the AS-path', group: 'AS-path', placeholder: '65001 65001' },
  { key: 'as-path exclude', label: 'AS-path Exclude', desc: 'Remove matching ASNs from the AS-path', group: 'AS-path', placeholder: '65001' },
  { key: 'community', label: 'Community', desc: 'Set communities (append with "additive")', group: 'Community', placeholder: '65000:100 or no-export' },
  { key: 'large-community', label: 'Large Community', desc: 'Set large communities', group: 'Community', placeholder: '65000:1:100' },
  { key: 'comm-list', label: 'Delete Communities', desc: 'Delete communities matching a community-list', group: 'Community', placeholder: 'COMM-LIST delete' },
  { key: 'aggregator as', label: 'Aggregator', desc: 'Set the BGP aggregator attribute', group: 'Advanced', placeholder: '65000 10.0.0.1' },
  { key: 'atomic-aggregate', label: 'Atomic Aggregate', desc: 'Set the atomic-aggregate attribute', group: 'Advanced', kind: 'none' },
  { key: 'originator-id', label: 'Originator ID', desc: 'Set the BGP originator-id attribute', group: 'Advanced', placeholder: '10.0.0.1' },
  { key: 'distance', label: 'Distance', desc: 'Administrative distance for the route', group: 'Advanced', kind: 'number', placeholder: '20' },
  { key: 'tag', label: 'Tag', desc: 'Set the route tag', group: 'Advanced', kind: 'number', placeholder: '100' },
  { key: 'table', label: 'Kernel Table', desc: 'Export the route to a non-main kernel table', group: 'Advanced', kind: 'number', placeholder: '100' },
  { key: 'src', label: 'Source Address', desc: 'Source address for the installed route', group: 'Advanced', placeholder: '10.0.0.1' },
]

export { ClauseEditor } from './ClauseEditorParts/ClauseEditor'
export { ClauseValueInput } from './ClauseEditorParts/ClauseValueInput'
