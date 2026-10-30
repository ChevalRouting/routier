import { DraftControls } from '@/components/DraftControls'
import { EMPTY_SAVE, type TabSaveState } from '@/components/system-config/shared'
import { CertificateSettings } from '@/components/certificates/CertificateSettings'
import { PageHeader } from 'cheval-ui'
import { useState } from 'react'

export default function Certificates() {
  const [saveState, setSaveState] = useState<TabSaveState>(EMPTY_SAVE)

  return (
    <div className="space-y-6">
      <PageHeader
        title="Certificates"
        action={<DraftControls isDirty={saveState.isDirty} saving={saveState.saving} onClick={saveState.save} />}
      />
      <CertificateSettings onStateChange={setSaveState} />
    </div>
  )
}
