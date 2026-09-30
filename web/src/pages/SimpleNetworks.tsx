import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { PageHeader, SaveButton, Spinner } from 'cheval-ui'
import { api } from '@/lib/client'
import { useFetch } from '@/lib/useFetch'
import type { Network } from '@/components/simple/types'
import type { TypesSystemNic as SystemNic } from '@/api'
import { NetworkEditor } from '@/components/simple/NetworkEditor'
import { DhcpLeases } from '@/components/monitor/DhcpPanel'

export default function SimpleNetworks() {
  const { data, isLoading, reload } = useFetch<Network[]>(() =>
    api.apiConfigSectionGet({ section: 'networks' }) as Promise<Network[]>,
  )
  const { data: nics } = useFetch<SystemNic[]>(() => api.apiSystemNicsGet())
  const [networks, setNetworks] = useState<Network[]>([])
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (data) {
      setNetworks(data)
      setDirty(false)
    }
  }, [data])

  if (isLoading) return <Spinner />

  const save = async () => {
    setSaving(true)
    try {
      await api.apiConfigSectionPut({ section: 'networks', body: networks as unknown as object })
      toast.success('Local networks staged')
      setDirty(false)
      reload()
    } catch (err: unknown) {
      toast.error((err as Error).message || 'Failed to save local networks')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Local networks" description="Configure addresses Routier provides to connected devices." />
      <div className="max-w-3xl space-y-5">
        <NetworkEditor networks={networks} nics={nics ?? []} onChange={(next) => { setNetworks(next); setDirty(true) }} />
        <div className="flex justify-end">
          <SaveButton isDirty={dirty} saving={saving} onClick={save} onCancel={() => { setNetworks(data ?? []); setDirty(false) }} />
        </div>
      </div>

      {networks.some((n) => n.manage_dhcp) && (
        <div className="space-y-3">
          <div>
            <h2 className="text-sm font-medium">Connected devices</h2>
            <p className="text-sm text-muted-foreground">Addresses handed out to devices on your networks. Reserve one to always give a device the same address.</p>
          </div>
          <DhcpLeases />
        </div>
      )}
    </div>
  )
}
