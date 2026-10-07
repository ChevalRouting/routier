import { KVPair } from '../shared'

export type RouteMapRow = {
  _id: number
  seq: number
  action: 'permit' | 'deny'
  match: KVPair[]
  set: KVPair[]
  call: string
  on_match: string
  continue: number
}

export { BGPRouteMapsPanel } from './RouteMapsParts/BGPRouteMapsPanel'
export { RouteMapEntryCard } from './RouteMapsParts/RouteMapEntryCard'
export { RouteMapForm } from './RouteMapsParts/RouteMapForm'
