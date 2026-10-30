import { api } from '@/lib/client'
import { Button, SectionLabel, Separator } from 'cheval-ui'
import { useRef, useState } from 'react'
import { toast } from 'sonner'

type BackupTabShape = { onChanged: () => void }

export function BackupTab({ onChanged }: BackupTabShape) {
  const fileRef = useRef<HTMLInputElement>(null)
  const [exporting, setExporting] = useState(false)
  const [importing, setImporting] = useState(false)

  const doExport = async () => {
    setExporting(true)
    try {
      const res = await api.apiBackupExportGetRaw()
      const blob = await res.raw.blob()
      const disposition = res.raw.headers.get('Content-Disposition') ?? ''
      const match = /filename=([^;]+)/.exec(disposition)
      const filename = match ? match[1].trim() : `routier-backup-${Date.now()}.bin`
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
    } catch (e) {
      toast.error(`Export failed: ${(e as Error).message}`)
    } finally {
      setExporting(false)
    }
  }

  const doImport = async (file: File) => {
    setImporting(true)
    try {
      const form = new FormData()
      form.append('file', file)
      const res = await api.apiBackupImportPostRaw({ method: 'POST', body: form })
      const parsed = await res.value()
      toast.success('Backup imported and applied')
      if (parsed.result?.warning) toast.warning(parsed.result.warning)
      onChanged()
    } catch (e) {
      toast.error(`Import failed: ${(e as Error).message}`)
    } finally {
      setImporting(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  return (
    <div className="max-w-3xl space-y-5">
      <div className="space-y-2">
        <SectionLabel>Export</SectionLabel>
        <p className="text-sm text-muted-foreground">
          Download a <code className="text-xs bg-muted px-1 py-0.5 rounded">.bin</code> archive (zstd) of the config and referenced files
          (nftables, WireGuard keys, template overrides). It contains private keys</p>
        <Button onClick={doExport} disabled={exporting} className="gap-2">
          {exporting ? 'Exporting…' : 'Export Backup'}
        </Button>
      </div>
      <Separator />
      <div className="space-y-2">
        <SectionLabel>Import</SectionLabel>
        <p className="text-sm text-muted-foreground">Restores and applies a <code>.bin</code> archive; confirm within the watchdog window to keep it</p>
        <input ref={fileRef} type="file" accept=".bin" className="hidden"
          onChange={(e) => { const f = e.target.files?.[0]; if (f) doImport(f) }} />
        <Button variant="outline" onClick={() => fileRef.current?.click()} disabled={importing} className="gap-2">
          {importing ? 'Importing…' : 'Import Backup…'}
        </Button>
      </div>
    </div>
  )
}
