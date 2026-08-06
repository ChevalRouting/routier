import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom'
import { Suspense, useEffect, useState, useMemo } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { useDataVersion } from '@/lib/dataVersion'
import { toast } from 'sonner'
import { clearToken, cn } from '@/lib/utils'
import { addStagingListener, api } from '@/lib/client'
import { DiffView, withContext, type ViewLine } from '@/components/DiffView'
import DiffModal from '@/components/DiffModal'
import { WatchdogConfirmBar } from '@/components/WatchdogConfirmBar'
import Logo from '@/components/Logo'
import { SectionLabel } from '@/components/SectionLabel'
import { useTheme } from '@/lib/useTheme'
import { Spinner } from '@/components/Spinner'
import { AnnouncementBanner } from '@/components/AnnouncementBanner'
import { InstanceSwitcher, InstanceMenu, useActiveInstance } from '@/components/InstanceSwitcher'
import { SELF, switchInstance } from '@/lib/instance'
import {
  LayoutDashboard, Network, GitBranch, Route, Lock, Radio, Shield,
  Settings2, Users, Server, LogOut, Activity, Share2, Megaphone,
  BookOpen, KeyRound, Workflow, Play, Terminal, Boxes, Handshake,
  ChevronLeft, ChevronRight, Sun, Moon, MoreHorizontal, X, RefreshCw, Cable,
  Calculator,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

interface NavItem {
  to: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  exact?: boolean
}

interface NavGroup {
  label: string
  items: NavItem[]
}

const navGroups: NavGroup[] = [
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
      { to: '/tunnels', label: 'Tunnels', icon: GitBranch },
      { to: '/routing', label: 'Routing', icon: Route },
      { to: '/wireguard', label: 'WireGuard', icon: Lock },
      { to: '/anycast', label: 'Anycast', icon: Radio },
      { to: '/dhcp', label: 'DHCP', icon: Cable },
      { to: '/firewall', label: 'Firewall', icon: Shield },
    ],
  },
  {
    label: 'System',
    items: [
      { to: '/system', label: 'General', icon: Settings2 },
      { to: '/users', label: 'Users', icon: Users },
      { to: '/services', label: 'Services', icon: Server },
      { to: '/conntrackd', label: 'Conntrackd', icon: Share2 },
      { to: '/friends', label: 'Friends', icon: Handshake },
    ],
  },
  {
    label: 'Monitoring',
    items: [
      { to: '/monitor', label: 'Monitor', icon: Activity },
      { to: '/topology', label: 'Topology', icon: Workflow },
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

interface MobileNavGroup {
  key: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  items: NavItem[]
  withExtras?: boolean
}

const mobileNavGroups: MobileNavGroup[] = [
  {
    key: 'dashboard',
    label: 'Home',
    icon: LayoutDashboard,
    items: navGroups[0].items,
  },
  {
    key: 'network',
    label: 'Network',
    icon: Network,
    items: navGroups[1].items,
  },
  {
    key: 'system',
    label: 'System',
    icon: Settings2,
    items: navGroups[2].items,
  },
  {
    key: 'monitor',
    label: 'Monitor',
    icon: Activity,
    items: navGroups[3].items,
  },
  {
    key: 'more',
    label: 'More',
    icon: MoreHorizontal,
    items: [
      ...navGroups[4].items,
      { to: '/settings', label: 'Settings', icon: KeyRound },
    ],
    withExtras: true,
  },
]

function groupKeyForPath(pathname: string): string {
  if (pathname === '/settings') return 'more'
  if (['/hostname', '/dns', '/ssh', '/sysctl'].includes(pathname)) return 'system'
  if (['/traffic', '/logs', '/ha-status', '/neighbors', '/system'].some(p => pathname === p || pathname.startsWith(p + '/'))) return 'monitor'
  for (const group of navGroups) {
    for (const item of group.items) {
      const match = item.exact
        ? pathname === item.to
        : pathname === item.to || pathname.startsWith(item.to + '/')
      if (match) {
        if (group.label === '') return 'dashboard'
        if (group.label === 'Network') return 'network'
        if (group.label === 'System') return 'system'
        if (group.label === 'Monitoring') return 'monitor'
        if (group.label === 'Tools') return 'more'
      }
    }
  }
  return 'dashboard'
}

interface MobileDrawerProps {
  group: MobileNavGroup
  onClose: () => void
  theme: string
  onToggleTheme: () => void
  onLogout: () => void
}

function MobileDrawer({ group, onClose, theme, onToggleTheme, onLogout }: MobileDrawerProps) {
  return (
    <Dialog.Root open onOpenChange={(o) => { if (!o) onClose() }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50 md:hidden" />
        <Dialog.Content
          aria-describedby={undefined}
          className="fixed inset-x-0 bottom-0 z-50 flex flex-col bg-background border-t rounded-t-2xl shadow-2xl max-h-[75vh] md:hidden focus:outline-none"
          onEscapeKeyDown={onClose}
          onInteractOutside={onClose}
        >
          <div className="flex justify-center pt-2.5 pb-1 shrink-0">
            <div className="h-1 w-10 rounded-full bg-muted-foreground/25" />
          </div>

          <div className="flex items-center justify-between px-5 py-3 border-b shrink-0">
            <Dialog.Title className="text-base font-semibold">{group.label}</Dialog.Title>
            <Dialog.Close asChild>
              <button
                type="button"
                className="text-muted-foreground hover:text-foreground transition-colors"
              >
                <X className="h-5 w-5" />
              </button>
            </Dialog.Close>
          </div>

          <div className="overflow-y-auto py-2 px-3 pb-4">
            <div className="space-y-0.5">
              {group.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.exact}
                  onClick={onClose}
                  className={({ isActive }) =>
                    cn(
                      'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                      isActive
                        ? 'bg-primary/10 text-primary'
                        : 'text-foreground/80 hover:bg-muted/60 hover:text-foreground'
                    )
                  }
                >
                  <item.icon className="h-4 w-4 shrink-0 opacity-60" />
                  {item.label}
                </NavLink>
              ))}
            </div>

            {group.withExtras && (
              <div className="mt-2 pt-2 border-t border-border/50 space-y-0.5">
                <div className="px-3 pb-1">
                  <SectionLabel className="text-muted-foreground/60">Instance</SectionLabel>
                </div>
                <InstanceMenu tone="sheet" onDone={onClose} />

                <div className="my-2 border-t border-border/50" />
                <button
                  onClick={() => { onToggleTheme(); onClose() }}
                  className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium text-foreground/80 hover:bg-muted/60 hover:text-foreground w-full transition-colors"
                >
                  {theme === 'dark'
                    ? <Sun className="h-4 w-4 shrink-0 opacity-60" />
                    : <Moon className="h-4 w-4 shrink-0 opacity-60" />
                  }
                  {theme === 'dark' ? 'Light mode' : 'Dark mode'}
                </button>
                <button
                  onClick={() => { onLogout(); onClose() }}
                  className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium text-destructive hover:bg-destructive/10 w-full transition-colors"
                >
                  <LogOut className="h-4 w-4 shrink-0" />
                  Sign out
                </button>
              </div>
            )}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { theme, toggle: toggleTheme } = useTheme()
  const [staging, setStaging] = useState(false)
  const [showDiff, setShowDiff] = useState(false)
  const [applying, setApplying] = useState(false)
  const [pendingPoll, setPendingPoll] = useState(0)
  const [showSaveMacro, setShowSaveMacro] = useState(false)
  const [macroName, setMacroName] = useState('')
  const [macroDesc, setMacroDesc] = useState('')
  const [savingMacro, setSavingMacro] = useState(false)
  const [macroError, setMacroError] = useState<string | null>(null)
  const [macroDiffLines, setMacroDiffLines] = useState<ViewLine[] | null>(null)
  const [macroDiffLoading, setMacroDiffLoading] = useState(false)
  const [collapsed, setCollapsed] = useState(() =>
    localStorage.getItem('sidebar-collapsed') === 'true'
  )
  const [mobileDrawer, setMobileDrawer] = useState<MobileNavGroup | null>(null)
  const { bump } = useDataVersion()
  const activeInstance = useActiveInstance()
  const onFriend = activeInstance.id !== 'self'

  const returnToSelf = () => {
    switchInstance(SELF)
    bump()
    navigate('/')
  }

  const activeMobileGroupKey = useMemo(
    () => groupKeyForPath(location.pathname),
    [location.pathname]
  )

  useEffect(() => addStagingListener(setStaging), [])

  useEffect(() => {
    api.apiConfigSectionGet({ section: 'hostname' }).then((h) => {
      const hostname = typeof h === 'string' && h ? h : 'Routier'
      document.title = hostname === 'Routier' ? 'Routier' : `${hostname} - Routier`
    }).catch(() => { document.title = 'Routier' })
  }, [])

  const handleDiscard = async () => {
    try {
      await api.apiConfigStagingDelete()
      setStaging(false)
      bump()
      toast.success('Pending changes discarded')
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to discard changes')
    }
  }

  const handleApply = async () => {
    setApplying(true)
    try {
      const result = await api.apiConfigApplyPost()
      setShowDiff(false)
      setStaging(false)
      bump()
      if (result.warning) {
        toast.warning(`Applied with warning: ${result.warning}`)
      }
      if (result.snapID) {
        setPendingPoll((k) => k + 1)
      } else {
        toast.success('Configuration applied')
      }
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to apply configuration')
    } finally {
      setApplying(false)
    }
  }

  const handleSaveMacro = async () => {
    const name = macroName.trim()
    if (!name) return
    setSavingMacro(true)
    setMacroError(null)
    try {
      await api.apiMacrosPost({ TypesCreateMacroRequest: { name, description: macroDesc.trim() } })
      toast.success(`Macro "${name}" saved`)
      setShowSaveMacro(false)
      setMacroName('')
      setMacroDesc('')
    } catch (err: unknown) {
      setMacroError((err as Error).message || 'Failed to save macro')
    } finally {
      setSavingMacro(false)
    }
  }

  const handleLogout = () => {
    clearToken()
    navigate('/login')
  }

  const toggleCollapse = () => {
    setCollapsed((c) => {
      localStorage.setItem('sidebar-collapsed', String(!c))
      return !c
    })
  }

  const handleMobileNavTap = (group: MobileNavGroup) => {
    if (group.key === 'dashboard') {
      navigate('/')
      return
    }
    setMobileDrawer(group)
  }

  return (
    <div className="flex h-screen overflow-hidden">
      <aside
        className={cn(
          'hidden md:flex flex-col bg-sidebar text-sidebar-foreground border-r border-sidebar-border shrink-0',
          'transition-[width] duration-200 ease-in-out overflow-hidden',
          collapsed ? 'w-14' : 'w-56',
        )}
      >
        <div
          className={cn(
            'flex items-center border-b border-sidebar-border shrink-0 h-12',
            collapsed ? 'justify-center' : 'px-3 gap-2.5',
          )}
        >
          <Logo className="h-7 w-7 shrink-0" />
          {!collapsed && (
            <span className="text-[15px] font-semibold tracking-tight text-white truncate">
              Routier
            </span>
          )}
        </div>

        <div className={cn('border-b border-sidebar-border py-2', collapsed ? 'px-1' : 'px-2')}>
          <InstanceSwitcher collapsed={collapsed} />
        </div>

        <div className="flex-1 overflow-y-auto overflow-x-hidden py-3">
          <nav className={cn('space-y-4', collapsed ? 'px-1' : 'px-2')}>
            {navGroups.map((group) => (
              <div key={group.label || '__home__'}>
                {group.label && !collapsed && (
                  <div className="px-2 mb-1.5">
                    <SectionLabel className="text-sidebar-foreground/35">{group.label}</SectionLabel>
                  </div>
                )}
                {group.label && collapsed && (
                  <div className="border-t border-sidebar-border/30 mx-1.5 mb-1" />
                )}

                <div className="space-y-0.5">
                  {group.items.map((item) => (
                    <NavLink
                      key={item.to}
                      to={item.to}
                      end={item.exact}
                      title={collapsed ? item.label : undefined}
                      className={({ isActive }) =>
                        cn(
                          'group flex items-center rounded-md text-[13px] font-medium transition-colors w-full',
                          collapsed
                            ? 'justify-center py-2 h-9'
                            : 'gap-2.5 px-2.5 py-1.5',
                          isActive
                            ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                            : 'text-sidebar-foreground/60 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground',
                        )
                      }
                    >
                      <item.icon className="h-3.5 w-3.5 shrink-0 opacity-70 group-hover:opacity-100" />
                      {!collapsed && (
                        <>
                          <span className="truncate">{item.label}</span>
                          <ChevronRight className="ml-auto h-3 w-3 opacity-0 group-hover:opacity-30 transition-opacity" />
                        </>
                      )}
                    </NavLink>
                  ))}
                </div>
              </div>
            ))}
          </nav>
        </div>

        <div className={cn('border-t border-sidebar-border', collapsed ? 'p-1' : 'p-2')}>
          <button
            onClick={toggleCollapse}
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            className={cn(
              'flex items-center rounded-md transition-colors w-full mb-0.5',
              'text-sidebar-foreground/40 hover:text-sidebar-foreground hover:bg-sidebar-accent/50',
              collapsed ? 'justify-center h-8' : 'gap-2.5 px-3 py-1.5 text-xs font-medium',
            )}
          >
            {collapsed
              ? <ChevronRight className="h-3.5 w-3.5" />
              : <><ChevronLeft className="h-3.5 w-3.5" /><span>Collapse</span></>
            }
          </button>

          {collapsed ? (
            <div className="space-y-0.5">
              <NavLink
                to="/settings"
                title="Settings"
                className={({ isActive }) =>
                  cn(
                    'flex items-center justify-center h-8 w-full rounded-md transition-colors',
                    isActive
                      ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                      : 'text-sidebar-foreground/60 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground',
                  )
                }
              >
                <KeyRound className="h-3.5 w-3.5" />
              </NavLink>
              <button
                onClick={toggleTheme}
                title={theme === 'dark' ? 'Light mode' : 'Dark mode'}
                className="flex items-center justify-center h-8 w-full rounded-md text-sidebar-foreground/60 hover:text-sidebar-foreground hover:bg-sidebar-accent/50 transition-colors"
              >
                {theme === 'dark' ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
              </button>
              <button
                onClick={handleLogout}
                title="Sign out"
                className="flex items-center justify-center h-8 w-full rounded-md text-sidebar-foreground/60 hover:text-destructive hover:bg-sidebar-accent/50 transition-colors"
              >
                <LogOut className="h-3.5 w-3.5" />
              </button>
            </div>
          ) : (
            <div className="space-y-0.5">
              <NavLink
                to="/settings"
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-2.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors w-full',
                    isActive
                      ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                      : 'text-sidebar-foreground/60 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground',
                  )
                }
              >
                <KeyRound className="h-3.5 w-3.5" />
                Settings
              </NavLink>
              <Button
                variant="ghost"
                className="w-full justify-start gap-2.5 text-sidebar-foreground/60 hover:text-sidebar-foreground hover:bg-sidebar-accent/50 h-8 px-3 text-sm font-medium"
                onClick={toggleTheme}
              >
                {theme === 'dark' ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
                {theme === 'dark' ? 'Light mode' : 'Dark mode'}
              </Button>
              <Button
                variant="ghost"
                className="w-full justify-start gap-2.5 text-sidebar-foreground/60 hover:text-sidebar-foreground hover:bg-sidebar-accent/50 h-8 px-3 text-sm font-medium"
                onClick={handleLogout}
              >
                <LogOut className="h-3.5 w-3.5" />
                Sign out
              </Button>
            </div>
          )}
        </div>
      </aside>

      <div className="flex-1 flex flex-col overflow-hidden bg-background">
        {onFriend && (
          <div className="flex items-center justify-between gap-3 px-4 py-2 bg-info/10 border-b border-info/30 text-info text-sm shrink-0">
            <div className="flex items-center gap-2">
              <Handshake className="h-3.5 w-3.5" />
              <span className="text-xs font-medium">Configuring {activeInstance.label}</span>
              <span className="text-xs text-info/70 hidden sm:inline">- changes apply to the remote node</span>
            </div>
            <Button
              size="sm"
              variant="outline"
              onClick={returnToSelf}
              className="h-6 gap-1.5 text-xs px-2.5 border-info/40 text-info hover:bg-info/15 hover:text-info"
            >
              <Server className="h-2.5 w-2.5" />
              Return to This Instance
            </Button>
          </div>
        )}
        <AnnouncementBanner />
        {pendingPoll > 0 && (
          <div className="px-4 pt-2 shrink-0 empty:hidden">
            <WatchdogConfirmBar
              pollKey={pendingPoll}
              onKept={bump}
              onRolledBack={bump}
            />
          </div>
        )}
        {staging && (
          <div className="flex items-center justify-between gap-3 px-4 py-2 bg-warning/8 border-b border-warning/20 text-warning text-sm shrink-0">
            <div className="flex items-center gap-2">
              <span className="inline-block h-1.5 w-1.5 rounded-full bg-warning animate-pulse" />
              <span className="text-xs font-medium text-warning/90">Pending changes</span>
              <span className="text-xs text-muted-foreground hidden sm:inline">- apply to activate</span>
            </div>
            <div className="flex items-center gap-2">
              <Button
                size="sm"
                onClick={() => setShowDiff(true)}
                disabled={applying}
                className="h-6 gap-1.5 bg-warning hover:bg-warning/85 text-warning-foreground border-0 text-xs px-2.5"
              >
                <Play className="h-2.5 w-2.5" />Apply
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  setMacroName('')
                  setMacroDesc('')
                  setMacroError(null)
                  setMacroDiffLines(null)
                  setMacroDiffLoading(true)
                  setShowSaveMacro(true)
                  api.apiConfigDiffGet()
                    .then((raw = []) => {
                      setMacroDiffLoading(false)
                      if (!raw.length || raw.every((l) => l.type === 'same')) {
                        setMacroDiffLines([])
                      } else {
                        setMacroDiffLines(withContext(raw))
                      }
                    })
                    .catch(() => setMacroDiffLoading(false))
                }}
                disabled={applying}
                className="h-6 gap-1.5 text-xs px-2.5 hidden sm:inline-flex border-warning/40 text-warning/80 hover:border-warning hover:text-warning hover:bg-warning/10"
              >
                <Boxes className="h-2.5 w-2.5" />
                Save as Macro
              </Button>
              <button
                onClick={handleDiscard}
                disabled={applying}
                className="text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                Discard
              </button>
            </div>
          </div>
        )}

        <main className="flex-1 overflow-auto">
          <div className="p-4 md:p-6">
            <Suspense fallback={<Spinner />}>
              <Outlet />
            </Suspense>
          </div>
        </main>

        <nav className="md:hidden shrink-0 h-14 bg-sidebar border-t border-sidebar-border flex items-stretch">
          {mobileNavGroups.map((group) => {
            const isActive = activeMobileGroupKey === group.key
            return (
              <button
                key={group.key}
                type="button"
                onClick={() => handleMobileNavTap(group)}
                className={cn(
                  'relative flex-1 flex flex-col items-center justify-center gap-0.5 transition-colors',
                  isActive
                    ? 'text-sidebar-primary'
                    : 'text-sidebar-foreground/45 hover:text-sidebar-foreground',
                )}
              >
                <group.icon className="h-5 w-5" />
                <span className="text-[10px] font-medium leading-none">{group.label}</span>
                {isActive && (
                  <span className="absolute bottom-0 inset-x-1/4 h-0.5 rounded-t-full bg-sidebar-primary" />
                )}
              </button>
            )
          })}
        </nav>
      </div>

      {mobileDrawer && (
        <MobileDrawer
          group={mobileDrawer}
          onClose={() => setMobileDrawer(null)}
          theme={theme}
          onToggleTheme={toggleTheme}
          onLogout={handleLogout}
        />
      )}

      {showDiff && (
        <DiffModal
          applying={applying}
          onApply={handleApply}
          onClose={() => setShowDiff(false)}
        />
      )}

      {showSaveMacro && (
        <Dialog.Root open onOpenChange={(o) => { if (!o) setShowSaveMacro(false) }}>
          <Dialog.Portal>
            <Dialog.Overlay className="fixed inset-0 z-50 bg-black/50" />
            <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-full max-w-2xl max-h-[85vh] -translate-x-1/2 -translate-y-1/2 rounded-xl border border-border bg-card shadow-lg flex flex-col overflow-hidden">
              <div className="p-6 pb-4 border-b border-border space-y-1 shrink-0">
                <Dialog.Title className="text-base font-semibold">Save as Macro</Dialog.Title>
                <Dialog.Description className="text-sm text-muted-foreground">
                  Capture the pending changes as a named, reusable macro.
                </Dialog.Description>
              </div>

              <div className="flex-1 overflow-y-auto p-6 space-y-4">
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-muted-foreground">Name</label>
                    <Input
                      value={macroName}
                      onChange={(e) => setMacroName(e.target.value)}
                      placeholder="e.g. enable-wireguard"
                      autoFocus
                      onKeyDown={(e) => { if (e.key === 'Enter' && macroName.trim()) handleSaveMacro() }}
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-muted-foreground">Description (optional)</label>
                    <Input
                      value={macroDesc}
                      onChange={(e) => setMacroDesc(e.target.value)}
                      placeholder="What does this macro do?"
                    />
                  </div>
                </div>
                {macroError && <p className="text-xs text-destructive">{macroError}</p>}

                <div className="space-y-2">
                  <p className="text-xs font-medium text-muted-foreground">Changes to capture</p>
                  {macroDiffLoading && (
                    <div className="flex items-center justify-center h-20 rounded border border-border bg-muted/20">
                      <RefreshCw className="h-4 w-4 animate-spin text-muted-foreground" />
                    </div>
                  )}
                  {!macroDiffLoading && macroDiffLines !== null && macroDiffLines.length === 0 && (
                    <p className="text-xs text-muted-foreground italic text-center py-4">
                      No differences detected.
                    </p>
                  )}
                  {!macroDiffLoading && macroDiffLines !== null && macroDiffLines.length > 0 && (
                    <div className="overflow-auto max-h-64 rounded border border-border p-3 bg-muted/30">
                      <DiffView lines={macroDiffLines} />
                    </div>
                  )}
                </div>
              </div>

              <div className="p-4 border-t border-border flex justify-end gap-2 shrink-0">
                <Button variant="outline" onClick={() => setShowSaveMacro(false)} disabled={savingMacro}>Cancel</Button>
                <Button onClick={handleSaveMacro} disabled={savingMacro || !macroName.trim()} className="gap-2">
                  {savingMacro && <RefreshCw className="h-3.5 w-3.5 animate-spin" />}
                  {savingMacro ? 'Saving…' : 'Save macro'}
                </Button>
              </div>
            </Dialog.Content>
          </Dialog.Portal>
        </Dialog.Root>
      )}
    </div>
  )
}
