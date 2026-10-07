import { PortForwardEditor } from '@/components/simple/PortForwardEditor'
import type { PortForward } from '@/components/simple/types'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import { PageHeader, SaveButton, Spinner } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

export default function SimplePortForwards() {
  const { data, isLoading, reload } = useFetch<PortForward[]>(() =>
    api.apiConfigSectionGet({ section: 'port_forwards' }) as Promise<PortForward[]>,
  )
  const [forwards, setForwards] = useState<PortForward[]>([])
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (data) {
      setForwards(data)
      setDirty(false)
    }
  }, [data])

  if (isLoading) return <Spinner />

  const save = async () => {
    setSaving(true)
    try {
      await api.apiConfigSectionPut({ section: 'port_forwards', body: forwards as unknown as object })
      toast.success('Port forwards staged')
      setDirty(false)
      reload()
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save port forwards')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Port forwards" description="Let devices on the internet reach a service on your network." />
      <div className="max-w-3xl space-y-5">
        <PortForwardEditor forwards={forwards} onChange={(next) => { setForwards(next); setDirty(true) }} />
        <div className="flex justify-end">
          <SaveButton isDirty={dirty} saving={saving} onClick={save} onCancel={() => { setForwards(data ?? []); setDirty(false) }} />
        </div>
      </div>
    </div>
  )
}
