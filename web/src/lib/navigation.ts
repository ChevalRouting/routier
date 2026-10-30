import {
  LayoutDashboard, Network, EthernetPort, Route, Lock, Radio, Shield,
  Settings2, Activity, HeartPulse, Megaphone,
  BookOpen, Workflow, Terminal, Boxes, Handshake,
  Cable, Calculator, Globe, KeyRound, Users, Cog, Gauge,
  Download, History, Camera, Archive, Code, FileText, ShieldCheck,
  Cpu, ListTree, ScrollText, ArrowDownUp, Share2, Waypoints, Radar, Waves, ChartSpline,
} from 'lucide-react'

type IconShape = { className?: string }

export interface NavItem {
  to: string
  label: string
  icon: React.ComponentType<IconShape>
  exact?: boolean
}

export interface NavGroup {
  label: string
  items: NavItem[]
}

export const navGroups: NavGroup[] = [
  {
    label: '',
    items: [
      { to: '/', label: 'Dashboard', icon: LayoutDashboard, exact: true },
    ],
  },
  {
    label: 'Network',
    items: [
      { to: '/interfaces', label: 'Interfaces', icon: Network },
      { to: '/switches', label: 'Switches', icon: EthernetPort },
      { to: '/routing', label: 'Routing', icon: Route },
      { to: '/vpn', label: 'VPN', icon: Lock },
      { to: '/anycast', label: 'Anycast', icon: Radio },
      { to: '/dhcp', label: 'DHCP', icon: Cable },
      { to: '/dns', label: 'DNS', icon: Globe },
      { to: '/ha', label: 'HA', icon: HeartPulse },
      { to: '/firewall', label: 'Firewall', icon: Shield },
    ],
  },
  {
    label: 'System',
    items: [
      { to: '/system', label: 'General', icon: Settings2 },
      { to: '/certificates', label: 'Certificates', icon: ShieldCheck },
      { to: '/friends', label: 'Friends', icon: Handshake },
    ],
  },
  {
    label: 'Monitoring',
    items: [
      { to: '/monitor/system', label: 'System', icon: Cpu },
      { to: '/monitor/network', label: 'Network', icon: Activity },
    ],
  },
  {
    label: 'Tools',
    items: [
      { to: '/macros', label: 'Macros', icon: Boxes },
      { to: '/ip-tools', label: 'IP Tools', icon: Calculator },
      { to: '/announcements', label: 'Announcements', icon: Megaphone },
      { to: '/config-browser', label: 'Config Browser', icon: BookOpen },
      { to: '/debug', label: 'Debug', icon: Terminal },
    ],
  },
]

export interface MenuSection {
  title?: string
  items: NavItem[]
}

export interface MenuGroup {
  key: string
  label: string
  items: NavItem[]
  sections?: MenuSection[]
}

export interface SystemSection {
  key: string
  label: string
  group: string
  icon: React.ComponentType<IconShape>
}

export const systemSections: SystemSection[] = [
  { key: 'general', label: 'General', group: 'Configuration', icon: Settings2 },
  { key: 'users', label: 'Users', group: 'Configuration', icon: Users },
  { key: 'services', label: 'Services', group: 'Configuration', icon: Cog },
  { key: 'sysctl', label: 'Sysctl', group: 'Configuration', icon: FileText },
  { key: 'monitoring', label: 'Monitoring', group: 'Configuration', icon: Gauge },
  { key: 'updates', label: 'Updates', group: 'Maintenance', icon: Download },
  { key: 'apply', label: 'Apply History', group: 'Maintenance', icon: History },
  { key: 'snapshots', label: 'Snapshots', group: 'Maintenance', icon: Camera },
  { key: 'backup', label: 'Backup', group: 'Maintenance', icon: Archive },
  { key: 'apikeys', label: 'API Keys', group: 'API', icon: KeyRound },
  { key: 'apidocs', label: 'API Docs', group: 'API', icon: Code },
]

export type MonitorSectionKey = 'system' | 'processes' | 'logs' | 'topology' | 'traffic' | 'live' | 'flows' | 'bgp' | 'neighbors' | 'probes'

export type MonitorGroup = 'System' | 'Network'

export interface MonitorSection extends SystemSection {
  key: MonitorSectionKey
}

export const monitorSections: MonitorSection[] = [
  { key: 'system', label: 'Resources', group: 'System', icon: Cpu },
  { key: 'processes', label: 'Processes', group: 'System', icon: ListTree },
  { key: 'logs', label: 'Logs', group: 'System', icon: ScrollText },
  { key: 'topology', label: 'Topology', group: 'Network', icon: Workflow },
  { key: 'traffic', label: 'Traffic', group: 'Network', icon: ArrowDownUp },
  { key: 'live', label: 'Live Traffic', group: 'Network', icon: ChartSpline },
  { key: 'flows', label: 'Flows', group: 'Network', icon: Waves },
  { key: 'bgp', label: 'BGP', group: 'Network', icon: Share2 },
  { key: 'neighbors', label: 'Neighbors', group: 'Network', icon: Waypoints },
  { key: 'probes', label: 'Probes', group: 'Network', icon: Radar },
]

function sectionMenu(sections: SystemSection[], param: string, path: string): MenuSection[] {
  const groups = [...new Set(sections.map((section) => section.group))]
  return groups.map((group) => ({
    title: group,
    items: sections
      .filter((section) => section.group === group)
      .map((section) => ({ to: `${path}?${param}=${section.key}`, label: section.label, icon: section.icon })),
  }))
}

export const monitorGroupPaths: Record<MonitorGroup, string> = { System: '/monitor/system', Network: '/monitor/network' }

export function monitorSectionPath(key: string): string {
  const section = monitorSections.find((s) => s.key === key)
  if (!section) return monitorGroupPaths.System
  const path = monitorGroupPaths[section.group as MonitorGroup]
  const first = monitorSections.find((s) => s.group === section.group)
  return first?.key === key ? path : `${path}?monitor=${key}`
}

function monitorMenuSections(): MenuSection[] {
  return (Object.keys(monitorGroupPaths) as MonitorGroup[]).map((group) => ({
    title: group,
    items: monitorSections
      .filter((section) => section.group === group)
      .map((section) => ({ to: monitorSectionPath(section.key), label: section.label, icon: section.icon })),
  }))
}

function systemMenuSections(): MenuSection[] {
  return [
    ...sectionMenu(systemSections, 'system-config', '/system'),
    { title: 'Peers', items: [{ to: '/friends', label: 'Friends', icon: Handshake }] },
  ]
}

export const advancedMenus: MenuGroup[] = [
  { key: 'network', label: 'Network', items: navGroups[1].items },
  { key: 'system', label: 'System', items: [], sections: systemMenuSections() },
  { key: 'monitor', label: 'Monitoring', items: [], sections: monitorMenuSections() },
  { key: 'more', label: 'More', items: navGroups[4].items },
]
