import type { TlscertCertificate as ServedCertificate } from '@/api'
import { Button, Row, StatusChip, type StatusIntent } from 'cheval-ui'

type CertificateRowShape = {
  certificate: ServedCertificate
  acme: boolean
  renewing?: boolean
  onRenew?: (domain: string) => void
}

type CertificateChip = { intent: StatusIntent; label: string }

const DAY = 24 * 60 * 60 * 1000

const SOURCE_LABELS: Record<string, string> = { 'self-signed': 'Self-Signed', file: 'Custom', acme: 'ACME' }

function chipFor(certificate: ServedCertificate): CertificateChip {
  if (!certificate.ready && certificate.error) return { intent: 'danger', label: 'Failed' }
  if (!certificate.ready) return { intent: 'neutral', label: 'Pending' }
  const remaining = certificate.not_after ? new Date(certificate.not_after).getTime() - Date.now() : Infinity
  if (remaining <= 0) return { intent: 'danger', label: 'Expired' }
  if (remaining < 14 * DAY) return { intent: 'warning', label: 'Expiring' }
  return { intent: 'success', label: 'Valid' }
}

export function CertificateRow({ certificate, acme, renewing, onRenew }: CertificateRowShape) {
  const chip = chipFor(certificate)
  const expires = certificate.not_after ? ` · expires ${new Date(certificate.not_after).toLocaleDateString()}` : ''
  const detail = certificate.ready ? `${certificate.issuer ?? ''}${expires}` : (certificate.error || 'ACME issuance in progress')
  const title = certificate.name !== 'fallback' ? certificate.name : acme ? 'Fallback' : 'Default'
  const source = SOURCE_LABELS[certificate.source] ?? certificate.source
  const handleRenew = () => onRenew?.(certificate.name)
  return (
    <Row title={<span className="font-mono">{title}</span>} subtitle={`${source} · ${detail}`}>
      <div className="flex items-center gap-2">
        <StatusChip intent={chip.intent} label={chip.label} />
        {onRenew && certificate.source === 'acme' && (
          <Button variant="outline" size="sm" onClick={handleRenew} disabled={renewing}>{renewing ? 'Renewing…' : 'Renew'}</Button>
        )}
      </div>
    </Row>
  )
}
