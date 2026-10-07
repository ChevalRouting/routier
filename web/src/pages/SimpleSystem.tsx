import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import ConfigUsers from '@/pages/ConfigUsers'
import { Button, EntryRow, PageHeader, PreferencesGroup, Spinner } from 'cheval-ui'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

interface System {
  hostname: string
}

export default function SimpleSystem() {
  const { data, isLoading, reload } = useFetch<System>(() =>
    api.apiConfigSectionGet({ section: 'system' }) as Promise<System>,
  )
  const [hostname, setHostname] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (data) setHostname(data.hostname)
  }, [data])

  if (isLoading) return <Spinner />

  const save = async () => {
    setSaving(true)
    try {
      await api.apiConfigSectionPut({ section: 'system', body: { hostname } })
      toast.success('System configuration staged')
      reload()
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save system configuration')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader title="System" description="Basic identity for this router." />
      <div className="max-w-2xl space-y-4">
        <PreferencesGroup title="Identity">
          <EntryRow title="Router name" value={hostname} onChange={(event) => setHostname(event.target.value)} />
        </PreferencesGroup>
        <div className="flex justify-end"><Button onClick={save} disabled={saving || !hostname.trim()}>{saving ? 'Saving…' : 'Save'}</Button></div>
      </div>
      <section className="space-y-3"><h2 className="text-sm font-semibold">Users</h2><ConfigUsers embedded /></section>
    </div>
  )
}
