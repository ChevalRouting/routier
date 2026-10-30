import { useHasParam } from '@/lib/useHasParam'
import { DraftControls } from '@/components/DraftControls'
import type { ConfigNftablesConfig, TypesNftValidateResult as NftValidateResult, RenderNftVar as NftVar } from '@/api'
import { ChainPane } from '@/components/firewall/ChainPane'
import { chainRuleCount, CodeArea, DRAFT_KEY, EDITOR_HEIGHT, fromApi, NftablesConfig, NftChain, OWNED_CHAINS, parseUserDefines, toApi, ValidationStrip } from '@/components/firewall/shared'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { Badge, PageHeader, SectionNav, Spinner, useTabState } from 'cheval-ui'
import { useEffect, useMemo, useState } from 'react'

export default function Firewall() {
  const { data, isLoading, error } = useFetch(() => api.apiConfigSectionGet({ section: 'nftables' }))
  const { data: rawVars } = useFetch<NftVar[]>(() => api.apiConfigNftablesVarsGet())
  const [cfg, setCfg] = useState<NftablesConfig>({})
  const [saved, setSaved] = useState<NftablesConfig>({})
  const [initialized, setInitialized] = useState(false)
  const { isDirty, markDirty, save, saving, reset } = usePageSave('nftables')
  useDataRefresh(() => {
    try { localStorage.removeItem(DRAFT_KEY) } catch {  }
    reset()
    setInitialized(false)
  })
  const [section, setSection] = useTabState<string>('firewall', 'input')
  const deepLinked = useHasParam('firewall')
  const [validation, setValidation] = useState<NftValidateResult | null>(null)
  const [validating, setValidating] = useState(false)

  const vars = useMemo(() => rawVars ?? [], [rawVars])
  const varNames = useMemo(() => vars.map((v) => v.name), [vars])

  const acVars = useMemo<NftVar[]>(() => {
    const userDefines = parseUserDefines(cfg.defines ?? '')
    const seen = new Set(vars.map((v) => v.name))
    return [...vars, ...userDefines.filter((v) => !seen.has(v.name))]
      .sort((a, b) => a.name.localeCompare(b.name))
  }, [vars, cfg.defines])

  useEffect(() => {
    if (!isLoading && !error && !initialized) {
      const loaded = fromApi(data)
      let next = loaded
      try {
        const raw = localStorage.getItem(DRAFT_KEY)
        if (raw) { next = JSON.parse(raw) as NftablesConfig; markDirty() }
      } catch {  }
      setSaved(loaded)
      setCfg(next)
      setInitialized(true)
    }
  }, [data, isLoading, error, initialized, markDirty])

  const validatePayload = useMemo(() => (initialized ? JSON.stringify(toApi(cfg)) : ''), [cfg, initialized])
  useEffect(() => {
    if (!initialized) return
    const ctrl = new AbortController()
    const id = setTimeout(async () => {
      setValidating(true)
      try {
        setValidation(await api.apiConfigNftablesValidatePost(
          { ConfigNftablesConfig: JSON.parse(validatePayload) as ConfigNftablesConfig },
          { signal: ctrl.signal },
        ))
      } catch {  } finally {
        setValidating(false)
      }
    }, 750)
    return () => { clearTimeout(id); ctrl.abort() }
  }, [validatePayload, initialized])

  const update = (next: NftablesConfig) => {
    setCfg(next)
    markDirty()
    try { localStorage.setItem(DRAFT_KEY, JSON.stringify(next)) } catch {  }
  }
  const setChain = (name: string, ch: NftChain) =>
    update({ ...cfg, chains: { ...cfg.chains, [name]: ch } })

  const handleSave = async () => {
    let draft: string | null = null
    try { draft = localStorage.getItem(DRAFT_KEY) } catch {  }
    const snapshot = cfg
    if (await save(toApi(snapshot))) {
      setSaved(snapshot)
      try {
        if (localStorage.getItem(DRAFT_KEY) === draft) localStorage.removeItem(DRAFT_KEY)
      } catch {  }
    }
  }

  const handleDiscard = () => {
    try { localStorage.removeItem(DRAFT_KEY) } catch {  }
    reset()
    setCfg(saved)
  }

  if (isLoading) return <Spinner />

  const items = [
    { key: 'defines', label: 'Defines' },
    ...OWNED_CHAINS.map((c) => {
      const n = chainRuleCount(cfg.chains?.[c.name])
      return { key: c.name, label: c.name.charAt(0).toUpperCase() + c.name.slice(1), badge: n > 0 ? n : undefined }
    }),
  ]

  const chainMeta = OWNED_CHAINS.find((c) => c.name === section)
  const errorLinesFor = (sec: string) =>
    (validation?.errors ?? []).filter((e) => e.section === sec && e.line > 0).map((e) => e.line)

  return (
    <div className="space-y-6">
      <PageHeader title="Firewall"
        action={<DraftControls isDirty={isDirty} saving={saving} onClick={handleSave} onCancel={handleDiscard} />} />

      <SectionNav items={items} active={section} onChange={setSection} initiallyEntered={deepLinked}>
        {section === 'defines' ? (
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <span className="font-mono font-semibold text-sm">defines</span>
              <Badge variant="secondary" className="text-xs">variables</Badge>
            </div>
            <CodeArea
              value={cfg.defines ?? ''}
              onChange={(v) => update({ ...cfg, defines: v })}
              placeholder={'define my_set = { 10.0.0.0/8, 2001:db8::/48 }'}
              height={EDITOR_HEIGHT}
              vars={acVars}
              errorLines={errorLinesFor('defines')} />
            <p className="text-xs text-muted-foreground">
              Raw nft <code>define</code> lines, emitted before the ruleset
            </p>
          </div>
        ) : chainMeta ? (
          <ChainPane
            meta={chainMeta}
            chain={cfg.chains?.[section] ?? {}}
            onChange={(ch) => setChain(section, ch)}
            varNames={varNames}
            acVars={acVars}
            errorLines={errorLinesFor(chainMeta.name)} />
        ) : null}
      </SectionNav>

      <ValidationStrip validating={validating} result={validation} />
    </div>
  )
}
