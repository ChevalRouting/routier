import type { TypesFriendStatus as FriendStatus } from '@/api'

export function statusColor(st?: FriendStatus): string {
  if (!st) return 'bg-muted-foreground/40'
  if (!st.reachable) return 'bg-danger'
  return 'bg-success'
}

export { AddFriendSheet } from './FriendsPanelParts/AddFriendSheet'
export { EditFriendSheet } from './FriendsPanelParts/EditFriendSheet'
export { FriendCard } from './FriendsPanelParts/FriendCard'
export { FriendsPanel } from './FriendsPanelParts/FriendsPanel'
