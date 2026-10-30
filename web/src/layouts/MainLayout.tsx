import PageErrorBoundary from '@/components/PageErrorBoundary'
import { DraftActionsContext, type DraftAction } from '@/lib/draftActions'
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom'
import { navGroups } from '@/lib/navigation'
import { Suspense, useEffect, useState, useMemo, useCallback } from 'react'
import { useDataVersion } from '@/lib/dataVersion'
import { toast } from 'sonner'
import { Button, cn, MobileNav, SectionLabel, Spinner, type MobileTab } from 'cheval-ui'
import { addStagingListener, api, ApiError } from '@/lib/client'
import DiffModal from '@/components/DiffModal'
import ApplyFailureModal from '@/components/ApplyFailureModal'
import { applyFailureFromResult, applyFailureFromError, type ApplyFailure } from '@/lib/applyFailure'
import { WatchdogConfirmBar } from '@/components/WatchdogConfirmBar'
import Logo from '@/components/Logo'
import { AnnouncementBanner } from '@/components/AnnouncementBanner'
import { InstanceSwitcher, useActiveInstance } from '@/components/InstanceSwitcher'
import { DraftBanner } from '@/components/shell/DraftBanner'
import { MainMenu } from '@/components/shell/MainMenu'
import { SaveMacroDialog } from '@/components/shell/SaveMacroDialog'
import { SELF, switchInstance } from '@/lib/instance'
import { useScrollRecovery } from '@/lib/useScrollRecovery'
import { LayoutDashboard, Network, Settings2, Activity, SlidersHorizontal, Handshake, ChevronLeft, ChevronRight, MoreHorizontal } from 'lucide-react'

const menuPath = (key: string) => `/menu/${key}`


function groupKeyForPath(pathname: string): string {
  if (pathname.startsWith('/menu/')) return pathname.slice(6)
  if (pathname === '/settings') return 'more'
  if (['/hostname', '/ssh', '/sysctl'].includes(pathname)) return 'system'
  if (['/traffic', '/logs', '/neighbors'].some(p => pathname === p || pathname.startsWith(p + '/'))) return 'monitor'
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

function editableTarget(target: EventTarget | null): boolean {
  return target instanceof HTMLElement && (target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName))
}

export default function MainLayout() {
  const [draftActions, setDraftActions] = useState<Record<string, DraftAction>>({})
  const [editorRevision, setEditorRevision] = useState(0)
  const registerDraftAction = useCallback((id: string, action: DraftAction | null) => {
    setDraftActions((current) => {
      const next = { ...current }
      if (action) next[id] = action
      else delete next[id]
      return next
    })
  }, [])
  const edits = Object.values(draftActions).filter((action) => action.isDirty || action.saving)
  const editing = edits.length > 0
  const savingDraft = edits.some((action) => action.saving)
  const saveDisabled = edits.some((action) => action.disabled)
  const saveDraft = useCallback(async () => {
    for (const action of edits) await action.save()
  }, [edits])
  const discardDraft = () => {
    if (edits.some((action) => !action.discard)) setEditorRevision((revision) => revision + 1)
    else edits.forEach((action) => action.discard?.())
  }
  const navigate = useNavigate()
  const location = useLocation()
  const [staging, setStaging] = useState(false)
  const [showDiff, setShowDiff] = useState(false)
  const [applying, setApplying] = useState(false)
  const [applyFailure, setApplyFailure] = useState<ApplyFailure | null>(null)
  const [showApplyLogs, setShowApplyLogs] = useState(false)
  const [pendingPoll, setPendingPoll] = useState(0)
  const [confirmationPending, setConfirmationPending] = useState<boolean | null>(null)
  const [showSaveMacro, setShowSaveMacro] = useState(false)
  const [collapsed, setCollapsed] = useState(() =>
    localStorage.getItem('sidebar-collapsed') === 'true'
  )
  const [hostname, setHostname] = useState('Routier')
  const { bump } = useDataVersion()
  const activeInstance = useActiveInstance()
  const onFriend = activeInstance.id !== 'self'
  useEffect(() => { setApplyFailure(null); setShowApplyLogs(false); setConfirmationPending(null); setShowDiff(false) }, [activeInstance.id])

  const returnToSelf = () => {
    switchInstance(SELF)
    bump()
    navigate('/')
  }

  const activeMobileGroupKey = useMemo(() => groupKeyForPath(location.pathname), [location.pathname])

  useScrollRecovery(location.pathname)

  useEffect(() => addStagingListener((state) => {
    setStaging(state.pending)
  }), [])

  useEffect(() => {
    if (location.pathname.startsWith('/simple')) navigate('/', { replace: true })
  }, [location.pathname, navigate])

  useEffect(() => {
    api.apiConfigSectionGet({ section: 'hostname' }).then((h) => {
      const name = typeof h === 'string' && h ? h : 'Routier'
      setHostname(name)
      document.title = name === 'Routier' ? 'Routier' : `${name} - Routier`
    }).catch(() => { document.title = 'Routier' })
  }, [activeInstance.id])

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's' && editing) {
        event.preventDefault()
        if (!savingDraft && !saveDisabled) void saveDraft()
      }
      if (event.altKey && event.key === 'ArrowLeft' && !editableTarget(event.target)) {
        event.preventDefault()
        navigate(-1)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [editing, savingDraft, saveDisabled, saveDraft, navigate])

  const handleDiscard = async () => {
    try {
      await api.apiConfigStagingDelete()
      setApplyFailure(null)
      setShowApplyLogs(false)
      setStaging(false)
      bump()
      toast.success('Draft discarded')
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to discard the draft')
    }
  }

  const handleApply = async () => {
    if (confirmationPending !== false || applying) return
    setApplying(true)
    try {
      const result = await api.apiConfigApplyPost()
      setShowDiff(false)
      const failure = applyFailureFromResult(result)
      if (failure) {
        setApplyFailure(failure)
        return
      }
      setApplyFailure(null)
      setShowApplyLogs(false)
      setStaging(false)
      bump()
      if (result.warning) {
        toast.warning(`Applied with warning: ${result.warning}`)
      }
      if (result.snapID) {
        setConfirmationPending(true)
        setPendingPoll((k) => k + 1)
      } else {
        toast.success('Configuration applied')
      }
    } catch (err: unknown) {
      setShowDiff(false)
      if (err instanceof ApiError && err.status === 409) {
        setConfirmationPending(true)
        setPendingPoll((k) => k + 1)
        return
      }
      setApplyFailure(applyFailureFromError(err))
    } finally {
      setApplying(false)
    }
  }

  const handleKept = () => {
    setApplyFailure(null)
    setShowApplyLogs(false)
    setShowDiff(false)
    setPendingPoll(0)
    bump()
  }

  const toggleCollapse = () => {
    setCollapsed((c) => {
      localStorage.setItem('sidebar-collapsed', String(!c))
      return !c
    })
  }

  const mobileTabs: MobileTab[] = [
    { key: 'dashboard', label: 'Home', icon: LayoutDashboard, to: '/', exact: true },
    { key: 'network', label: 'Network', icon: Network, to: menuPath('network'), items: navGroups[1].items },
    { key: 'system', label: 'System', icon: Settings2, to: menuPath('system'), items: navGroups[2].items },
    { key: 'monitor', label: 'Monitor', icon: Activity, to: menuPath('monitor'), items: navGroups[3].items },
    { key: 'more', label: 'More', icon: MoreHorizontal, to: menuPath('more'), items: [...navGroups[4].items, { to: '/settings', label: 'Preferences', icon: SlidersHorizontal }] },
  ]

  return (
    <DraftActionsContext.Provider value={registerDraftAction}>
    <div className="app-shell flex overflow-hidden">
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
          {!collapsed && (
            <>
              <Logo className="h-7 w-7 shrink-0" />
              <span className="flex-1 truncate text-sm font-semibold text-sidebar-foreground">Routier</span>
            </>
          )}
          <MainMenu className="h-8 w-8 text-sidebar-foreground/70 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground" />
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
                    <SectionLabel className="text-sidebar-foreground/50">{group.label}</SectionLabel>
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
                          'flex h-8 items-center rounded-md text-sm font-medium transition-colors w-full',
                          collapsed ? 'justify-center' : 'gap-2.5 px-2.5',
                          isActive
                            ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                            : 'text-sidebar-foreground/70 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground',
                        )
                      }
                    >
                      <item.icon className="h-4 w-4 shrink-0" />
                      {!collapsed && <span className="truncate">{item.label}</span>}
                    </NavLink>
                  ))}
                </div>
              </div>
            ))}
          </nav>
        </div>

        <div className={cn('border-t border-sidebar-border', collapsed ? 'p-1' : 'p-2')}>
          <button
            type="button"
            onClick={toggleCollapse}
            aria-label={collapsed ? 'Expand Sidebar' : 'Collapse Sidebar'}
            title={collapsed ? 'Expand Sidebar' : 'Collapse Sidebar'}
            className={cn(
              'flex h-8 items-center rounded-md text-sm font-medium transition-colors w-full',
              'text-sidebar-foreground/70 hover:text-sidebar-foreground hover:bg-sidebar-accent/60',
              collapsed ? 'justify-center' : 'gap-2.5 px-2.5',
            )}
          >
            {collapsed
              ? <ChevronRight className="h-4 w-4" />
              : <><ChevronLeft className="h-4 w-4" /><span>Collapse</span></>
            }
          </button>
        </div>
      </aside>

      <div className="min-h-0 min-w-0 flex-1 flex flex-col overflow-hidden bg-background">
        <header className="md:hidden shrink-0 safe-top border-b border-border bg-sidebar">
          <div className="h-12 flex items-center gap-2 px-3">
            {location.pathname !== '/' && !location.pathname.startsWith('/menu/') ? (
              <button
                type="button"
                onClick={() => navigate(-1)}
                aria-label="Back"
                title="Back"
                className="-ml-1 flex items-center justify-center h-10 w-10 rounded-md text-sidebar-foreground/70 hover:text-sidebar-foreground hover:bg-sidebar-accent/50 transition-colors"
              >
                <ChevronLeft className="h-5 w-5" />
              </button>
            ) : (
              <Logo className="h-7 w-7 shrink-0" />
            )}
            <span className="truncate text-base font-semibold">
              {hostname || activeInstance.label}
            </span>
          </div>
        </header>
        {onFriend && (
          <div className="flex shrink-0 flex-wrap items-center gap-x-3 gap-y-2 border-b border-info/30 bg-info/10 px-4 py-2 text-sm text-info">
            <Handshake className="h-4 w-4 shrink-0" />
            <p className="min-w-0 flex-1 truncate font-medium">Configuring {activeInstance.label}</p>
            <Button variant="outline" onClick={returnToSelf}>Return to This Instance</Button>
          </div>
        )}
        <AnnouncementBanner />
        <WatchdogConfirmBar
          key={activeInstance.id}
          className="shrink-0 rounded-none border-x-0 border-t-0 bg-warning/10 px-4"
          pollKey={pendingPoll}
          onPendingChange={setConfirmationPending}
          onKept={handleKept}
          onRolledBack={() => { setPendingPoll(0); bump() }}
        />
        <DraftBanner
          editing={editing}
          savingDraft={savingDraft}
          saveDisabled={saveDisabled}
          staging={staging}
          applying={applying}
          applyBlocked={confirmationPending !== false}
          applyFailure={applyFailure}
          canSaveMacro
          onSaveDraft={saveDraft}
          onDiscardEdits={discardDraft}
          onDiscardDraft={handleDiscard}
          onApply={() => setShowDiff(true)}
          onSaveMacro={() => setShowSaveMacro(true)}
          onShowLogs={() => setShowApplyLogs(true)}
        />

        {showApplyLogs && applyFailure && <ApplyFailureModal failure={applyFailure} onClose={() => setShowApplyLogs(false)} />}

        <main className="min-h-0 flex-1 overflow-auto">
          <div className="p-4 pb-8 md:p-6" key={`${activeInstance.id}:${editorRevision}`}>
            <PageErrorBoundary key={location.pathname}>
              <Suspense fallback={<Spinner />}>
                <Outlet />
              </Suspense>
            </PageErrorBoundary>
          </div>
        </main>

        <MobileNav className="mobile-nav" tabs={mobileTabs} activeKey={activeMobileGroupKey} subNav={false} />
      </div>

      {showDiff && (
        <DiffModal
          applying={applying}
          applyBlocked={confirmationPending !== false}
          onApply={handleApply}
          onClose={() => setShowDiff(false)}
        />
      )}

      <SaveMacroDialog open={showSaveMacro} onClose={() => setShowSaveMacro(false)} />
    </div>
    </DraftActionsContext.Provider>
  )
}
