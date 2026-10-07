import type { DhcpLeaseView, TypesNeighbor } from '@/api'
import { AutocompleteInput, type AutocompleteOption } from '@/components/ui/AutocompleteInput'
import { api } from '@/lib/client'
import { useEffect, useMemo, useState } from 'react'

type KnownDeviceSelectShape = {
  value: string
  devices: KnownDevice[]
  disabled?: boolean
  onChange: (value: string) => void
}

export interface KnownDevice {
  address: string
  hostname?: string
  hardwareAddress?: string
  interface?: string
  source: 'DHCP' | 'Neighbor'
}

export function useKnownDevices(): KnownDevice[] {
  const [leases, setLeases] = useState<DhcpLeaseView[]>([])
  const [neighbors, setNeighbors] = useState<TypesNeighbor[]>([])

  useEffect(() => {
    api.apiDhcpLeasesGet().then((value) => setLeases(value ?? [])).catch(() => setLeases([]))
    api.apiRoutingNeighborsGet({}).then((value) => setNeighbors(value.neighbors ?? [])).catch(() => setNeighbors([]))
  }, [])

  return useMemo(() => mergeDevices(leases, neighbors), [leases, neighbors])
}

export function KnownDeviceSelect({ value, devices, disabled, onChange }: KnownDeviceSelectShape) {
  const options: AutocompleteOption[] = devices.map((device) => ({
    value: device.address,
    label: device.hostname || device.address,
    description: device.hostname
      ? `${device.address} · ${device.hardwareAddress || device.interface || device.source}`
      : device.hardwareAddress || device.interface || undefined,
    meta: device.source,
    keywords: [device.hostname, device.hardwareAddress, device.interface].filter((part): part is string => !!part),
  }))

  return (
    <AutocompleteInput
      value={value}
      options={options}
      disabled={disabled}
      className="font-mono"
      popupMinWidth={480}
      placeholder="192.168.1.10 or hostname"
      emptyMessage="No known device matches. You can enter an address manually."
      onChange={onChange}
    />
  )
}

function mergeDevices(leases: DhcpLeaseView[], neighbors: TypesNeighbor[]): KnownDevice[] {
  const byAddress = new Map<string, KnownDevice>()
  for (const neighbor of neighbors) {
    byAddress.set(neighbor.dst, {
      address: neighbor.dst,
      hardwareAddress: neighbor.lladdr,
      interface: neighbor.dev,
      source: 'Neighbor',
    })
  }
  for (const lease of leases) {
    const existing = byAddress.get(lease.ip_address)
    byAddress.set(lease.ip_address, {
      address: lease.ip_address,
      hostname: lease.hostname || existing?.hostname,
      hardwareAddress: lease.hw_address || existing?.hardwareAddress,
      interface: existing?.interface,
      source: 'DHCP',
    })
  }

  return [...byAddress.values()].sort((left, right) => {
    if (!!left.hostname !== !!right.hostname) return left.hostname ? -1 : 1
    return (left.hostname || left.address).localeCompare(right.hostname || right.address, undefined, { numeric: true })
  })
}
