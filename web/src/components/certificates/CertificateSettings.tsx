import type { ConfigWeb as WebConfig, TlscertStatus as CertificateStatus } from '@/api'
import type { OnStateChange } from '@/components/system-config/shared'
import { CertificateLog } from '@/components/certificates/CertificateLog'
import { CertificateRow } from '@/components/certificates/CertificateRow'
import { PemField } from '@/components/certificates/PemField'
import { api } from '@/lib/client'
import { useDataRefresh } from '@/lib/dataVersion'
import { useFetch } from '@/lib/useFetch'
import { usePageSave } from '@/lib/usePageSave'
import { checkPort } from '@/lib/validate'
import { EntryRow, NoticeBanner, PreferencesGroup, Row, Segmented, Spinner, TagInput } from 'cheval-ui'
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'

type CertificateSettingsShape = { onStateChange: OnStateChange }

type Mode = 'self-signed' | 'files' | 'acme'

type ModeOption = { value: Mode; label: string }

type StoredFiles = { certFile: string; keyFile: string }

const MODE_OPTIONS: ModeOption[] = [
  { value: 'self-signed', label: 'Self-Signed' },
  { value: 'files', label: 'Custom' },
  { value: 'acme', label: 'ACME' },
]

const LETS_ENCRYPT = 'https://acme-v02.api.letsencrypt.org/directory'

const domainPattern = /^(?=.{1,253}$)([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9-]{2,63}\.?$/i

function modeOf(web: WebConfig | null): Mode {
  if (web?.tls?.acme) return 'acme'
  if (web?.tls?.cert_file) return 'files'
  return 'self-signed'
}

function validDomain(value: string) {
  return domainPattern.test(value) ? null : 'Use a fully qualified name'
}

async function storePEM(suffix: string, content: string): Promise<string> {
  const name = `web-${Date.now().toString(36)}.${suffix}`
  await api.apiWebTlsFilesNamePut({ name, WebtlsFileRequest: { content } })
  return name
}

export function CertificateSettings({ onStateChange }: CertificateSettingsShape) {
  const { data, isLoading } = useFetch<WebConfig | null>(() => api.apiConfigSectionGet({ section: 'web' }) as Promise<WebConfig | null>)
  const status = useFetch<CertificateStatus>(() => api.apiWebTlsGet())
  const [mode, setMode] = useState<Mode>('self-signed')
  const [stored, setStored] = useState<StoredFiles>({ certFile: '', keyFile: '' })
  const [storedCertPEM, setStoredCertPEM] = useState('')
  const [certPEM, setCertPEM] = useState('')
  const [keyPEM, setKeyPEM] = useState('')
  const [domains, setDomains] = useState<string[]>([])
  const [email, setEmail] = useState('')
  const [directory, setDirectory] = useState('')
  const [httpPort, setHttpPort] = useState('')
  const [caPem, setCaPem] = useState('')
  const [eabKid, setEabKid] = useState('')
  const [eabKey, setEabKey] = useState('')
  const [initialized, setInitialized] = useState(false)
  const [renewing, setRenewing] = useState('')
  const { isDirty, markDirty, save, saving } = usePageSave('web')
  useDataRefresh(() => setInitialized(false))

  useEffect(() => {
    if (isLoading || initialized) return
    const tls = data?.tls
    setMode(modeOf(data)); setStored({ certFile: tls?.cert_file ?? '', keyFile: tls?.key_file ?? '' }); setKeyPEM('')
    setDomains(tls?.acme?.domains ?? []); setEmail(tls?.acme?.email ?? ''); setDirectory(tls?.acme?.directory ?? '')
    setHttpPort(tls?.acme?.http_port ? String(tls.acme.http_port) : '')
    setCaPem(tls?.acme?.ca_pem ?? ''); setEabKid(tls?.acme?.eab_kid ?? ''); setEabKey(tls?.acme?.eab_key ?? '')
    setInitialized(true)
  }, [data, isLoading, initialized])

  useEffect(() => {
    if (!stored.certFile) { setStoredCertPEM(''); setCertPEM(''); return }
    let active = true
    api.apiWebTlsFilesNameGet({ name: stored.certFile })
      .then((content) => { if (active) { setStoredCertPEM(content as unknown as string); setCertPEM(content as unknown as string) } })
      .catch(() => { if (active) { setStoredCertPEM(''); setCertPEM('') } })
    return () => { active = false }
  }, [stored.certFile])

  const acmeActive = modeOf(data) === 'acme'
  const reloadStatus = status.reload
  useEffect(() => {
    if (!acmeActive) return
    const timer = window.setInterval(() => { void reloadStatus() }, 5000)
    return () => window.clearInterval(timer)
  }, [acmeActive, reloadStatus])

  const handleRenew = useCallback(async (domain: string) => {
    setRenewing(domain)
    try {
      await api.apiWebTlsRenewPost({ WebtlsRenewRequest: { domain } })
      toast.success(`Renewed the certificate for ${domain}`)
    } catch (e) {
      toast.error(`Failed to renew ${domain}: ${(e as Error).message}`)
    } finally {
      setRenewing('')
      void reloadStatus()
    }
  }, [reloadStatus])

  const keepingFiles = mode === 'files' && stored.certFile !== '' && stored.keyFile !== ''

  const buildPayload = useCallback(async (): Promise<WebConfig | null> => {
    if (mode === 'files') {
      const certFile = certPEM === storedCertPEM && stored.certFile ? stored.certFile : await storePEM('crt', certPEM)
      const keyFile = keyPEM.trim() ? await storePEM('key', keyPEM) : stored.keyFile
      setStored({ certFile, keyFile }); setKeyPEM('')
      return { tls: { cert_file: certFile, key_file: keyFile } }
    }
    if (mode !== 'acme') return null
    const port = parseInt(httpPort)
    return {
      tls: {
        acme: {
          domains,
          email: email || undefined,
          directory: directory || undefined,
          http_port: port > 0 ? port : undefined,
          ca_pem: caPem.trim() ? caPem : undefined,
          eab_kid: eabKid || undefined,
          eab_key: eabKey || undefined,
        },
      },
    }
  }, [mode, certPEM, storedCertPEM, stored, keyPEM, domains, email, directory, httpPort, caPem, eabKid, eabKey])

  const handleSave = useCallback(async () => {
    try {
      return await save(await buildPayload())
    } catch (e) {
      toast.error(`Failed to store the certificate: ${(e as Error).message}`)
      return false
    }
  }, [save, buildPayload])
  useEffect(() => { onStateChange({ isDirty, saving, save: handleSave }) }, [isDirty, saving, handleSave, onStateChange])

  const edit = <T,>(setter: (value: T) => void) => (value: T) => { setter(value); markDirty() }

  if (isLoading) return <Spinner />

  return (
    <div className="max-w-3xl space-y-6">
      {status.latest?.error && <NoticeBanner intent="danger">{status.latest.error}</NoticeBanner>}

      <PreferencesGroup title="Certificate" description="Served by the web UI and API">
        <Row title="Source">
          <Segmented value={mode} onChange={edit(setMode)} options={MODE_OPTIONS} />
        </Row>
        {mode === 'self-signed' && <Row subtitle="Generated in /var/lib/routier; browsers warn until it is trusted" />}
        {mode === 'files' && (
          <>
            <PemField
              id="web-tls-cert"
              label="Certificate Chain"
              value={certPEM}
              placeholder="-----BEGIN CERTIFICATE-----"
              onChange={edit(setCertPEM)}
            />
            <PemField
              id="web-tls-key"
              label="Private Key"
              value={keyPEM}
              placeholder={keepingFiles ? 'Leave empty to keep the current key' : '-----BEGIN PRIVATE KEY-----'}
              help={keepingFiles ? 'Files changed on disk are reloaded automatically' : undefined}
              onChange={edit(setKeyPEM)}
            />
          </>
        )}
      </PreferencesGroup>

      {mode === 'acme' && (
        <PreferencesGroup title="ACME" description="Certificates are requested on apply and renewed 30 days before expiry">
          <EntryRow
            title="Directory"
            value={directory}
            placeholder={LETS_ENCRYPT}
            help={!directory || directory === LETS_ENCRYPT ? "Let's Encrypt" : undefined}
            onChange={(e) => edit(setDirectory)(e.target.value)}
            className="font-mono"
          />
          <EntryRow title="Contact Email" type="email" value={email} onChange={(e) => edit(setEmail)(e.target.value)} />
          <div className="flex min-w-0 flex-col gap-1.5 px-4 py-3">
            <span className="text-sm font-medium">Domains</span>
            <TagInput values={domains} mono placeholder="router.example.com" validate={validDomain} onChange={edit(setDomains)} />
          </div>
          <EntryRow
            title="Challenge Port"
            value={httpPort}
            placeholder="80"
            help="http-01 challenges; the ACME server must reach it on port 80"
            onChange={(e) => edit(setHttpPort)(e.target.value)}
            className="font-mono"
            error={checkPort(httpPort)}
          />
          <PemField
            id="web-acme-ca"
            label="Directory CA"
            value={caPem}
            placeholder="Optional PEM bundle for a private ACME server"
            onChange={edit(setCaPem)}
          />
          <EntryRow title="EAB Key ID" value={eabKid} onChange={(e) => edit(setEabKid)(e.target.value)} className="font-mono" />
          <EntryRow title="EAB HMAC Key" type="password" autoComplete="off" value={eabKey} onChange={(e) => edit(setEabKey)(e.target.value)} className="font-mono" />
        </PreferencesGroup>
      )}

      {status.latest && (
        <PreferencesGroup title="Available Certificates">
          {status.latest.certificates.map((certificate) => (
            <CertificateRow
              key={certificate.name}
              certificate={certificate}
              acme={status.latest?.mode === 'acme'}
              renewing={renewing !== ''}
              onRenew={handleRenew}
            />
          ))}
        </PreferencesGroup>
      )}

      {status.latest && acmeActive && <CertificateLog entries={status.latest.log} />}
    </div>
  )
}
